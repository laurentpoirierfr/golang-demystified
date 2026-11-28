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
	docs := filepath.Join(d, "docs")
	accueil := filepath.Join(d, "_index.md")
	ecrit(t, accueil, "---\ntitle: \"Livre\"\n---\n\n# Livre\n\n<!-- sommaire -->\n<!-- /sommaire -->\n")
	ecrit(t, filepath.Join(docs, "_index.md"), "---\ntitle: \"Livre\"\nweight: 1\n---\n\n<!-- sommaire -->\n<!-- /sommaire -->\n")
	ecrit(t, filepath.Join(docs, "b", "_index.md"), "---\ntitle: \"Deuxième\"\nweight: 20\n---\n")
	ecrit(t, filepath.Join(docs, "a", "_index.md"), "---\ntitle: \"Première\"\nlinkTitle: \"I - Première\"\nweight: 10\n---\nTexte.\n\n<!-- sommaire -->\n<!-- /sommaire -->\n")
	ecrit(t, filepath.Join(docs, "a", "z.md"), "---\ntitle: \"JSON : encoding/json\"\nweight: 5\n---\nCorps.\n")
	ecrit(t, filepath.Join(docs, "a", "y.md"), "---\ntitle: \"Un titre très long : et sa suite\"\nlinkTitle: \"7. Nom court\"\nweight: 30\n---\n")
	ecrit(t, filepath.Join(docs, "b", "x.md"), "---\ntitle: \"Seul\"\nweight: 10\n---\n")
	ecrit(t, filepath.Join(docs, "annexes", "_index.md"), "---\ntitle: \"Annexes\"\nweight: 90\n---\n")
	ecrit(t, filepath.Join(docs, "annexes", "g.md"), "---\ntitle: \"Glossaire\"\nweight: 1\n---\nTermes.\n")

	charge := func() *content.Node {
		t.Helper()
		racine, err := content.LoadSection(docs)
		if err != nil {
			t.Fatal(err)
		}
		return racine
	}
	// En mode vérification, rien n'est écrit, mais tout est signalé.
	changements, err := numerote(charge(), accueil, false)
	if err != nil || len(changements) != 9 {
		t.Fatalf("%d changements, err %v : %v", len(changements), err, changements)
	}
	if strings.Contains(lit(t, filepath.Join(docs, "a", "z.md")), "I.1") {
		t.Fatal("le mode vérification ne doit rien écrire")
	}

	if _, err := numerote(charge(), accueil, true); err != nil {
		t.Fatal(err)
	}
	for chemin, attendu := range map[string]string{
		"docs/a/_index.md": "---\ntitle: \"I. Première\"\nbookCollapseSection: true\nweight: 10\n---\n\n# Partie I — Première\n\nTexte.\n\n" +
			"<!-- sommaire -->\n\n- **I.1** [JSON](z/)\n- **I.2** [Nom court](y/)\n\n<!-- /sommaire -->\n",
		"docs/a/z.md":            "---\ntitle: \"I.1 JSON\"\nweight: 5\n---\n\n# Chapitre I.1 — JSON : encoding/json\n\nCorps.\n",
		"docs/a/y.md":            "---\ntitle: \"I.2 Nom court\"\nweight: 30\n---\n\n# Chapitre I.2 — Un titre très long : et sa suite\n",
		"docs/b/x.md":            "---\ntitle: \"II.1 Seul\"\nweight: 10\n---\n\n# Chapitre II.1 — Seul\n",
		"docs/annexes/_index.md": "---\ntitle: \"Annexes\"\nbookCollapseSection: true\nweight: 90\n---\n\n# Annexes\n",
		"docs/annexes/g.md":      "---\ntitle: \"A. Glossaire\"\nweight: 1\n---\n\n# Annexe A — Glossaire\n\nTermes.\n",
		"docs/_index.md": "---\ntitle: \"Chapitres\"\nbookFlatSection: true\nweight: 1\n---\n\n# Chapitres\n\n<!-- sommaire -->\n\n" +
			"**[Partie I — Première](a/)**\n\n- **I.1** [JSON](a/z/)\n- **I.2** [Nom court](a/y/)\n\n" +
			"**[Partie II — Deuxième](b/)**\n\n- **II.1** [Seul](b/x/)\n\n" +
			"**[Annexes](annexes/)**\n\n- **A** [Glossaire](annexes/g/)\n\n<!-- /sommaire -->\n",
	} {
		if got := lit(t, filepath.Join(d, chemin)); got != attendu {
			t.Errorf("%s :\n%s\nattendu :\n%s", chemin, got, attendu)
		}
	}
	if got := lit(t, accueil); !strings.Contains(got, "### Partie II — Deuxième\n\n- **II.1** [Seul](docs/b/x/)\n") {
		t.Errorf("sommaire de l'accueil :\n%s", got)
	}

	// Le paquet content retire la numérotation pour le livre PDF.
	if p, _ := content.LoadPage(filepath.Join(docs, "a", "z.md")); p.Title != "JSON : encoding/json" || p.Body != "Corps.\n" {
		t.Errorf("content.LoadPage : titre %q, corps %q", p.Title, p.Body)
	}
	if p, _ := content.LoadPage(filepath.Join(docs, "a", "_index.md")); p.Title != "Première" || strings.Contains(p.Body, "sommaire") {
		t.Errorf("content.LoadPage : titre %q, corps %q", p.Title, p.Body)
	}

	// Une seconde passe ne change plus rien.
	if changements, _ := numerote(charge(), accueil, false); len(changements) != 0 {
		t.Errorf("la numérotation devrait être stable : %v", changements)
	}
}
