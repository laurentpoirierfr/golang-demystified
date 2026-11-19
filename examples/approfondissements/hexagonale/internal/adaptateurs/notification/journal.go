// Package notification est un adaptateur secondaire (« driven ») qui
// implémente le port Notifieur de l'application.
package notification

import (
	"context"
	"log/slog"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/domaine"
)

// Journal « envoie » les notifications dans le journal. Un adaptateur SMTP
// ou SMS implémenterait le même port sans que l'application change.
type Journal struct{ Log *slog.Logger }

func (j Journal) Notifie(ctx context.Context, m domaine.Membre, message string) error {
	j.Log.InfoContext(ctx, "notification", "destinataire", m.Email, "message", message)
	return nil
}
