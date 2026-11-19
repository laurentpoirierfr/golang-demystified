package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/domaine"
)

// Service regroupe les cas d'usage. Il ne dépend que du domaine et des ports.
type Service struct {
	catalogue    Catalogue
	emprunts     Emprunts
	notifieur    Notifieur
	horloge      Horloge
	identifiants Identifiants
	tx           Transactions
	log          *slog.Logger
}

// Dependances regroupe les adaptateurs à brancher sur les ports.
type Dependances struct {
	Catalogue    Catalogue
	Emprunts     Emprunts
	Notifieur    Notifieur
	Horloge      Horloge
	Identifiants Identifiants
	Transactions Transactions
	Journal      *slog.Logger
}

// NouveauService assemble le service.
func NouveauService(d Dependances) *Service {
	return &Service{
		catalogue: d.Catalogue, emprunts: d.Emprunts, notifieur: d.Notifieur,
		horloge: d.Horloge, identifiants: d.Identifiants, tx: d.Transactions, log: d.Journal,
	}
}

// Emprunte enregistre l'emprunt d'un livre par un membre.
func (s *Service) Emprunte(ctx context.Context, membreID, isbn string) (domaine.Emprunt, error) {
	var e domaine.Emprunt
	err := s.tx.Atomique(ctx, func(ctx context.Context) error {
		m, err := s.catalogue.Membre(ctx, membreID)
		if err != nil {
			return err
		}
		if _, err := s.catalogue.Livre(ctx, isbn); err != nil {
			return err
		}
		pris, err := s.emprunts.EnCoursPourLivre(ctx, isbn)
		if err != nil {
			return err
		}
		enCours, err := s.emprunts.EnCoursPourMembre(ctx, membreID)
		if err != nil {
			return err
		}
		if err := domaine.PeutEmprunter(m, len(enCours), !pris); err != nil {
			return err
		}
		e = domaine.NouvelEmprunt(s.identifiants.Nouveau(), isbn, membreID, s.horloge.Maintenant())
		return s.emprunts.Enregistre(ctx, e)
	})
	if err != nil {
		return domaine.Emprunt{}, fmt.Errorf("emprunt de %s par %s : %w", isbn, membreID, err)
	}
	return e, nil
}

// Rend clôt un emprunt, et ajoute l'éventuelle pénalité au compte du membre.
func (s *Service) Rend(ctx context.Context, empruntID string) (penalite int, err error) {
	err = s.tx.Atomique(ctx, func(ctx context.Context) error {
		e, err := s.emprunts.Emprunt(ctx, empruntID)
		if err != nil {
			return err
		}
		penalite, err = e.Rend(s.horloge.Maintenant())
		if err != nil {
			return err
		}
		if err := s.emprunts.Enregistre(ctx, e); err != nil {
			return err
		}
		if penalite == 0 {
			return nil
		}
		m, err := s.catalogue.Membre(ctx, e.Membre)
		if err != nil {
			return err
		}
		m.Penalites += penalite
		return s.catalogue.EnregistreMembre(ctx, m)
	})
	if err != nil {
		return 0, fmt.Errorf("retour de %s : %w", empruntID, err)
	}
	return penalite, nil
}

// RelanceRetards notifie chaque membre en retard, et renvoie le nombre de
// relances envoyées. Un échec de notification n'arrête pas les autres.
func (s *Service) RelanceRetards(ctx context.Context) (int, error) {
	enCours, err := s.emprunts.EnCours(ctx)
	if err != nil {
		return 0, err
	}
	maintenant := s.horloge.Maintenant()
	relances := 0
	for _, e := range enCours {
		if !e.EnRetard(maintenant) {
			continue
		}
		m, err := s.catalogue.Membre(ctx, e.Membre)
		if err != nil {
			return relances, err
		}
		livre, err := s.catalogue.Livre(ctx, e.ISBN)
		if err != nil {
			return relances, err
		}
		message := fmt.Sprintf("« %s » devait être rendu le %s.", livre.Titre, e.Echeance.Format("02/01/2006"))
		if err := s.notifieur.Notifie(ctx, m, message); err != nil {
			s.log.WarnContext(ctx, "relance impossible", "membre", m.ID, "erreur", err)
			continue
		}
		relances++
	}
	return relances, nil
}
