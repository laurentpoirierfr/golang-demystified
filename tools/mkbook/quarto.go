package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

// writeQuarto produit un projet de livre Quarto dans le répertoire out :
//   - un fichier .qmd par partie (son introduction) et par chapitre ;
//   - index.qmd, la préface, tirée de la page racine du contenu ;
//   - _quarto.yml, qui liste les parties et chapitres dans l'ordre du site.
//
// La configuration commune (titre, formats, couverture) vient du fichier
// config, auquel la liste des chapitres est ajoutée.
func writeQuarto(root *content.Node, static, config, out string) (int, error) {
	if err := os.RemoveAll(out); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return 0, err
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return 0, err
	}
	staticAbs, err := filepath.Abs(static)
	if err != nil {
		return 0, err
	}
	c := &converter{static: staticAbs, base: abs, callouts: true}

	write := func(name, title string, p content.Page, shift int) error {
		body := c.convert(p, shift)
		content := fmt.Sprintf("# %s\n\n%s", title, body)
		return os.WriteFile(filepath.Join(out, name), []byte(content), 0o644)
	}

	// La préface : la page racine du contenu, non numérotée.
	if err := write("index.qmd", "Avant-propos {.unnumbered}", root.Page, 0); err != nil {
		return 0, err
	}
	pages := 1

	var chapters strings.Builder
	chapters.WriteString("  chapters:\n    - index.qmd\n")
	for i, part := range root.Children {
		partFile := fmt.Sprintf("%02d-%s.qmd", i+1, slug(part.Title))
		if err := write(partFile, part.Title, part.Page, 0); err != nil {
			return 0, err
		}
		pages++
		fmt.Fprintf(&chapters, "    - part: %s\n      chapters:\n", partFile)
		for j, ch := range part.Children {
			name := fmt.Sprintf("%02d-%02d-%s.qmd", i+1, j+1, slug(ch.Title))
			// Les pages commencent au niveau 2 : elles restent sous le titre
			// du chapitre (niveau 1) sans décalage.
			if err := write(name, ch.Title, ch.Page, 0); err != nil {
				return 0, err
			}
			pages++
			fmt.Fprintf(&chapters, "        - %s\n", name)
		}
	}
	if c.err != nil {
		return 0, c.err
	}

	cfg, err := os.ReadFile(config)
	if err != nil {
		return 0, err
	}
	yml := strings.Replace(string(cfg), "  # chapters: généré par mkbook\n", chapters.String(), 1)
	if yml == string(cfg) {
		return 0, fmt.Errorf("%s : marqueur « # chapters: généré par mkbook » introuvable", config)
	}
	return pages, os.WriteFile(filepath.Join(out, "_quarto.yml"), []byte(yml), 0o644)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug produit un nom de fichier ASCII à partir d'un titre.
func slug(s string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ê", "e", "à", "a", "â", "a",
		"ç", "c", "î", "i", "ï", "i", "ô", "o", "û", "u", "ù", "u", "'", "-")
	s = r.Replace(strings.ToLower(s))
	return strings.Trim(nonSlug.ReplaceAllString(s, "-"), "-")
}
