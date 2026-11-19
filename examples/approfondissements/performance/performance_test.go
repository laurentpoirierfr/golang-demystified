package performance

import (
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"slices"
	"strings"
	"testing"
	"time"
)

var texte = strings.Repeat("Le gopher creuse un terrier. Le terrier du Gopher est profond ! ", 2000)

func TestVersionsIdentiques(t *testing.T) {
	naif, err1 := MotsFrequentsNaif(strings.NewReader(texte), 5)
	optimise, err2 := MotsFrequents(strings.NewReader(texte), 5)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if !slices.Equal(naif, optimise) {
		t.Errorf("résultats différents :\n%v\n%v", naif, optimise)
	}
}

func ExampleMotsFrequents() {
	freqs, _ := MotsFrequents(strings.NewReader(texte), 3)
	for _, f := range freqs {
		fmt.Println(f.Mot, f.Compte)
	}
	// Output:
	// gopher 4000
	// le 4000
	// terrier 4000
}

func BenchmarkMotsFrequentsNaif(b *testing.B) {
	for b.Loop() {
		MotsFrequentsNaif(strings.NewReader(texte), 10)
	}
}

func BenchmarkMotsFrequents(b *testing.B) {
	for b.Loop() {
		MotsFrequents(strings.NewReader(texte), 10)
	}
}

var parts = strings.Fields(strings.Repeat("alpha beta gamma delta ", 250))

func BenchmarkConcateneNaif(b *testing.B) {
	for b.Loop() {
		ConcateneNaif(parts)
	}
}

func BenchmarkConcatene(b *testing.B) {
	for b.Loop() {
		Concatene(parts)
	}
}

// TestAllocations fixe le nombre d'allocations : une régression de
// performance devient un échec de test.
func TestAllocations(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() { Concatene(parts) }); n != 1 {
		t.Errorf("Concatene : %.0f allocations, attendu 1", n)
	}
	if n := testing.AllocsPerRun(10, func() { ConcateneNaif(parts) }); n < float64(len(parts)) {
		t.Errorf("ConcateneNaif : %.0f allocations, attendu au moins %d", n, len(parts))
	}
	// Une allocation par mot distinct, pas par occurrence : le texte contient
	// 8 mots distincts et 22 000 occurrences.
	if n := testing.AllocsPerRun(10, func() { MotsFrequents(strings.NewReader(texte), 10) }); n > 50 {
		t.Errorf("MotsFrequents : %.0f allocations, attendu 50 au plus", n)
	}
}

// TestProfilCPU écrit un profil CPU, comme le ferait go test -cpuprofile.
func TestProfilCPU(t *testing.T) {
	f, err := os.Create(t.TempDir() + "/cpu.pprof")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		MotsFrequentsNaif(strings.NewReader(texte), 10)
	}
	pprof.StopCPUProfile()
	if info, _ := f.Stat(); info.Size() == 0 {
		t.Error("profil vide")
	}
}

// TestFlightRecorder garde en mémoire les dernières secondes de trace, et les
// écrit à la demande : par exemple quand une requête est anormalement lente.
func TestFlightRecorder(t *testing.T) {
	fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
		MinAge:   5 * time.Second,
		MaxBytes: 4 << 20,
	})
	if err := fr.Start(); err != nil {
		t.Fatal(err)
	}
	defer fr.Stop()

	debut := time.Now()
	MotsFrequentsNaif(strings.NewReader(texte), 10)
	if time.Since(debut) > 0 { // en vrai : un seuil, par exemple 500 ms
		var b bytes.Buffer
		if _, err := fr.WriteTo(&b); err != nil {
			t.Fatal(err)
		}
		if b.Len() == 0 {
			t.Error("trace vide")
		}
	}
}
