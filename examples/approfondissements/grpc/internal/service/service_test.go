package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connectinprocess"

	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1/tachesv1connect"
)

// client relie un client au service dans le même processus, sans réseau :
// connectinprocess remplace le serveur HTTP de test.
func client(t *testing.T, intercepteurs ...connect.ServerInterceptor) (tachesv1connect.TacheServiceClient, *Service) {
	t.Helper()
	s := Nouveau()
	s.Maintenant = func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }
	serveur := connect.NewServer(intercepteurs...)
	s.Enregistre(serveur)
	return tachesv1connect.NewTacheServiceClient(connect.NewClient(connectinprocess.New(serveur))), s
}

func TestCreeEtTrouve(t *testing.T) {
	c, _ := client(t)
	cree, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "  Apprendre Connect  "})
	if err != nil {
		t.Fatal(err)
	}
	if cree.GetTache().GetId() != 1 || cree.GetTache().GetTitre() != "Apprendre Connect" {
		t.Errorf("tâche créée : %v", cree.GetTache())
	}

	trouve, err := c.Trouve(t.Context(), &tachesv1.TrouveRequest{Id: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := trouve.GetTache().GetCreation().AsTime(); !got.Equal(time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("date de création : %v", got)
	}
}

func TestErreurs(t *testing.T) {
	c, _ := client(t)
	tests := []struct {
		nom     string
		appel   func() error
		code    connect.Code
		message string
	}{
		{"titre vide", func() error {
			_, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: " "})
			return err
		}, connect.CodeInvalidArgument, "le titre est obligatoire"},
		{"tâche introuvable", func() error {
			_, err := c.Trouve(t.Context(), &tachesv1.TrouveRequest{Id: 42})
			return err
		}, connect.CodeNotFound, "tâche 42 introuvable"},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			err := tt.appel()
			if connect.CodeOf(err) != tt.code {
				t.Errorf("code = %v, attendu %v", connect.CodeOf(err), tt.code)
			}
			// errors.AsType (Go 1.26) : la version générique de errors.As.
			if e, ok := errors.AsType[*connect.Error](err); !ok || e.Message() != tt.message {
				t.Errorf("erreur = %v, message attendu %q", err, tt.message)
			}
		})
	}
}

func TestImporte(t *testing.T) {
	c, _ := client(t)
	flux, err := c.Importe(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, titre := range []string{"Lire", "Écrire", "Compter"} {
		if err := flux.Send(&tachesv1.ImporteRequest{Titre: titre}); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := flux.CloseAndReceive()
	if err != nil {
		t.Fatal(err)
	}
	if rep.GetNombre() != 3 {
		t.Errorf("nombre = %d, attendu 3", rep.GetNombre())
	}
}

func TestSuit(t *testing.T) {
	// synctest : le temps est virtuel, et synctest.Wait attend que toutes les
	// goroutines de la bulle soient bloquées. Pas de time.Sleep fragile.
	synctest.Test(t, func(t *testing.T) {
		c, _ := client(t)
		if _, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Avant"}); err != nil {
			t.Fatal(err)
		}

		ctx, arrete := context.WithCancel(t.Context())
		flux, err := c.Suit(ctx, &tachesv1.SuitRequest{})
		if err != nil {
			t.Fatal(err)
		}
		recu := make(chan string)
		go func() {
			defer close(recu)
			for {
				rep, err := flux.Receive()
				if err != nil {
					return
				}
				recu <- rep.GetTache().GetTitre()
			}
		}()

		if titre := <-recu; titre != "Avant" {
			t.Errorf("première tâche = %q, attendu « Avant »", titre)
		}
		synctest.Wait() // l'abonnement est en place
		if _, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Après"}); err != nil {
			t.Fatal(err)
		}
		if titre := <-recu; titre != "Après" {
			t.Errorf("tâche suivante = %q, attendu « Après »", titre)
		}

		arrete() // le client se déconnecte : le flux se termine
		if _, ok := <-recu; ok {
			t.Error("le flux aurait dû se terminer")
		}
	})
}

func TestAbonneTropLent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, s := client(t)
		if _, err := s.ajoute("première"); err != nil {
			t.Fatal(err)
		}
		flux, err := c.Suit(t.Context(), &tachesv1.SuitRequest{})
		if err != nil {
			t.Fatal(err)
		}
		// Le service ne démarre qu'au premier Receive : une fois la première
		// tâche reçue, l'abonnement est en place.
		if _, err := flux.Receive(); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		// L'abonné ne lit rien : au-delà du tampon, il est déconnecté, et les
		// créations ne sont jamais bloquées. Le service peut avoir retiré une
		// tâche du tampon, bloqué sur Send : il en faut tailleTampon+2 pour
		// déborder à coup sûr.
		for range tailleTampon + 2 {
			if _, err := s.ajoute("tâche"); err != nil {
				t.Fatal(err)
			}
		}
		var err2 error
		for err2 == nil {
			_, err2 = flux.Receive()
		}
		if connect.CodeOf(err2) != connect.CodeResourceExhausted {
			t.Errorf("code = %v, attendu resource_exhausted", connect.CodeOf(err2))
		}
	})
}

func TestIntercepteurs(t *testing.T) {
	var journal bytes.Buffer
	log := slog.New(slog.NewTextHandler(&journal, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == "duree" {
				return slog.Attr{} // valeurs variables, retirées du journal
			}
			return a
		},
	}))
	c, _ := client(t, Journal(log), Authentification("secret"))

	// Sans jeton : refusé avant d'atteindre le service.
	_, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Refusée"})
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("sans jeton : code = %v", connect.CodeOf(err))
	}

	// Avec le bon jeton, ajouté par l'intercepteur du client.
	s := Nouveau()
	serveur := connect.NewServer(Journal(log), Authentification("secret"))
	s.Enregistre(serveur)
	authentifie := tachesv1connect.NewTacheServiceClient(
		connect.NewClient(connectinprocess.New(serveur), Jeton("secret")))
	if _, err := authentifie.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Acceptée"}); err != nil {
		t.Fatalf("avec jeton : %v", err)
	}

	lignes := strings.Split(strings.TrimSpace(journal.String()), "\n")
	attendu := []string{
		`level=WARN msg=appel procedure=/taches.v1.TacheService/Cree code=unauthenticated`,
		`level=INFO msg=appel procedure=/taches.v1.TacheService/Cree code=ok`,
	}
	if strings.Join(lignes, "\n") != strings.Join(attendu, "\n") {
		t.Errorf("journal :\n%s\nattendu :\n%s", journal.String(), strings.Join(attendu, "\n"))
	}
}

func TestRecuperation(t *testing.T) {
	serveur := connect.NewServer(Recuperation(slog.New(slog.NewTextHandler(io.Discard, nil))))
	tachesv1connect.RegisterTacheServiceHandler(serveur, paniqueur{})
	c := tachesv1connect.NewTacheServiceClient(connect.NewClient(connectinprocess.New(serveur)))

	_, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Boum"})
	if e, ok := errors.AsType[*connect.Error](err); !ok || e.Code() != connect.CodeInternal || e.Message() != "erreur interne" {
		t.Errorf("erreur = %v", err)
	}
}

type paniqueur struct {
	tachesv1connect.UnimplementedTacheServiceHandler
}

func (paniqueur) Cree(context.Context, *tachesv1.CreeRequest) (*tachesv1.CreeResponse, error) {
	panic("détail interne à ne pas divulguer")
}

func TestImporteAtomique(t *testing.T) {
	c, _ := client(t)
	flux, err := c.Importe(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, titre := range []string{"Lire", " ", "Compter"} {
		if err := flux.Send(&tachesv1.ImporteRequest{Titre: titre}); err != nil {
			break // le serveur a répondu : l'erreur vient avec CloseAndReceive
		}
	}
	if _, err := flux.CloseAndReceive(); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, attendu invalid_argument", connect.CodeOf(err))
	}
	// Aucune tâche créée, pas même la première.
	if _, err := c.Trouve(t.Context(), &tachesv1.TrouveRequest{Id: 1}); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("la tâche « Lire » n'aurait pas dû être créée")
	}
}
