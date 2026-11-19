// Package notification regroupe des adaptateurs secondaires (« driven ») qui
// implémentent les ports techniques de l'application.
package notification

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/domaine"
)

// Journal « envoie » les notifications dans le journal. Un adaptateur SMTP
// ou SMS implémenterait le même port sans que l'application change.
type Journal struct{ Log *slog.Logger }

func (j Journal) Notifie(ctx context.Context, m domaine.Membre, message string) error {
	j.Log.InfoContext(ctx, "notification", "destinataire", m.Email, "message", message)
	return nil
}

// HorlogeSysteme donne l'heure réelle.
type HorlogeSysteme struct{}

func (HorlogeSysteme) Maintenant() time.Time { return time.Now() }

// UUID produit des identifiants UUID v7.
type UUID struct{}

func (UUID) Nouveau() string { return uuid.NewV7().String() }
