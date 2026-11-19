package main

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	headingRe   = regexp.MustCompile(`^(#{1,6}) `)
	imageRe     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)`)
	linkRe      = regexp.MustCompile(`(^|[^!])\[([^\]]+)\]\(([^)]+)\)`)
	shortcodeRe = regexp.MustCompile(`\{\{[<%]`)
	alertRe     = regexp.MustCompile(`^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$`)
)

// converter transforme le Markdown d'une page Hugo en Markdown pour le livre.
type converter struct {
	static string // répertoire des fichiers statiques Hugo
	base   string // répertoire depuis lequel les chemins d'images sont résolus
	// callouts convertit les alertes GitHub (> [!NOTE]) en blocs Quarto.
	callouts bool
	err      error
}

// convert renvoie le corps de la page, avec ses titres décalés de shift
// niveaux, ses images résolues et ses liens internes réduits à leur texte.
func (c *converter) convert(p page, shift int) string {
	var b strings.Builder
	sc := bufio.NewScanner(strings.NewReader(p.body))
	inCode, inCallout := false, false
	for line := 1; sc.Scan(); line++ {
		l := sc.Text()

		if c.callouts {
			if m := alertRe.FindStringSubmatch(l); m != nil && !inCode {
				fmt.Fprintf(&b, "::: {.callout-%s}\n", strings.ToLower(m[1]))
				inCallout = true
				continue
			}
			if inCallout {
				if rest, ok := strings.CutPrefix(l, ">"); ok {
					l = strings.TrimPrefix(rest, " ")
				} else {
					b.WriteString(":::\n")
					inCallout = false
				}
			}
		}

		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inCode = !inCode
		}
		if !inCode {
			if shortcodeRe.MatchString(l) {
				c.err = errors.Join(c.err, fmt.Errorf("%s:%d : shortcode Hugo interdit dans le livre", p.path, line))
			}
			if m := headingRe.FindStringSubmatch(l); m != nil {
				level := min(len(m[1])+shift, 6)
				l = strings.Repeat("#", level) + l[len(m[1]):]
			}
			l = imageRe.ReplaceAllStringFunc(l, func(s string) string {
				m := imageRe.FindStringSubmatch(s)
				return "![" + m[1] + "](" + c.resolve(p.dir, m[2])
			})
			// Les liens internes n'ont pas de sens dans un livre : on garde le texte.
			l = linkRe.ReplaceAllStringFunc(l, func(s string) string {
				m := linkRe.FindStringSubmatch(s)
				if strings.Contains(m[3], "://") || strings.HasPrefix(m[3], "mailto:") {
					return s
				}
				return m[1] + m[2]
			})
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	if inCallout {
		b.WriteString(":::\n")
	}
	return b.String()
}

// resolve convertit le chemin d'une image tel que Hugo le comprend en chemin
// relatif au répertoire base.
func (c *converter) resolve(dir, src string) string {
	var p string
	switch {
	case strings.Contains(src, "://"):
		return src
	case strings.HasPrefix(src, "/"):
		p = filepath.Join(c.static, src)
	default:
		p = filepath.Join(dir, src)
	}
	if rel, err := filepath.Rel(c.base, p); err == nil {
		p = rel
	}
	return filepath.ToSlash(p)
}
