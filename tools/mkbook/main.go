// Command mkbook transforme les pages du site Hugo en projet de livre Quarto,
// rendu ensuite en PDF (via Typst) et en EPUB. L'ordre des parties et des
// chapitres suit le champ « weight » du front matter, exactement comme sur le
// site.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func main() {
	docs := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	static := flag.String("static", "site/static", "répertoire des fichiers statiques Hugo")
	config := flag.String("config", "book/quarto/_quarto.yml", "configuration Quarto du livre")
	out := flag.String("out", "book/build/quarto", "répertoire du projet Quarto à produire")
	flag.Parse()

	root, err := content.LoadSection(*docs)
	if err != nil {
		log.Fatal(err)
	}
	pages, err := writeQuarto(root, *static, *config, *out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s : %d pages\n", *out, pages)
}
