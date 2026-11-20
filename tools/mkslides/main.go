// Command mkslides transforme chaque chapitre du site en un jeu de
// diapositives Marp (https://marp.app) : titre, plan, une ou plusieurs
// diapositives par section, et les liens vers le chapitre complet. Marp les
// rend ensuite en HTML et en PDF (make slides).
//
// Les diapositives sont tirées du texte du livre, sans être écrites à la main :
// la première phrase de chaque section, ses listes, son premier tableau, son
// premier exemple de code et sa première alerte.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func main() {
	docs := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	out := flag.String("out", "slides/build/md", "répertoire des diapositives Markdown à produire")
	site := flag.String("site", "https://laurentpoirierfr.github.io/golang-demystified/", "adresse du site")
	edition := flag.String("edition", "Go 1.27", "édition du livre")
	flag.Parse()

	root, err := content.LoadSection(*docs)
	if err != nil {
		log.Fatal(err)
	}
	n, err := write(root, *docs, *out, *site, *edition)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s : %d jeux de diapositives\n", *out, n)
}

// entry is a deck, for the index page.
type entry struct {
	Number      int
	Title, File string
}

// part is a rubrique, for the index page : its deck, then its chapters.
type part struct {
	entry
	Entries []entry
}

func write(root *content.Node, docs, out, site, edition string) (int, error) {
	var parts []part
	n := 0
	save := func(file, md string) error {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return err
		}
		n++
		return os.WriteFile(file, []byte(md), 0o644)
	}
	for _, p := range root.Children {
		if !p.Section {
			continue
		}
		// Numbered names follow the order of the book : 02-langage/07-interfaces.
		num := len(parts) + 1
		dir := fmt.Sprintf("%02d-%s", num, filepath.Base(p.Dir))
		r := rubrique{Number: num, Title: p.Title, Edition: edition, Intro: p.Body,
			URL: site + "docs/" + filepath.Base(p.Dir) + "/"}
		pt := part{entry: entry{Number: num, Title: p.Title, File: dir + "/00-presentation"}}
		for _, c := range p.Children {
			if c.Section {
				continue
			}
			cnum := len(pt.Entries) + 1
			name := fmt.Sprintf("%02d-%s", cnum, strings.TrimSuffix(filepath.Base(c.Path), ".md"))
			rel, err := filepath.Rel(docs, c.Path)
			if err != nil {
				return n, err
			}
			ch := chapter{
				Part:    fmt.Sprintf("%d. %s", num, p.Title),
				Title:   c.Title,
				URL:     site + "docs/" + filepath.ToSlash(strings.TrimSuffix(rel, ".md")) + "/",
				Edition: edition,
			}
			if err := save(filepath.Join(out, dir, name+".md"), deck(ch, c.Body)); err != nil {
				return n, err
			}
			pt.Entries = append(pt.Entries, entry{Number: cnum, Title: c.Title, File: dir + "/" + name})
			r.Chapters = append(r.Chapters, rubriqueChapter{Number: cnum, Title: c.Title, Body: c.Body})
		}
		// The rubrique deck comes first in its directory : 00-presentation.
		if err := save(filepath.Join(out, dir, "00-presentation.md"), rubriqueDeck(r)); err != nil {
			return n, err
		}
		parts = append(parts, pt)
	}
	f, err := os.Create(filepath.Join(out, "index.html"))
	if err != nil {
		return n, err
	}
	defer f.Close()
	return n, index.Execute(f, map[string]any{"Parts": parts, "Site": site, "Edition": edition})
}

var index = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Diapositives · Golang Demystified</title>
<style>
  @font-face { font-family: "GD Sans"; src: url("polices/SourceSans3-Regular.otf"); }
  @font-face { font-family: "GD Sans"; font-weight: bold; src: url("polices/SourceSans3-Bold.otf"); }
  :root { --bleu: #00ADD8; --bleu-fonce: #007D9C; --texte: #1f2937; --doux: #6b7280; --fond: #ffffff; --carte: #f3f7f9; }
  @media (prefers-color-scheme: dark) {
    :root { --texte: #e5e7eb; --doux: #9ca3af; --fond: #111827; --carte: #1f2937; }
  }
  * { box-sizing: border-box; }
  body { margin: 0; font-family: "GD Sans", "Source Sans 3", system-ui, sans-serif; color: var(--texte); background: var(--fond); }
  header { background: radial-gradient(circle at 85% 0%, rgba(0, 173, 216, .55) 0, rgba(0, 173, 216, 0) 45%), radial-gradient(circle at 0% 100%, rgba(0, 125, 156, .55) 0, rgba(0, 125, 156, 0) 50%), #0b1d26; color: #fff; padding: 3rem 1rem; }
  header div, main { max-width: 64rem; margin: 0 auto; }
  header h1 { margin: 0 0 .5rem; font-size: 2.2rem; }
  header p { margin: 0; opacity: .9; }
  header a { color: #fff; }
  main { padding: 1rem 1rem 3rem; }
  h2 { color: var(--bleu-fonce); border-bottom: 3px solid var(--bleu); padding-bottom: .3rem; margin-top: 2.5rem; }
  @media (prefers-color-scheme: dark) { h2 { color: var(--bleu); } }
  ul { list-style: none; padding: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(17rem, 1fr)); gap: .75rem; }
  li { background: var(--carte); border-radius: .75rem; padding: .75rem 1rem; display: flex; justify-content: space-between; align-items: center; gap: .5rem; border: 1px solid transparent; transition: border-color .15s; }
  li:hover { border-color: var(--bleu); }
  li span { font-weight: bold; }
  li a { color: var(--bleu-fonce); text-decoration: none; font-size: .9rem; white-space: nowrap; }
  @media (prefers-color-scheme: dark) { li a { color: var(--bleu); } }
  li a:hover { text-decoration: underline; }
  .rubrique { background: var(--carte); border-left: 5px solid var(--bleu); padding: .8rem 1rem; border-radius: .25rem; display: flex; gap: 1rem; align-items: center; flex-wrap: wrap; }
  .rubrique a { color: var(--bleu-fonce); font-weight: bold; }
  @media (prefers-color-scheme: dark) { .rubrique a { color: var(--bleu); } }
  .detail { color: var(--doux); margin: 1rem 0 .5rem; }
</style>
</head>
<body>
<header><div>
  <h1>Diapositives</h1>
  <p>Un diaporama par rubrique de <a href="{{.Site}}">Golang Demystified</a>, édition {{.Edition}}, pour présenter le livre, et un par chapitre pour l'approfondir. Numérotés dans l'ordre du livre, tirés de son texte, en HTML (flèches du clavier, touche F pour le plein écran) et en PDF.</p>
</div></header>
<main>
{{range .Parts}}<section>
<h2>{{printf "%02d" .Number}} · {{.Title}}</h2>
<p class="rubrique"><strong>Présentation de la rubrique</strong> <a href="{{.File}}.html">diaporama</a> <a href="{{.File}}.pdf">PDF</a></p>
<p class="detail">Une présentation détaillée par chapitre :</p>
<ul>
{{range .Entries}}  <li><span>{{printf "%02d" .Number}} · {{.Title}}</span> <a href="{{.File}}.html">diaporama</a> <a href="{{.File}}.pdf">PDF</a></li>
{{end}}</ul>
</section>
{{end}}</main>
</body>
</html>
`))
