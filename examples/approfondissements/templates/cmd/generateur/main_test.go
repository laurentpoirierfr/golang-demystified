package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGenere(t *testing.T) {
	code, err := Genere(Enum{Package: "commande", Type: "Statut", Valeurs: []string{"Brouillon", "Valide"}})
	if err != nil {
		t.Fatal(err)
	}
	// Le code produit doit être du Go valide.
	if _, err := parser.ParseFile(token.NewFileSet(), "statut_string.go", code, 0); err != nil {
		t.Fatalf("code invalide : %v\n%s", err, code)
	}
	for _, attendu := range []string{
		"func (v Statut) String() string",
		`case Valide:`,
		`return "valide"`,
		"func ParseStatut(s string) (Statut, error)",
	} {
		if !strings.Contains(string(code), attendu) {
			t.Errorf("%q absent du code généré :\n%s", attendu, code)
		}
	}
}

func TestEnTeteGenere(t *testing.T) {
	code, err := Genere(Enum{Package: "p", Type: "T", Valeurs: []string{"A"}})
	if err != nil {
		t.Fatal(err)
	}
	// Convention reconnue par les outils : https://go.dev/s/generatedcode
	premiere, _, _ := strings.Cut(string(code), "\n")
	if !strings.HasPrefix(premiere, "// Code generated ") || !strings.HasSuffix(premiere, " DO NOT EDIT.") {
		t.Errorf("en-tête non conforme : %q", premiere)
	}
}
