package fichiers

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompteLignes(t *testing.T) {
	n, err := CompteLignes(strings.NewReader("un\n\ndeux\n   \ntrois"))
	if err != nil || n != 3 {
		t.Errorf("CompteLignes = %d, %v", n, err)
	}
}

func TestParExtension(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go":            {},
		"README.md":          {},
		"pkg/a/a.go":         {},
		"pkg/a/a_test.go":    {},
		"pkg/b/donnees.json": {},
	}
	got, err := ParExtension(fsys, ".go")
	if err != nil || !slices.Equal(got, []string{"main.go", "pkg/a/a.go", "pkg/a/a_test.go"}) {
		t.Errorf("ParExtension = %v, %v", got, err)
	}
}

func TestEcritAtomique(t *testing.T) {
	d := t.TempDir()
	chemin := filepath.Join(d, "config.json")
	for _, contenu := range []string{`{"v": 1}`, `{"v": 2}`} {
		if err := EcritAtomique(chemin, []byte(contenu)); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(chemin); string(b) != contenu {
			t.Errorf("contenu %q, attendu %q", b, contenu)
		}
	}
	if entrees, _ := os.ReadDir(d); len(entrees) != 1 {
		t.Errorf("%d fichiers : il reste un fichier temporaire", len(entrees))
	}
	if err := EcritAtomique(filepath.Join(d, "absent", "x"), nil); err == nil {
		t.Error("un répertoire absent doit donner une erreur")
	}
}
