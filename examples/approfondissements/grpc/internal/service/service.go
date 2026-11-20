// Package service implémente le service des tâches défini dans
// proto/taches/v1/taches.proto. Il ne dépend que du code généré : il ignore
// tout du transport (HTTP/1.1, HTTP/2, en mémoire) et du protocole (Connect,
// gRPC, gRPC-Web).
package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1/tachesv1connect"
)

// tailleTampon est le nombre de tâches qu'un abonné peut avoir en retard
// avant d'être déconnecté.
const tailleTampon = 16

// Service conserve les tâches en mémoire.
type Service struct {
	// Les méthodes non implémentées renvoient CodeUnimplemented : le code
	// compile toujours quand le contrat gagne une méthode.
	tachesv1connect.UnimplementedTacheServiceHandler

	mu      sync.Mutex
	taches  []*tachesv1.Tache
	abonnes map[chan *tachesv1.Tache]struct{}

	Maintenant func() time.Time // remplaçable dans les tests
}

// Nouveau crée un service vide.
func Nouveau() *Service {
	return &Service{abonnes: map[chan *tachesv1.Tache]struct{}{}, Maintenant: time.Now}
}

// Enregistre ajoute les méthodes du service au serveur.
func (s *Service) Enregistre(serveur *connect.Server) {
	tachesv1connect.RegisterTacheServiceHandler(serveur, s)
}

// Cree enregistre une tâche.
func (s *Service) Cree(_ context.Context, req *tachesv1.CreeRequest) (*tachesv1.CreeResponse, error) {
	t, err := s.ajoute(req.GetTitre())
	if err != nil {
		return nil, err
	}
	return &tachesv1.CreeResponse{Tache: t}, nil
}

// Trouve renvoie une tâche, ou l'erreur NOT_FOUND.
func (s *Service) Trouve(_ context.Context, req *tachesv1.TrouveRequest) (*tachesv1.TrouveResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.taches {
		if t.GetId() == req.GetId() {
			return &tachesv1.TrouveResponse{Tache: t}, nil
		}
	}
	// Seul un *connect.Error envoie son message au client.
	return nil, connect.Errorf(connect.CodeNotFound, "tâche %d introuvable", req.GetId())
}

// Suit envoie les tâches existantes, puis chaque nouvelle tâche, jusqu'à ce
// que le client se déconnecte.
func (s *Service) Suit(ctx context.Context, _ *tachesv1.SuitRequest, flux tachesv1connect.TacheServiceSuitServerStream) error {
	// La copie des tâches existantes et l'abonnement se font sous le même
	// verrou : aucune tâche ne peut être manquée ni envoyée deux fois.
	s.mu.Lock()
	existantes := append([]*tachesv1.Tache(nil), s.taches...)
	nouvelles := make(chan *tachesv1.Tache, tailleTampon)
	s.abonnes[nouvelles] = struct{}{}
	s.mu.Unlock()
	defer s.desabonne(nouvelles)

	for _, t := range existantes {
		if err := flux.Send(&tachesv1.SuitResponse{Tache: t}); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done(): // le client s'est déconnecté
			return ctx.Err()
		case t, ok := <-nouvelles:
			if !ok {
				return connect.NewError(connect.CodeResourceExhausted, "abonné trop lent, déconnecté")
			}
			if err := flux.Send(&tachesv1.SuitResponse{Tache: t}); err != nil {
				return err
			}
		}
	}
}

// Importe reçoit des titres jusqu'à la fin du flux, puis crée les tâches.
// L'import est atomique : un seul titre invalide, et aucune tâche n'est créée.
func (s *Service) Importe(_ context.Context, flux tachesv1connect.TacheServiceImporteServerStream) (*tachesv1.ImporteResponse, error) {
	var titres []string
	for {
		req, err := flux.Receive()
		if errors.Is(err, io.EOF) { // le client a fini d'envoyer
			break
		}
		if err != nil {
			return nil, err
		}
		titre, err := nettoie(req.GetTitre())
		if err != nil {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "titre n° %d : le titre est obligatoire", len(titres)+1)
		}
		titres = append(titres, titre)
	}
	for _, titre := range titres {
		if _, err := s.ajoute(titre); err != nil {
			return nil, err
		}
	}
	return &tachesv1.ImporteResponse{Nombre: int64(len(titres))}, nil
}

// nettoie valide un titre et retire les espaces qui l'entourent.
func nettoie(titre string) (string, error) {
	titre = strings.TrimSpace(titre)
	if titre == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, "le titre est obligatoire")
	}
	return titre, nil
}

// ajoute crée la tâche et la diffuse aux abonnés.
func (s *Service) ajoute(titre string) (*tachesv1.Tache, error) {
	titre, err := nettoie(titre)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &tachesv1.Tache{
		Id:       int64(len(s.taches) + 1),
		Titre:    titre,
		Creation: timestamppb.New(s.Maintenant()),
	}
	s.taches = append(s.taches, t)
	for abonne := range s.abonnes {
		select {
		case abonne <- t:
		default:
			// Un abonné trop lent ne doit pas bloquer les créations : il est
			// déconnecté.
			close(abonne)
			delete(s.abonnes, abonne)
		}
	}
	return t, nil
}

func (s *Service) desabonne(abonne chan *tachesv1.Tache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.abonnes[abonne]; ok {
		delete(s.abonnes, abonne)
		close(abonne)
	}
}
