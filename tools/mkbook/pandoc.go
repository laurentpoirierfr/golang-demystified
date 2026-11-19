package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// writePandoc assemble tout le livre en un seul fichier Markdown pour pandoc.
// Une section de profondeur d devient un titre de niveau d ; une page reçoit un
// titre de niveau d et son contenu, qui commence au niveau 2, est décalé pour
// rester sous ce titre. Les chemins d'images sont relatifs à la racine du
// dépôt, où pandoc est lancé.
func writePandoc(root *node, static, out string) (int, error) {
	c := &converter{static: static, base: "."}
	var b strings.Builder
	pages := 0
	var walk func(n *node, depth int)
	walk = func(n *node, depth int) {
		fmt.Fprintf(&b, "%s %s\n\n", strings.Repeat("#", depth), n.title)
		b.WriteString(c.convert(n.page, depth-1))
		b.WriteByte('\n')
		pages++
		for _, child := range n.children {
			walk(child, depth+1)
		}
	}
	for _, n := range root.children {
		walk(n, 1)
	}
	if c.err != nil {
		return 0, c.err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return 0, err
	}
	return pages, os.WriteFile(out, []byte(b.String()), 0o644)
}
