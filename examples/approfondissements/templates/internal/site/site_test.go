package site

import (
	"strings"
	"testing"
	"time"
)

func TestSite(t *testing.T) {
	s, err := NouveauSite()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	err = s.Affiche(&b, "article", map[string]any{
		"Date": time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		"Article": Article{
			Titre:  "Go 1.27 <enfin>",
			Auteur: "Alice",
			Tags:   []string{"génériques", "json"},
			Resume: "<em>Méthodes génériques</em>",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	html := b.String()
	for _, attendu := range []string{
		"<title>Go 1.27 &lt;enfin&gt; – Golang Demystified</title>",
		"<h1>Go 1.27 &lt;enfin&gt;</h1>",
		`<a href="/tags/g%c3%a9n%c3%a9riques">génériques</a>`,
		"<div><em>Méthodes génériques</em></div>",
		"Généré le 12/08/2026",
	} {
		if !strings.Contains(html, attendu) {
			t.Errorf("%q absent de :\n%s", attendu, html)
		}
	}
}

func TestSiteAccueilVide(t *testing.T) {
	s, err := NouveauSite()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := s.Affiche(&b, "accueil", map[string]any{"Date": time.Now()}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "<title>Golang Demystified</title>") ||
		!strings.Contains(b.String(), "Aucun article.") {
		t.Errorf("page d'accueil inattendue :\n%s", b.String())
	}
}
