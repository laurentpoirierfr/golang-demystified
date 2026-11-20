package concurrence

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSommeParallele(t *testing.T) {
	nombres := make([]int, 1001)
	for i := range nombres {
		nombres[i] = i
	}
	for _, parts := range []int{1, 3, 8, 2000} {
		if s := SommeParallele(nombres, parts); s != 500500 {
			t.Errorf("%d parts : %d", parts, s)
		}
	}
}

func TestPipeline(t *testing.T) {
	var r []int
	for v := range Carres(Genere(5)) {
		r = append(r, v)
	}
	if !slices.Equal(r, []int{1, 4, 9, 16, 25}) {
		t.Errorf("pipeline = %v", r)
	}
}

func TestTraite(t *testing.T) {
	var enCours, maxi atomic.Int32
	f := func(s string) string {
		n := enCours.Add(1)
		for m := maxi.Load(); n > m && !maxi.CompareAndSwap(m, n); m = maxi.Load() {
		}
		time.Sleep(5 * time.Millisecond)
		enCours.Add(-1)
		return strings.ToUpper(s)
	}
	taches := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	r := Traite(taches, 3, f)
	if !slices.Equal(r, []string{"A", "B", "C", "D", "E", "F", "G", "H"}) {
		t.Errorf("résultats = %v", r)
	}
	if m := maxi.Load(); m > 3 || m < 2 {
		t.Errorf("%d appels simultanés, attendu au plus 3 (et du parallélisme)", m)
	}
}
