package main

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type page struct {
	title  string
	weight int
	body   string
	dir    string // répertoire de la page, pour résoudre les images relatives
	path   string
}

type node struct {
	page
	children []*node
	section  bool
}

// loadSection lit une section Hugo (un répertoire contenant _index.md).
func loadSection(dir string) (*node, error) {
	idx, err := loadPage(filepath.Join(dir, "_index.md"))
	if err != nil {
		return nil, err
	}
	n := &node{page: *idx, section: true}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		switch {
		case e.IsDir() && exists(filepath.Join(p, "_index.md")):
			child, err := loadSection(p)
			if err != nil {
				return nil, err
			}
			if child != nil {
				n.children = append(n.children, child)
			}
		case e.IsDir() && exists(filepath.Join(p, "index.md")): // page bundle
			pg, err := loadPage(filepath.Join(p, "index.md"))
			if err != nil {
				return nil, err
			}
			if pg != nil {
				n.children = append(n.children, &node{page: *pg})
			}
		case !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "_index.md":
			pg, err := loadPage(p)
			if err != nil {
				return nil, err
			}
			if pg != nil {
				n.children = append(n.children, &node{page: *pg})
			}
		}
	}
	slices.SortStableFunc(n.children, func(a, b *node) int {
		return cmp.Or(cmp.Compare(a.weight, b.weight), cmp.Compare(a.path, b.path))
	})
	return n, nil
}

// loadPage lit une page et son front matter YAML (clés simples uniquement).
// Elle renvoie nil pour une page en brouillon ou marquée « pdf: false ».
func loadPage(path string) (*page, error) {
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
	p := &page{path: path, dir: filepath.Dir(path), body: s[4+end+5:]}
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
			p.title = v
		case "weight":
			p.weight, _ = strconv.Atoi(v)
		case "draft", "pdf":
			if (k == "draft") == (v == "true") {
				return nil, nil
			}
		}
	}
	if p.title == "" {
		return nil, fmt.Errorf("%s : titre manquant", path)
	}
	return p, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
