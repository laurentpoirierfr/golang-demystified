package structures

import (
	"errors"
	"slices"
	"testing"
)

func TestRectangle(t *testing.T) {
	r := Rectangle{Largeur: 3, Hauteur: 4}
	if r.Aire() != 12 || r.Perimetre() != 14 {
		t.Errorf("aire %v, périmètre %v", r.Aire(), r.Perimetre())
	}
}

func TestPile(t *testing.T) {
	var p Pile // la valeur zéro est utilisable
	if _, ok := p.Depile(); ok {
		t.Error("une pile vide n'a pas de sommet")
	}
	for _, v := range []int{1, 2, 3} {
		p.Empile(v)
	}
	if v, _ := p.Depile(); v != 3 || p.Taille() != 2 {
		t.Errorf("sommet %d, taille %d", v, p.Taille())
	}
}

func TestCompte(t *testing.T) {
	var c Compte
	c.Depose(100)
	if err := c.Retire(30); err != nil {
		t.Fatal(err)
	}
	if err := c.Retire(500); !errors.Is(err, ErrSoldeInsuffisant) {
		t.Errorf("err = %v", err)
	}
	if c.Solde() != 70 {
		t.Errorf("solde %d", c.Solde())
	}
	// Lignes est une méthode d'Historique, promue sur Compte.
	attendu := []string{"dépôt de 100", "retrait de 30", "retrait de 500 refusé"}
	if !slices.Equal(c.Lignes(), attendu) {
		t.Errorf("historique %q", c.Lignes())
	}
}
