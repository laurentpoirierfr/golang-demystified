// Package content lit les pages du site Hugo (site/content/docs) : front
// matter, corps Markdown, et arborescence des parties et des chapitres, dans
// l'ordre du champ « weight ». Il est partagé par mkbook (le livre) et
// mkslides (les diapositives).
package content

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Page est une page du site.
type Page struct {
	Title       string
	Description string // phrase reprise par les sommaires
	Weight      int
	Body        string // Markdown, sans le front matter
	Dir         string // répertoire de la page, pour résoudre les images relatives
	Path        string
}

// Node est une page et ses enfants (pour une section).
type Node struct {
	Page
	Children []*Node
	Section  bool
}

// LoadSection lit une section Hugo (un répertoire contenant _index.md).
func LoadSection(dir string) (*Node, error) {
	idx, err := LoadPage(filepath.Join(dir, "_index.md"))
	if err != nil {
		return nil, err
	}
	n := &Node{Page: *idx, Section: true}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		switch {
		case e.IsDir() && exists(filepath.Join(p, "_index.md")):
			child, err := LoadSection(p)
			if err != nil {
				return nil, err
			}
			if child != nil {
				n.Children = append(n.Children, child)
			}
		case e.IsDir() && exists(filepath.Join(p, "index.md")): // page bundle
			pg, err := LoadPage(filepath.Join(p, "index.md"))
			if err != nil {
				return nil, err
			}
			if pg != nil {
				n.Children = append(n.Children, &Node{Page: *pg})
			}
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "_index.md":
			pg, err := LoadPage(p)
			if err != nil {
				return nil, err
			}
			if pg != nil {
				n.Children = append(n.Children, &Node{Page: *pg})
			}
		}
	}
	slices.SortStableFunc(n.Children, func(a, b *Node) int {
		return cmp.Or(cmp.Compare(a.Weight, b.Weight), cmp.Compare(a.Path, b.Path))
	})
	return n, nil
}

// LoadPage lit une page et son front matter YAML (clés simples uniquement).
// Elle renvoie nil pour une page en brouillon ou marquée « pdf: false ».
func LoadPage(path string) (*Page, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := string(raw)
	if !strings.HasPrefix(s, "---\n") {
		return nil, fmt.Errorf("%s : front matter manquant", path)
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("%s : front matter non terminé", path)
	}
	p := &Page{Path: path, Dir: filepath.Dir(path), Body: s[4+end+5:]}
	for line := range strings.SplitSeq(s[4:4+end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "title":
			if u, err := strconv.Unquote(v); err == nil {
				v = u
			} else {
				v = strings.Trim(v, `'`)
			}
			p.Title = v
		case "description":
			if u, err := strconv.Unquote(v); err == nil {
				v = u
			}
			p.Description = v
		case "weight":
			p.Weight, _ = strconv.Atoi(v)
		case "draft", "pdf":
			if (k == "draft") == (v == "true") {
				return nil, nil
			}
		}
	}
	if p.Title == "" {
		return nil, fmt.Errorf("%s : titre manquant", path)
	}
	normalise(p)
	return p, nil
}

var (
	// Numéro en tête du champ title : « II. », « II.1 », « A. ».
	numeroTitreRe = regexp.MustCompile(`^([IVXLC]+\.(\d+)?|[A-Z]\.) `)
	// Numéro en tête du titre de niveau 1 : « Chapitre II.1 — ».
	numeroH1Re = regexp.MustCompile(`^(Partie [IVXLC]+|Chapitre [IVXLC]+\.\d+|Annexe [A-Z]) — `)
	// Sommaire tenu à jour par numerote, inutile au livre et aux diapositives.
	sommaireRe = regexp.MustCompile(`(?s)\n*<!-- sommaire -->.*?<!-- /sommaire -->\n*`)
)

// normalise retire la numérotation du site, que le livre PDF et les
// diapositives calculent eux-mêmes : le numéro du titre, le titre de niveau 1
// qui ouvre la page (le titre complet, qui remplace alors Title) et les
// sommaires.
func normalise(p *Page) {
	p.Title = numeroTitreRe.ReplaceAllString(p.Title, "")
	corps := strings.TrimLeft(p.Body, "\n")
	if ligne, reste, _ := strings.Cut(corps, "\n"); strings.HasPrefix(ligne, "# ") {
		p.Title = numeroH1Re.ReplaceAllString(strings.TrimSpace(ligne[2:]), "")
		p.Body = strings.TrimLeft(reste, "\n")
	}
	p.Body = sommaireRe.ReplaceAllString(p.Body, "\n\n")
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Romain écrit n en chiffres romains (1 à 3999) : les parties du livre sont
// numérotées ainsi (« V - Approfondissements »).
func Romain(n int) string {
	valeurs := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	symboles := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range valeurs {
		for n >= v {
			b.WriteString(symboles[i])
			n -= v
		}
	}
	return b.String()
}
