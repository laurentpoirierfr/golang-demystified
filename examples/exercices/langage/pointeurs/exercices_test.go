package pointeurs

import (
	"slices"
	"testing"
)

func TestEchange(t *testing.T) {
	a, b := 1, 2
	Echange(&a, &b)
	if a != 2 || b != 1 {
		t.Errorf("a = %d, b = %d", a, b)
	}
}

func TestInverse(t *testing.T) {
	l := Liste(1, 2, 3, 4)
	deuxieme := l.Suivant
	l = Inverse(l)
	if !slices.Equal(l.Valeurs(), []int{4, 3, 2, 1}) {
		t.Errorf("Inverse = %v", l.Valeurs())
	}
	if deuxieme.Suivant.Valeur != 1 {
		t.Error("l'inversion doit réutiliser les mêmes nœuds")
	}
	if Inverse(nil) != nil {
		t.Error("l'inverse d'une liste vide est vide")
	}
}

func TestArbre(t *testing.T) {
	var a *Arbre
	for _, v := range []int{5, 3, 8, 1, 4, 8} {
		a = a.Insere(v)
	}
	if !slices.Equal(a.EnOrdre(), []int{1, 3, 4, 5, 8}) {
		t.Errorf("EnOrdre = %v", a.EnOrdre())
	}
	if !a.Contient(4) || a.Contient(6) {
		t.Error("Contient se trompe")
	}
}
