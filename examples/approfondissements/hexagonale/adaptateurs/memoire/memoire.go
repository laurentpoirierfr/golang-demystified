// Package memoire est un adaptateur de stockage en mémoire. Il implémente les
// ports Catalogue, Emprunts et Transactions. Il sert aux tests, aux
// démonstrations, et à démarrer un projet avant de choisir une base.
package memoire

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/domaine"
)

// Depot stocke tout en mémoire, protégé par un verrou.
type Depot struct {
	mu       sync.Mutex
	livres   map[string]domaine.Livre
	membres  map[string]domaine.Membre
	emprunts map[string]domaine.Emprunt
}

// Nouveau crée un dépôt contenant les livres et membres donnés.
func Nouveau(livres []domaine.Livre, membres []domaine.Membre) *Depot {
	d := &Depot{
		livres:   make(map[string]domaine.Livre),
		membres:  make(map[string]domaine.Membre),
		emprunts: make(map[string]domaine.Emprunt),
	}
	for _, l := range livres {
		d.livres[l.ISBN] = l
	}
	for _, m := range membres {
		d.membres[m.ID] = m
	}
	return d
}

// Atomique exécute f en détenant le verrou du dépôt : les cas d'usage
// s'exécutent l'un après l'autre, comme dans une transaction sérialisable.
func (d *Depot) Atomique(ctx context.Context, f func(context.Context) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return f(ctx)
}

// Les méthodes suivantes sont appelées à l'intérieur d'Atomique : elles ne
// prennent pas le verrou elles-mêmes.

func (d *Depot) Livre(_ context.Context, isbn string) (domaine.Livre, error) {
	l, ok := d.livres[isbn]
	if !ok {
		return domaine.Livre{}, domaine.ErrIntrouvable
	}
	return l, nil
}

func (d *Depot) Membre(_ context.Context, id string) (domaine.Membre, error) {
	m, ok := d.membres[id]
	if !ok {
		return domaine.Membre{}, domaine.ErrIntrouvable
	}
	return m, nil
}

func (d *Depot) EnregistreMembre(_ context.Context, m domaine.Membre) error {
	d.membres[m.ID] = m
	return nil
}

func (d *Depot) Emprunt(_ context.Context, id string) (domaine.Emprunt, error) {
	e, ok := d.emprunts[id]
	if !ok {
		return domaine.Emprunt{}, domaine.ErrIntrouvable
	}
	return e, nil
}

func (d *Depot) EnCoursPourLivre(_ context.Context, isbn string) (bool, error) {
	for _, e := range d.emprunts {
		if e.ISBN == isbn && e.EnCours() {
			return true, nil
		}
	}
	return false, nil
}

func (d *Depot) EnCoursPourMembre(_ context.Context, membre string) ([]domaine.Emprunt, error) {
	var res []domaine.Emprunt
	for _, e := range d.emprunts {
		if e.Membre == membre && e.EnCours() {
			res = append(res, e)
		}
	}
	return res, nil
}

func (d *Depot) EnCours(_ context.Context) ([]domaine.Emprunt, error) {
	var res []domaine.Emprunt
	for _, e := range d.emprunts {
		if e.EnCours() {
			res = append(res, e)
		}
	}
	slices.SortFunc(res, func(a, b domaine.Emprunt) int { return strings.Compare(a.ID, b.ID) })
	return res, nil
}

func (d *Depot) Enregistre(_ context.Context, e domaine.Emprunt) error {
	d.emprunts[e.ID] = e
	return nil
}
