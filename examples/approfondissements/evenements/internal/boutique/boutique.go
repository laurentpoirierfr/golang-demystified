// Package boutique contient la logique métier : passer une commande, réserver
// le stock, confirmer ou annuler, notifier le client. Chaque étape réagit à un
// événement et en publie un autre : c'est une saga, sans orchestrateur
// central.
//
//	Passe ──▶ CommandePassee ──▶ réservation ──▶ StockReserve ─────▶ confirmation, notification
//	                                         └─▶ StockInsuffisant ─▶ annulation, notification
//
// Le package ignore tout du transport (mémoire, NATS…) et du stockage
// (mémoire, PostgreSQL) : il ne connaît que les interfaces Depot et Notifieur.
package boutique

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

// Statut d'une commande.
type Statut string

const (
	EnAttente Statut = "en_attente"
	Confirmee Statut = "confirmee"
	Annulee   Statut = "annulee"
)

// Commande est l'état d'une commande.
type Commande struct {
	ID       string              `json:"id"`
	Client   string              `json:"client"`
	Articles []evenement.Article `json:"articles"`
	Statut   Statut              `json:"statut"`
	Raison   string              `json:"raison,omitempty"`
}

// ErrIntrouvable signale une commande inconnue.
var ErrIntrouvable = errors.New("commande introuvable")

// Unite est une unité de travail : ses modifications et les événements
// qu'elle publie sont validés ensemble, ou abandonnés ensemble.
type Unite interface {
	CreeCommande(ctx context.Context, c Commande) error
	ChangeStatut(ctx context.Context, id string, s Statut, raison string) error
	// Reserve réserve tous les articles, ou aucun : elle renvoie la référence
	// du premier article manquant, sans rien modifier.
	Reserve(ctx context.Context, articles []evenement.Article) (manquant string, err error)
	// Publie publie un événement, qui ne part que si l'unité est validée.
	Publie(ctx context.Context, evt any) error
}

// Depot stocke l'état de la boutique.
type Depot interface {
	// Execute exécute f dans une unité de travail, validée si f réussit. Si
	// cle n'est pas vide, f n'est exécutée qu'une seule fois pour cette clé :
	// un message reçu deux fois n'est traité qu'une fois (idempotence).
	Execute(ctx context.Context, cle string, f func(ctx context.Context, u Unite) error) error
	Commande(ctx context.Context, id string) (Commande, error)
	Stock(ctx context.Context, ref string) (int, error)
}

// Notifieur prévient le client. C'est un effet de bord externe (courriel,
// SMS) : il ne peut pas faire partie de l'unité de travail.
type Notifieur interface {
	// Notifie envoie le message. cle identifie la notification : un
	// fournisseur sérieux s'en sert pour ignorer les doublons.
	Notifie(ctx context.Context, cle, client, message string) error
}

// Service implémente la boutique.
type Service struct {
	depot     Depot
	notifieur Notifieur
	NouvelID  func() string // remplaçable dans les tests
}

// Nouveau crée le service.
func Nouveau(depot Depot, notifieur Notifieur, nouvelID func() string) *Service {
	return &Service{depot: depot, notifieur: notifieur, NouvelID: nouvelID}
}

// Passe enregistre une commande en attente et publie CommandePassee, dans la
// même unité de travail : soit les deux ont lieu, soit aucun.
func (s *Service) Passe(ctx context.Context, client string, articles []evenement.Article) (string, error) {
	client = strings.TrimSpace(client)
	if client == "" || len(articles) == 0 {
		return "", errors.New("client et articles obligatoires")
	}
	for _, a := range articles {
		if a.Ref == "" || a.Quantite <= 0 {
			return "", fmt.Errorf("article invalide : %+v", a)
		}
	}
	c := Commande{ID: s.NouvelID(), Client: client, Articles: articles, Statut: EnAttente}
	err := s.depot.Execute(ctx, "", func(ctx context.Context, u Unite) error {
		if err := u.CreeCommande(ctx, c); err != nil {
			return err
		}
		return u.Publie(ctx, &evenement.CommandePassee{CommandeID: c.ID, Client: c.Client, Articles: c.Articles})
	})
	return c.ID, err
}

// Commande renvoie l'état d'une commande.
func (s *Service) Commande(ctx context.Context, id string) (Commande, error) {
	return s.depot.Commande(ctx, id)
}

// Gestionnaires renvoie les gestionnaires d'événements de la saga. Chacun a
// un nom stable : il identifie son abonnement (groupe de consommateurs) et
// ses clés d'idempotence.
func (s *Service) Gestionnaires() []cqrs.EventHandler {
	return []cqrs.EventHandler{
		cqrs.NewEventHandler("stock.reserve", s.reserve),
		cqrs.NewEventHandler("commandes.confirme", s.confirme),
		cqrs.NewEventHandler("commandes.annule", s.annule),
		cqrs.NewEventHandler("notifications.confirmation", s.notifieConfirmation),
		cqrs.NewEventHandler("notifications.annulation", s.notifieAnnulation),
	}
}

// cle identifie le traitement d'un message par un gestionnaire : le même
// message, livré deux fois au même gestionnaire, a la même clé.
func cle(ctx context.Context, gestionnaire string) string {
	return gestionnaire + "/" + cqrs.OriginalMessageFromCtx(ctx).UUID
}

func (s *Service) reserve(ctx context.Context, e *evenement.CommandePassee) error {
	return s.depot.Execute(ctx, cle(ctx, "stock.reserve"), func(ctx context.Context, u Unite) error {
		manquant, err := u.Reserve(ctx, e.Articles)
		if err != nil {
			return err
		}
		if manquant != "" {
			return u.Publie(ctx, &evenement.StockInsuffisant{CommandeID: e.CommandeID, Ref: manquant})
		}
		return u.Publie(ctx, &evenement.StockReserve{CommandeID: e.CommandeID})
	})
}

func (s *Service) confirme(ctx context.Context, e *evenement.StockReserve) error {
	return s.depot.Execute(ctx, cle(ctx, "commandes.confirme"), func(ctx context.Context, u Unite) error {
		return u.ChangeStatut(ctx, e.CommandeID, Confirmee, "")
	})
}

// annule est la compensation : la commande ne peut pas être servie.
func (s *Service) annule(ctx context.Context, e *evenement.StockInsuffisant) error {
	return s.depot.Execute(ctx, cle(ctx, "commandes.annule"), func(ctx context.Context, u Unite) error {
		return u.ChangeStatut(ctx, e.CommandeID, Annulee, "stock insuffisant : "+e.Ref)
	})
}

func (s *Service) notifieConfirmation(ctx context.Context, e *evenement.StockReserve) error {
	c, err := s.depot.Commande(ctx, e.CommandeID)
	if err != nil {
		return err
	}
	return s.notifieur.Notifie(ctx, cle(ctx, "notifications.confirmation"), c.Client,
		"Votre commande "+c.ID+" est confirmée.")
}

func (s *Service) notifieAnnulation(ctx context.Context, e *evenement.StockInsuffisant) error {
	c, err := s.depot.Commande(ctx, e.CommandeID)
	if err != nil {
		return err
	}
	return s.notifieur.Notifie(ctx, cle(ctx, "notifications.annulation"), c.Client,
		"Votre commande "+c.ID+" est annulée : l'article "+e.Ref+" manque.")
}
