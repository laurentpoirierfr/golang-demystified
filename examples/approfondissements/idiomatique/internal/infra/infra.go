// Package infra contient les implémentations concrètes des dépendances du
// service d'inscription : un dépôt en mémoire et un courrier qui journalise.
// Elles satisfont les interfaces d'inscription sans les connaître.
package infra

import (
	"context"
	"log/slog"
	"sync"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
)

// DepotMemoire stocke les membres en mémoire.
type DepotMemoire struct {
	mu      sync.Mutex
	membres map[string]inscription.Membre
}

// NouveauDepotMemoire crée un dépôt vide.
func NouveauDepotMemoire() *DepotMemoire {
	return &DepotMemoire{membres: map[string]inscription.Membre{}}
}

func (d *DepotMemoire) Existe(_ context.Context, courriel string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.membres[courriel]
	return ok, nil
}

func (d *DepotMemoire) Enregistre(_ context.Context, m inscription.Membre) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.membres[m.Courriel] = m
	return nil
}

// CourrierJournal « envoie » les messages dans le journal.
type CourrierJournal struct {
	log *slog.Logger
}

// NouveauCourrierJournal crée un courrier qui écrit dans log.
func NouveauCourrierJournal(log *slog.Logger) *CourrierJournal {
	return &CourrierJournal{log: log}
}

func (c *CourrierJournal) Envoie(ctx context.Context, a, sujet, _ string) error {
	c.log.InfoContext(ctx, "courriel envoyé", "a", a, "sujet", sujet)
	return nil
}
