// Package memoire implémente le dépôt de la boutique en mémoire, pour les
// tests et la démonstration. Une unité de travail prépare ses modifications à
// part, et ne les applique qu'à la validation, comme une transaction.
package memoire

import (
	"context"
	"maps"
	"sync"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/boutique"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

// Depot est un dépôt en mémoire.
type Depot struct {
	bus *cqrs.EventBus

	mu        sync.Mutex
	commandes map[string]boutique.Commande
	stock     map[string]int
	traites   map[string]bool // clés d'idempotence déjà traitées
}

// Nouveau crée un dépôt avec le stock initial donné. Les événements sont
// publiés sur bus.
func Nouveau(bus *cqrs.EventBus, stock map[string]int) *Depot {
	return &Depot{bus: bus, commandes: map[string]boutique.Commande{}, stock: maps.Clone(stock), traites: map[string]bool{}}
}

// Execute exécute f dans une unité de travail.
//
// Les événements ne sont publiés qu'après la validation, hors du verrou. Ce
// dépôt n'a pas d'outbox : si le programme s'arrêtait entre la validation et
// la publication, les événements seraient perdus. C'est acceptable pour des
// tests ; en production, le dépôt PostgreSQL utilise une outbox.
func (d *Depot) Execute(ctx context.Context, cle string, f func(context.Context, boutique.Unite) error) error {
	d.mu.Lock()
	if cle != "" && d.traites[cle] {
		d.mu.Unlock()
		return nil // déjà traité : rien à faire
	}
	u := &unite{depot: d, commandes: map[string]boutique.Commande{}, stock: map[string]int{}}
	if err := f(ctx, u); err != nil {
		d.mu.Unlock()
		return err // rien n'est appliqué
	}
	maps.Copy(d.commandes, u.commandes)
	maps.Copy(d.stock, u.stock)
	if cle != "" {
		d.traites[cle] = true
	}
	d.mu.Unlock()

	for _, evt := range u.evenements {
		if err := d.bus.Publish(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

// Commande renvoie une commande.
func (d *Depot) Commande(_ context.Context, id string) (boutique.Commande, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.commandes[id]
	if !ok {
		return boutique.Commande{}, boutique.ErrIntrouvable
	}
	return c, nil
}

// Stock renvoie le stock d'un article.
func (d *Depot) Stock(_ context.Context, ref string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stock[ref], nil
}

// unite prépare les modifications ; elle s'exécute sous le verrou du dépôt.
type unite struct {
	depot      *Depot
	commandes  map[string]boutique.Commande
	stock      map[string]int
	evenements []any
}

func (u *unite) commande(id string) (boutique.Commande, bool) {
	if c, ok := u.commandes[id]; ok {
		return c, true
	}
	c, ok := u.depot.commandes[id]
	return c, ok
}

func (u *unite) CreeCommande(_ context.Context, c boutique.Commande) error {
	u.commandes[c.ID] = c
	return nil
}

func (u *unite) ChangeStatut(_ context.Context, id string, s boutique.Statut, raison string) error {
	c, ok := u.commande(id)
	if !ok {
		return boutique.ErrIntrouvable
	}
	c.Statut, c.Raison = s, raison
	u.commandes[id] = c
	return nil
}

func (u *unite) Reserve(_ context.Context, articles []evenement.Article) (string, error) {
	// Vérifier tout, puis modifier : tout ou rien.
	besoin := map[string]int{}
	for _, a := range articles {
		besoin[a.Ref] += a.Quantite
	}
	for _, a := range articles {
		if u.depot.stock[a.Ref] < besoin[a.Ref] {
			return a.Ref, nil
		}
	}
	for ref, n := range besoin {
		u.stock[ref] = u.depot.stock[ref] - n
	}
	return "", nil
}

func (u *unite) Publie(_ context.Context, evt any) error {
	u.evenements = append(u.evenements, evt)
	return nil
}
