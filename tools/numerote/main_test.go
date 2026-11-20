package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func TestRomain(t *testing.T) {
	for n, attendu := range map[int]string{1: "I", 4: "IV", 5: "V", 9: "IX", 14: "XIV", 40: "XL", 2026: "MMXXVI"} {
		if r := content.Romain(n); r != attendu {
			t.Errorf("content.Romain(%d) = %q, attendu %q", n, r, attendu)
		}
	}
}

func ecrit(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

func lit(t *testing.T, chemin string) string {
	t.Helper()
	b, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNumerote(t *testing.T) {
	d := t.TempDir()
	ecrit(t, filepath.Join(d, "_index.md"), "---\ntitle: \"Livre\"\nweight: 1\n---\n")
	ecrit(t, filepath.Join(d, "b", "_index.md"), "---\ntitle: \"Deuxième\"\nweight: 20\n---\n")
	ecrit(t, filepath.Join(d, "a", "_index.md"), "---\ntitle: \"Première\"\nweight: 10\n---\nTexte.\n")
	ecrit(t, filepath.Join(d, "a", "z.md"), "---\ntitle: \"JSON : encoding/json\"\nweight: 5\n---\nCorps.\n")
	ecrit(t, filepath.Join(d, "a", "y.md"), "---\ntitle: \"Un titre très long : et sa suite\"\nlinkTitle: \"7. Nom court\"\nweight: 30\n---\n")
	ecrit(t, filepath.Join(d, "b", "x.md"), "---\ntitle: \"Seul\"\nweight: 10\n---\n")

	racine, err := content.LoadSection(d)
	if err != nil {
		t.Fatal(err)
	}
	// En mode vérification, rien n'est écrit, mais tout est signalé.
	changements, err := numerote(racine, false)
	if err != nil || len(changements) != 5 {
		t.Fatalf("%d changements, err %v : %v", len(changements), err, changements)
	}
	if strings.Contains(lit(t, filepath.Join(d, "a", "z.md")), "linkTitle") {
		t.Fatal("le mode vérification ne doit rien écrire")
	}

	if _, err := numerote(racine, true); err != nil {
		t.Fatal(err)
	}
	for chemin, attendu := range map[string]string{
		"a/_index.md": "---\ntitle: \"Première\"\nlinkTitle: \"I - Première\"\nweight: 10\n---\nTexte.\n",
		"b/_index.md": "---\ntitle: \"Deuxième\"\nlinkTitle: \"II - Deuxième\"\nweight: 20\n---\n",
		"a/z.md":      "---\ntitle: \"JSON : encoding/json\"\nlinkTitle: \"1. JSON\"\nweight: 5\n---\nCorps.\n",
		"a/y.md":      "---\ntitle: \"Un titre très long : et sa suite\"\nlinkTitle: \"2. Nom court\"\nweight: 30\n---\n",
		"b/x.md":      "---\ntitle: \"Seul\"\nlinkTitle: \"1. Seul\"\nweight: 10\n---\n",
	} {
		if got := lit(t, filepath.Join(d, chemin)); got != attendu {
			t.Errorf("%s :\n%s\nattendu :\n%s", chemin, got, attendu)
		}
	}
	// Une seconde passe ne change plus rien.
	if changements, _ := numerote(racine, false); len(changements) != 0 {
		t.Errorf("la numérotation devrait être stable : %v", changements)
	}
}
