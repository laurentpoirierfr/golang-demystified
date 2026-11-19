package templates

import (
	"fmt"
	htmltemplate "html/template"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"
)

func Example_texte() {
	const lettre = `Bonjour {{.Nom}},
{{if .Premium}}Merci pour votre fidélité !{{else}}Découvrez notre offre Premium.{{end}}
Votre commande contient {{len .Articles}} article(s) :
{{- range $i, $a := .Articles}}
  {{$i}}. {{$a.Nom | printf "%-10s"}} {{printf "%6.2f" $a.Prix}} €
{{- end}}
`
	type ligne struct {
		Nom  string
		Prix float64
	}
	t := template.Must(template.New("lettre").Parse(lettre))
	t.Execute(os.Stdout, map[string]any{
		"Nom":      "Alice",
		"Premium":  true,
		"Articles": []ligne{{"Clavier", 89.9}, {"Souris", 25}},
	})
	// Output:
	// Bonjour Alice,
	// Merci pour votre fidélité !
	// Votre commande contient 2 article(s) :
	//   0. Clavier     89.90 €
	//   1. Souris      25.00 €
}

func Example_fonctions() {
	t := template.Must(template.New("").Funcs(template.FuncMap{
		"majuscules": strings.ToUpper,
		"join":       strings.Join,
	}).Parse(`{{majuscules .Titre}} : {{join .Tags ", "}}` + "\n"))
	t.Execute(os.Stdout, map[string]any{"Titre": "go 1.27", "Tags": []string{"json", "uuid"}})
	// Output: GO 1.27 : json, uuid
}

func Example_rangeEntier() {
	t := template.Must(template.New("").Parse(`{{range $i := 3}}[{{$i}}]{{end}}` + "\n"))
	t.Execute(os.Stdout, nil)
	// Output: [0][1][2]
}

func Example_cleManquante() {
	t := template.Must(template.New("").Option("missingkey=error").Parse("{{.Inconnue}}"))
	err := t.Execute(os.Stdout, map[string]string{})
	fmt.Println(err)
	// Output: template: :1:2: executing "" at <.Inconnue>: map has no entry for key "Inconnue"
}

func Example_echappement() {
	t := htmltemplate.Must(htmltemplate.New("").Parse(
		`<p title="{{.}}">{{.}}</p><a href="/recherche?q={{.}}">lien</a><script>var q = {{.}};</script>` + "\n"))
	t.Execute(os.Stdout, `<script>alert("x")</script>`)
	// Output: <p title="&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;">&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</p><a href="/recherche?q=%3cscript%3ealert%28%22x%22%29%3c%2fscript%3e">lien</a><script>var q = "\u003cscript\u003ealert(\"x\")\u003c/script\u003e";</script>
}

func TestSite(t *testing.T) {
	site, err := NouveauSite()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	err = site.Affiche(&b, "article", map[string]any{
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
	site, err := NouveauSite()
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := site.Affiche(&b, "accueil", map[string]any{"Date": time.Now()}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "<title>Golang Demystified</title>") ||
		!strings.Contains(b.String(), "Aucun article.") {
		t.Errorf("page d'accueil inattendue :\n%s", b.String())
	}
}
