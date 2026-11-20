package ccalcul_test

import (
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/coulisses/pkg/ccalcul"
)

func TestSomme(t *testing.T) {
	t.Logf("code C utilisé : %v", ccalcul.AvecC)
	if s := ccalcul.Somme([]int64{1, 2, 3, 4}); s != 10 {
		t.Errorf("somme = %d", s)
	}
	if ccalcul.Somme(nil) != 0 {
		t.Error("somme vide")
	}
}

func TestParcourt(t *testing.T) {
	var vus []int64
	ccalcul.Parcourt([]int64{5, 6, 7}, func(v int64) { vus = append(vus, v) })
	if len(vus) != 3 || vus[0] != 5 || vus[2] != 7 {
		t.Errorf("vus = %v", vus)
	}
}

// Ce que coûte un appel cgo : la même somme sur 4 éléments, en C et en Go.
var (
	petit = []int64{1, 2, 3, 4}
	grand = make([]int64, 100_000)
	puits int64
)

func sommeGo(t []int64) int64 {
	var s int64
	for _, v := range t {
		s += v
	}
	return s
}

func BenchmarkPetitC(b *testing.B) {
	for b.Loop() {
		puits = ccalcul.Somme(petit)
	}
}

func BenchmarkPetitGo(b *testing.B) {
	for b.Loop() {
		puits = sommeGo(petit)
	}
}

func BenchmarkGrandC(b *testing.B) {
	for b.Loop() {
		puits = ccalcul.Somme(grand)
	}
}

func BenchmarkGrandGo(b *testing.B) {
	for b.Loop() {
		puits = sommeGo(grand)
	}
}
