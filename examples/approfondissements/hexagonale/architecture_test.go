// Package hexagonale_test vérifie les règles de dépendance de l'architecture.
package hexagonale_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/"

// imports renvoie les imports des fichiers non test d'un répertoire.
func imports(t *testing.T, dir string) []string {
	t.Helper()
	fichiers, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var res []string
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		ast, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			chemin, _ := strconv.Unquote(imp.Path.Value)
			res = append(res, chemin)
		}
	}
	return res
}

// TestDirectionDesDependances garantit que les dépendances pointent vers le
// centre : le domaine ne dépend de rien, l'application ne dépend que du
// domaine, et aucun des deux ne connaît les adaptateurs.
func TestDirectionDesDependances(t *testing.T) {
	regles := []struct {
		paquet    string
		autorises []string // packages du projet autorisés
	}{
		{"domaine", nil},
		{"application", []string{"domaine"}},
	}
	for _, r := range regles {
		for _, imp := range imports(t, r.paquet) {
			interne, ok := strings.CutPrefix(imp, module)
			if !ok {
				if strings.Contains(imp, ".") { // module externe (contient un domaine)
					t.Errorf("%s importe un module externe : %s", r.paquet, imp)
				}
				continue
			}
			autorise := false
			for _, a := range r.autorises {
				autorise = autorise || interne == a
			}
			if !autorise {
				t.Errorf("%s ne doit pas importer %s", r.paquet, interne)
			}
		}
	}
}
