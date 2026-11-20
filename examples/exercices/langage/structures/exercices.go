// Package structures contient les solutions des exercices du chapitre
// « Structures et méthodes ».
package structures

import (
	"errors"
	"fmt"
)

// Exercice 1 ★ : des méthodes.

// Rectangle est défini par sa largeur et sa hauteur.
type Rectangle struct {
	Largeur, Hauteur float64
}

// Aire et Perimetre n'ont pas besoin de modifier le rectangle : récepteur
// par valeur.
func (r Rectangle) Aire() float64      { return r.Largeur * r.Hauteur }
func (r Rectangle) Perimetre() float64 { return 2 * (r.Largeur + r.Hauteur) }

// Exercice 2 ★★ : une pile, et un récepteur pointeur.

// Pile est une pile d'entiers. Sa valeur zéro est une pile vide, prête à
// servir.
type Pile struct {
	elements []int
}

// Empile ajoute v au sommet. Elle modifie la pile : récepteur pointeur.
func (p *Pile) Empile(v int) { p.elements = append(p.elements, v) }

// Depile retire et renvoie le sommet, ou faux si la pile est vide.
func (p *Pile) Depile() (int, bool) {
	if len(p.elements) == 0 {
		return 0, false
	}
	v := p.elements[len(p.elements)-1]
	p.elements = p.elements[:len(p.elements)-1]
	return v, true
}

// Taille renvoie le nombre d'éléments.
func (p *Pile) Taille() int { return len(p.elements) }

// Exercice 3 ★★★ : l'intégration.

// Historique garde une trace des opérations.
type Historique struct {
	lignes []string
}

// Note ajoute une ligne à l'historique.
func (h *Historique) Note(format string, args ...any) {
	h.lignes = append(h.lignes, fmt.Sprintf(format, args...))
}

// Lignes renvoie l'historique.
func (h *Historique) Lignes() []string { return h.lignes }

// Compte est un compte bancaire, qui intègre un Historique : ses méthodes
// (Note, Lignes) sont promues sur Compte.
type Compte struct {
	Historique
	solde int
}

// ErrSoldeInsuffisant signale un retrait impossible.
var ErrSoldeInsuffisant = errors.New("solde insuffisant")

// Depose ajoute un montant positif.
func (c *Compte) Depose(montant int) {
	c.solde += montant
	c.Note("dépôt de %d", montant)
}

// Retire enlève un montant, si le solde le permet.
func (c *Compte) Retire(montant int) error {
	if montant > c.solde {
		c.Note("retrait de %d refusé", montant)
		return ErrSoldeInsuffisant
	}
	c.solde -= montant
	c.Note("retrait de %d", montant)
	return nil
}

// Solde renvoie le solde.
func (c *Compte) Solde() int { return c.solde }
