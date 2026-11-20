package collections

import (
	"maps"
	"slices"
	"testing"
)

func TestInverse(t *testing.T) {
	s := []int{1, 2, 3, 4}
	if r := Inverse(s); !slices.Equal(r, []int{4, 3, 2, 1}) {
		t.Errorf("Inverse = %v", r)
	}
	if !slices.Equal(s, []int{1, 2, 3, 4}) {
		t.Error("l'originale ne doit pas être modifiée")
	}
	if r := Inverse(nil); len(r) != 0 {
		t.Errorf("Inverse(nil) = %v", r)
	}
}

func TestFrequences(t *testing.T) {
	f := Frequences([]string{"go", "rust", "go", "zig", "go"})
	if !maps.Equal(f, map[string]int{"go": 3, "rust": 1, "zig": 1}) {
		t.Errorf("Frequences = %v", f)
	}
	if len(Frequences(nil)) != 0 {
		t.Error("aucun mot, aucune entrée")
	}
}

func TestUniques(t *testing.T) {
	if r := Uniques([]string{"b", "a", "b", "c", "a"}); !slices.Equal(r, []string{"b", "a", "c"}) {
		t.Errorf("Uniques = %v", r)
	}
}
