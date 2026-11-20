package collections

import (
	"slices"
	"strings"
	"testing"
)

func TestTrieParAge(t *testing.T) {
	p := []Personne{{"Rob", 68}, {"Ken", 82}, {"Ada", 36}, {"Bob", 68}}
	TrieParAge(p)
	attendu := []Personne{{"Ada", 36}, {"Bob", 68}, {"Rob", 68}, {"Ken", 82}}
	if !slices.Equal(p, attendu) {
		t.Errorf("TrieParAge = %v", p)
	}
}

func TestRegroupe(t *testing.T) {
	g := Regroupe([]string{"go", "rust", "gleam", "ruby", "c"}, func(s string) string { return s[:1] })
	if len(g) != 3 || !slices.Equal(g["g"], []string{"go", "gleam"}) || !slices.Equal(g["r"], []string{"rust", "ruby"}) {
		t.Errorf("Regroupe = %v", g)
	}
	parLongueur := Regroupe([]string{"go", "c", "zig"}, func(s string) int { return len(s) })
	if !slices.Equal(parLongueur[1], []string{"c"}) {
		t.Errorf("par longueur = %v", parLongueur)
	}
}

func TestIterateurs(t *testing.T) {
	multiplesDe7 := Filtre(Naturels(), func(n int) bool { return n%7 == 0 })
	if got := slices.Collect(Premiers(multiplesDe7, 4)); !slices.Equal(got, []int{7, 14, 21, 28}) {
		t.Errorf("4 premiers multiples de 7 = %v", got)
	}
	if got := slices.Collect(Premiers(Naturels(), 0)); len(got) != 0 {
		t.Errorf("Premiers(0) = %v", got)
	}
	// Les fonctions se combinent avec celles de la bibliothèque standard.
	mots := Filtre(slices.Values([]string{"Go", "rust", "Gleam"}), func(s string) bool { return strings.HasPrefix(s, "G") })
	if got := slices.Collect(mots); !slices.Equal(got, []string{"Go", "Gleam"}) {
		t.Errorf("Filtre sur slices.Values = %v", got)
	}
}
