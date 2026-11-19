// Command mkbook assemble les pages du site Hugo en un seul fichier Markdown
// destiné à pandoc. L'ordre des parties et des chapitres suit le champ
// « weight » du front matter, exactement comme sur le site.
package main

import (
	"bufio"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func main() {
	content := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	static := flag.String("static", "site/static", "répertoire des fichiers statiques Hugo")
	out := flag.String("out", "book/build/book.md", "fichier Markdown à produire")
	flag.Parse()

	root, err := loadSection(*content)
	if err != nil {
		log.Fatal(err)
	}
	var b strings.Builder
	w := writer{b: &b, static: *static}
	for _, c := range root.children {
		w.node(c, 1)
	}
	if w.err != nil {
		log.Fatal(w.err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s : %d pages\n", *out, w.pages)
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

type writer struct {
	b      *strings.Builder
	static string
	pages  int
	err    error
}

var (
	headingRe   = regexp.MustCompile(`^(#{1,6}) `)
	imageRe     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)`)
	linkRe      = regexp.MustCompile(`(^|[^!])\[([^\]]+)\]\(([^)]+)\)`)
	shortcodeRe = regexp.MustCompile(`\{\{[<%]`)
)

// node écrit une section ou une page. Une section de profondeur d devient un
// titre de niveau d ; une page reçoit un titre de niveau d et son contenu,
// qui commence au niveau 2, est décalé pour rester sous ce titre.
func (w *writer) node(n *node, depth int) {
	fmt.Fprintf(w.b, "%s %s\n\n", strings.Repeat("#", depth), n.title)
	w.body(n.page, depth-1)
	w.pages++
	for _, c := range n.children {
		w.node(c, depth+1)
	}
}

func (w *writer) body(p page, shift int) {
	sc := bufio.NewScanner(strings.NewReader(p.body))
	inCode := false
	for line := 1; sc.Scan(); line++ {
		l := sc.Text()
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
		}
		if !inCode {
			if shortcodeRe.MatchString(l) {
				w.err = errors.Join(w.err, fmt.Errorf("%s:%d : shortcode Hugo interdit dans le livre", p.path, line))
			}
			if m := headingRe.FindStringSubmatch(l); m != nil {
				level := min(len(m[1])+shift, 6)
				l = strings.Repeat("#", level) + l[len(m[1]):]
			}
			l = imageRe.ReplaceAllStringFunc(l, func(s string) string {
				m := imageRe.FindStringSubmatch(s)
				return "![" + m[1] + "](" + w.resolve(p.dir, m[2])
			})
			// Les liens internes n'ont pas de sens dans le PDF : on garde le texte.
			l = linkRe.ReplaceAllStringFunc(l, func(s string) string {
				m := linkRe.FindStringSubmatch(s)
				if strings.Contains(m[3], "://") || strings.HasPrefix(m[3], "mailto:") {
					return s
				}
				return m[1] + m[2]
			})
		}
		w.b.WriteString(l)
		w.b.WriteByte('\n')
	}
	w.b.WriteByte('\n')
}

// resolve convertit le chemin d'une image tel que Hugo le comprend en chemin
// relatif à la racine du dépôt, là où pandoc est lancé.
func (w *writer) resolve(dir, src string) string {
	switch {
	case strings.Contains(src, "://"):
		return src
	case strings.HasPrefix(src, "/"):
		return filepath.ToSlash(filepath.Join(w.static, src))
	default:
		return filepath.ToSlash(filepath.Join(dir, src))
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
