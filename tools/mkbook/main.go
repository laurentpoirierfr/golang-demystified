// Command mkbook transforme les pages du site Hugo en projet de livre Quarto,
// rendu ensuite en PDF (via Typst) et en EPUB. L'ordre des parties et des
// chapitres suit le champ « weight » du front matter, exactement comme sur le
// site.
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func main() {
	docs := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	static := flag.String("static", "site/static", "répertoire des fichiers statiques Hugo")
	config := flag.String("config", "book/quarto/_quarto.yml", "configuration Quarto du livre")
	out := flag.String("out", "book/build/quarto", "répertoire du projet Quarto à produire")
	accueil := flag.String("accueil", "site/content/_index.md", "page d'accueil du site, dont la section « Préface » ouvre le livre")
	flag.Parse()

	root, err := content.LoadSection(*docs)
	if err != nil {
		log.Fatal(err)
	}
	if root.Body, err = preface(*accueil); err != nil {
		log.Fatal(err)
	}
	pages, err := writeQuarto(root, *static, *config, *out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s : %d pages\n", *out, pages)
}

// preface renvoie la section « ## Préface » de la page d'accueil du site :
// l'avant-propos du livre.
func preface(accueil string) (string, error) {
	p, err := content.LoadPage(accueil)
	if err != nil {
		return "", err
	}
	_, apres, ok := strings.Cut(p.Body, "## Préface\n")
	if !ok {
		return "", fmt.Errorf("%s : section « ## Préface » introuvable", accueil)
	}
	texte, _, _ := strings.Cut(apres, "\n## ")
	return strings.TrimSpace(texte) + "\n", nil
}
