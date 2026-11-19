// Command mkbook transforme les pages du site Hugo en livre. L'ordre des
// parties et des chapitres suit le champ « weight » du front matter, exactement
// comme sur le site.
//
// Deux formats de sortie sont disponibles :
//
//	-format pandoc : un seul fichier Markdown, pour pandoc et eisvogel ;
//	-format quarto : un projet de livre Quarto (PDF via Typst, EPUB).
package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	content := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	static := flag.String("static", "site/static", "répertoire des fichiers statiques Hugo")
	format := flag.String("format", "pandoc", "format de sortie : pandoc ou quarto")
	out := flag.String("out", "book/build/book.md", "fichier (pandoc) ou répertoire (quarto) à produire")
	config := flag.String("config", "book/quarto/_quarto.yml", "configuration Quarto (format quarto)")
	flag.Parse()

	root, err := loadSection(*content)
	if err != nil {
		log.Fatal(err)
	}

	var pages int
	switch *format {
	case "pandoc":
		pages, err = writePandoc(root, *static, *out)
	case "quarto":
		pages, err = writeQuarto(root, *static, *config, *out)
	default:
		err = fmt.Errorf("format inconnu : %q", *format)
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s : %d pages\n", *out, pages)
}
