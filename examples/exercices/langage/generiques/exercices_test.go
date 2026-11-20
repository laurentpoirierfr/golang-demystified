package generiques

import (
	"slices"
	"strconv"
	"testing"
)

func TestMax(t *testing.T) {
	if m, ok := Max([]int{3, 9, 2}); !ok || m != 9 {
		t.Errorf("Max entiers = %v", m)
	}
	if m, _ := Max([]string{"poire", "abricot", "zeste"}); m != "zeste" {
		t.Errorf("Max textes = %v", m)
	}
	if _, ok := Max([]float64{}); ok {
		t.Error("une slice vide n'a pas de maximum")
	}
}

func TestTransformeFiltre(t *testing.T) {
	textes := Transforme([]int{1, 2, 3}, strconv.Itoa)
	if !slices.Equal(textes, []string{"1", "2", "3"}) {
		t.Errorf("Transforme = %q", textes)
	}
	pairs := Filtre([]int{1, 2, 3, 4}, func(n int) bool { return n%2 == 0 })
	if !slices.Equal(pairs, []int{2, 4}) {
		t.Errorf("Filtre = %v", pairs)
	}
}

func TestEnsemble(t *testing.T) {
	a := NouvelEnsemble("go", "rust")
	a.Ajoute("go")
	if a.Taille() != 2 || !a.Contient("rust") || a.Contient("zig") {
		t.Error("Ajoute ou Contient se trompe")
	}
	u := a.Union(NouvelEnsemble("zig", "go"))
	if got := Trie(u); !slices.Equal(got, []string{"go", "rust", "zig"}) {
		t.Errorf("Union = %v", got)
	}
	if a.Taille() != 2 {
		t.Error("Union ne doit pas modifier l'ensemble d'origine")
	}
}
