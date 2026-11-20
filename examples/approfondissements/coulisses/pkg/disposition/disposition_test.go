package disposition_test

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/coulisses/pkg/disposition"
)

// Sur une architecture 64 bits (amd64, arm64).
func ExamplePlan() {
	for _, c := range disposition.Plan() {
		fmt.Printf("%-8s décalage %2d, taille %d\n", c.Nom, c.Decalage, c.Taille)
	}
	fmt.Println("MalRangee :", unsafe.Sizeof(disposition.MalRangee{}), "octets")
	fmt.Println("BienRangee :", unsafe.Sizeof(disposition.BienRangee{}), "octets")
	// Output:
	// Actif    décalage  0, taille 1
	// Montant  décalage  8, taille 8
	// Code     décalage 16, taille 1
	// Nombre   décalage 20, taille 4
	// Visible  décalage 24, taille 1
	// MalRangee : 32 octets
	// BienRangee : 16 octets
}

func TestSansCopie(t *testing.T) {
	b := []byte("gopher")
	s := disposition.Chaine(b)
	if s != "gopher" || unsafe.StringData(s) != &b[0] {
		t.Error("la chaîne devrait partager la mémoire des octets")
	}
	// Le danger : modifier les octets modifie la chaîne « immuable ».
	b[0] = 'G'
	if s != "Gopher" {
		t.Errorf("s = %q", s)
	}
	if o := disposition.Octets("go"); string(o) != "go" || cap(o) != 2 {
		t.Errorf("o = %q", o)
	}
	if disposition.Chaine(nil) != "" || disposition.Octets("") != nil {
		t.Error("valeurs vides")
	}
}

var (
	donnees = make([]byte, 4096)
	puits   string
)

func BenchmarkConversionCopie(b *testing.B) {
	for b.Loop() {
		puits = string(donnees) // copie de 4 Kio
	}
}

func BenchmarkConversionSansCopie(b *testing.B) {
	for b.Loop() {
		puits = disposition.Chaine(donnees)
	}
}

// Tranche de 10 000 structures : la mémoire gagnée par un bon rangement.
func BenchmarkTrancheMalRangee(b *testing.B) {
	for b.Loop() {
		_ = make([]disposition.MalRangee, 10_000)
	}
}

func BenchmarkTrancheBienRangee(b *testing.B) {
	for b.Loop() {
		_ = make([]disposition.BienRangee, 10_000)
	}
}
