---
title: "Templates"
linkTitle: "2. Templates"
weight: 20
---

Go fournit deux moteurs de gabarits qui partagent la même syntaxe :

- **`text/template`** produit du texte : e-mails, fichiers de configuration,
  rapports, code source ;
- **`html/template`** produit du HTML, et **échappe automatiquement** les
  données selon leur contexte, ce qui protège contre les injections (XSS).

Ce chapitre présente la syntaxe, puis construit un petit site avec un gabarit
de mise en page commun, et termine par la génération de code. Le code est dans
[`examples/approfondissements/templates`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/templates).

## La syntaxe

Un gabarit est un texte qui contient des **actions** entre `{{` et `}}`. Le
point, `.`, désigne la donnée courante, passée à `Execute` :

```go
const lettre = `Bonjour {{.Nom}},
{{if .Premium}}Merci pour votre fidélité !{{else}}Découvrez notre offre Premium.{{end}}
Votre commande contient {{len .Articles}} article(s) :
{{- range $i, $a := .Articles}}
  {{$i}}. {{$a.Nom | printf "%-10s"}} {{printf "%6.2f" $a.Prix}} €
{{- end}}
`

t := template.Must(template.New("lettre").Parse(lettre))
err := t.Execute(os.Stdout, map[string]any{
	"Nom":      "Alice",
	"Premium":  true,
	"Articles": []ligne{{"Clavier", 89.9}, {"Souris", 25}},
})
```

```
Bonjour Alice,
Merci pour votre fidélité !
Votre commande contient 2 article(s) :
  0. Clavier     89.90 €
  1. Souris      25.00 €
```

| Action                              | Rôle                                                   |
|-------------------------------------|--------------------------------------------------------|
| `{{.Champ}}`, `{{.Methode}}`        | Champ ou méthode de la donnée courante, clé de map     |
| `{{if X}}…{{else if Y}}…{{else}}…{{end}}` | Condition (valeur vide = fausse)                 |
| `{{range X}}…{{else}}…{{end}}`      | Boucle ; `else` si la collection est vide              |
| `{{range $i, $v := X}}`             | Boucle avec variables                                  |
| `{{with X}}…{{end}}`                | Change `.` en `X`, si `X` n'est pas vide               |
| `{{$v := X}}`                       | Déclare une variable                                   |
| `{{X \| f}}`                         | *Pipeline* : passe `X` en dernier argument de `f`       |
| `{{define "nom"}}…{{end}}`          | Définit un sous-gabarit                                |
| `{{template "nom" .}}`              | Insère un sous-gabarit                                 |
| `{{block "nom" .}}défaut{{end}}`    | Définit et insère un sous-gabarit, redéfinissable      |
| `{{/* commentaire */}}`             | Commentaire                                            |
| `{{- …}}`, `{{… -}}`                | Supprime les espaces avant ou après l'action           |

Les fonctions intégrées comprennent `len`, `index`, `slice`, `printf`,
`print`, `and`, `or`, `not`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `html`,
`urlquery` et `js`.

> [!NOTE]
> **Go 1.24** — `range` accepte, dans les gabarits comme dans le langage, un
> entier ou un itérateur : `{{range $i := 3}}[{{$i}}]{{end}}` affiche
> `[0][1][2]`.

### Ajouter des fonctions

`Funcs` ajoute des fonctions, **avant** l'analyse du gabarit :

```go
t := template.Must(template.New("").Funcs(template.FuncMap{
	"majuscules": strings.ToUpper,
	"join":       strings.Join,
}).Parse(`{{majuscules .Titre}} : {{join .Tags ", "}}`))

t.Execute(os.Stdout, map[string]any{"Titre": "go 1.27", "Tags": []string{"json", "uuid"}})
// GO 1.27 : json, uuid
```

Une fonction renvoie une valeur, ou une valeur et une `error` : une erreur
interrompt l'exécution du gabarit.

### Les erreurs

Par défaut, une clé absente d'une map affiche `<no value>`. L'option
`missingkey=error` transforme cet oubli en erreur, ce qui est plus sûr :

```go
t := template.Must(template.New("").Option("missingkey=error").Parse("{{.Inconnue}}"))
err := t.Execute(os.Stdout, map[string]string{})
// template: :1:2: executing "" at <.Inconnue>: map has no entry for key "Inconnue"
```

`template.Must` panique si l'analyse échoue. On l'utilise pour les gabarits
connus à la compilation, analysés au démarrage du programme : une erreur de
syntaxe est alors détectée immédiatement, et non à la première requête.

## html/template et l'échappement contextuel

`html/template` a la même API que `text/template`, mais il comprend la
structure du HTML. Chaque donnée est échappée selon **l'endroit** où elle est
insérée : texte, attribut, URL, JavaScript, CSS.

```go
t := template.Must(template.New("").Parse(
	`<p title="{{.}}">{{.}}</p><a href="/recherche?q={{.}}">lien</a><script>var q = {{.}};</script>`))
t.Execute(os.Stdout, `<script>alert("x")</script>`)
```

Le même texte malveillant est neutralisé de quatre façons différentes :

| Contexte      | Résultat                                                         |
|---------------|------------------------------------------------------------------|
| texte HTML    | `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`                |
| attribut      | identique, entre guillemets                                      |
| URL           | `%3cscript%3ealert%28%22x%22%29%3c%2fscript%3e`                  |
| JavaScript    | une chaîne JavaScript valide, où `<` et `>` sont encodés          |

> [!CAUTION]
> N'utilisez **jamais** `text/template` pour produire du HTML : il n'échappe
> rien. Et ne désactivez l'échappement, avec les types `template.HTML`,
> `template.URL` ou `template.JS`, que pour des contenus dont vous maîtrisez
> l'origine, comme du HTML produit par votre propre code.

## Un site avec une mise en page commune

Les pages d'un site partagent une mise en page : en-tête, pied de page,
feuilles de style. Avec `block`, le gabarit de base définit des
emplacements, que chaque page redéfinit.

Le site est un package interne, `internal/site`, qui embarque ses gabarits.
Le gabarit de base, `internal/site/gabarits/base.html` :

```html
{{define "base"}}<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <title>{{block "titre" .}}Golang Demystified{{end}}</title>
</head>
<body>
  <header><a href="/">Accueil</a></header>
  <main>
{{block "contenu" .}}{{end}}
  </main>
  <footer>Généré le {{.Date | date}}</footer>
</body>
</html>
{{end}}
```

Une page, `gabarits/article.html`, redéfinit les deux emplacements :

```html
{{define "titre"}}{{.Article.Titre}} – Golang Demystified{{end}}

{{define "contenu"}}
    <article>
      <h1>{{.Article.Titre}}</h1>
      <p class="auteur">par {{.Article.Auteur}}</p>
      {{- with .Article.Tags}}
      <ul>
        {{- range .}}
        <li><a href="/tags/{{.}}">{{.}}</a></li>
        {{- end}}
      </ul>
      {{- end}}
      <div>{{.Article.Resume}}</div>
    </article>
{{end}}
```

La page d'accueil ne redéfinit que le contenu : son titre est celui par
défaut.

### Analyser une fois, au démarrage

Deux pages qui redéfinissent le même bloc ne peuvent pas cohabiter dans le
même ensemble de gabarits : la seconde définition écraserait la première. On
crée donc **un ensemble par page**, composé de la base et de la page. Les
gabarits sont embarqués dans l'exécutable avec `embed` et analysés une seule
fois, au démarrage :

```go
//go:embed gabarits/*.html
var gabarits embed.FS

var fonctions = template.FuncMap{
	"date": func(t time.Time) string { return t.Format("02/01/2006") },
	"add":  func(a, b int) int { return a + b },
}

type Site struct {
	pages map[string]*template.Template
}

func NouveauSite() (*Site, error) {
	s := &Site{pages: make(map[string]*template.Template)}
	for _, page := range []string{"accueil", "article"} {
		t, err := template.New(page).Funcs(fonctions).ParseFS(gabarits,
			"gabarits/base.html", "gabarits/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("gabarit %s : %w", page, err)
		}
		s.pages[page] = t
	}
	return s, nil
}

func (s *Site) Affiche(w io.Writer, page string, donnees any) error {
	t, ok := s.pages[page]
	if !ok {
		return fmt.Errorf("page inconnue : %q", page)
	}
	return t.ExecuteTemplate(w, "base", donnees)
}
```

Un `*template.Template` analysé peut être exécuté par plusieurs goroutines à
la fois : un même `Site` sert toutes les requêtes HTTP.

```go
func (a *App) article(w http.ResponseWriter, r *http.Request) {
	art, err := a.store.Article(r.Context(), r.PathValue("slug"))
	…
	var buf bytes.Buffer
	if err := a.site.Affiche(&buf, "article", map[string]any{"Article": art, "Date": time.Now()}); err != nil {
		http.Error(w, "erreur interne", http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}
```

Le rendu dans un tampon avant l'écriture évite d'envoyer une page à moitié
construite si le gabarit échoue en cours d'exécution.

### Tester les gabarits

Un test exécute chaque page avec des données représentatives, et vérifie à la
fois le contenu et l'échappement :

```go
html := rendu(t, "article", Article{Titre: "Go 1.27 <enfin>", …})
// <title>Go 1.27 &lt;enfin&gt; – Golang Demystified</title>
// <a href="/tags/g%c3%a9n%c3%a9riques">génériques</a>
```

## Générer du code

`text/template` est aussi l'outil standard pour **générer du code Go**. La
directive `//go:generate` déclare la commande à exécuter, et `go generate`
la lance :

```go
//go:generate go run ./cmd/generateur -type Statut -package commande Brouillon Valide Expediee
```

```bash
go generate ./...
```

Le générateur est un exécutable du projet, placé dans `cmd/generateur`. Il
produit une méthode `String` et une fonction d'analyse pour un
type énuméré :

```go
var gabarit = template.Must(template.New("enum").Funcs(template.FuncMap{
	"lower": strings.ToLower,
}).Parse(`// Code generated by generateur. DO NOT EDIT.

package {{.Package}}

import "fmt"

// String renvoie le nom de la valeur.
func (v {{.Type}}) String() string {
	switch v {
	{{- range .Valeurs}}
	case {{.}}:
		return "{{lower .}}"
	{{- end}}
	}
	return fmt.Sprintf("{{.Type}}(%d)", int(v))
}
…
`))

func Genere(e Enum) ([]byte, error) {
	var b bytes.Buffer
	if err := gabarit.Execute(&b, e); err != nil {
		return nil, err
	}
	return format.Source(b.Bytes()) // gofmt, et vérification de la syntaxe
}
```

Trois bonnes pratiques :

- passer le résultat à **`go/format.Source`**, qui le formate comme `gofmt` et
  échoue si le code produit n'est pas du Go valide ;
- commencer le fichier par le commentaire `// Code generated … DO NOT EDIT.`,
  **en anglais et exactement sous cette forme** : c'est une convention
  reconnue par les outils ([gopls](https://pkg.go.dev/golang.org/x/tools/gopls), linters, revue de code), qui évitent de
  modifier ou d'analyser ces fichiers ;
- **versionner** les fichiers générés : un utilisateur du module n'a pas à
  lancer `go generate` pour compiler.

Pour des générateurs plus élaborés, le package `go/ast` analyse le code
existant, par exemple pour trouver automatiquement les constantes d'un type,
comme le fait l'outil [`stringer`](https://pkg.go.dev/golang.org/x/tools/cmd/stringer).
