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
	"slices"
	"strconv"
	"strings"
)

// Page est une page du site.
type Page struct {
	Title  string
	Weight int
	Body   string // Markdown, sans le front matter
	Dir    string // répertoire de la page, pour résoudre les images relatives
	Path   string
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
	return p, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
