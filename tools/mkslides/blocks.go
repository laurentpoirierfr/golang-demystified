package main

import (
	"regexp"
	"strings"
)

// kind is the kind of a Markdown block.
type kind int

const (
	paragraph kind = iota
	heading
	list
	code
	table
	alert
	quote
)

// block is a top-level Markdown block of a chapter.
type block struct {
	kind  kind
	level int      // heading level
	text  string   // heading text, paragraph or quote text (lines joined)
	items []string // list items (first level only), table rows, code lines
	lang  string   // code language
	alert string   // NOTE, TIP, IMPORTANT, WARNING, CAUTION
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6}) +(.*?) *#*$`)
	itemRe    = regexp.MustCompile(`^([-*+]|\d+[.)]) +(.*)$`)
	alertRe   = regexp.MustCompile(`^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$`)
)

func isFence(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), "```") }

// parse cuts a chapter into top-level blocks. It is not a full CommonMark
// parser: it understands the subset the book uses, and ignores what it does
// not need for slides (nested lists, code nested in list items).
func parse(md string) []block {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var blocks []block
	for i := 0; i < len(lines); {
		l := lines[i]
		switch {
		case strings.TrimSpace(l) == "":
			i++

		case isFence(l) && !strings.HasPrefix(l, " "):
			b := block{kind: code, lang: strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "`"))}
			for i++; i < len(lines) && !isFence(lines[i]); i++ {
				b.items = append(b.items, lines[i])
			}
			i++ // closing fence
			blocks = append(blocks, b)

		case headingRe.MatchString(l):
			m := headingRe.FindStringSubmatch(l)
			blocks = append(blocks, block{kind: heading, level: len(m[1]), text: m[2]})
			i++

		case alertRe.MatchString(l):
			b := block{kind: alert, alert: alertRe.FindStringSubmatch(l)[1]}
			var text []string
			for i++; i < len(lines) && strings.HasPrefix(lines[i], ">"); i++ {
				text = append(text, strings.TrimSpace(strings.TrimPrefix(lines[i], ">")))
			}
			b.text = strings.Join(text, " ")
			blocks = append(blocks, b)

		case strings.HasPrefix(l, ">"):
			var text []string
			for ; i < len(lines) && strings.HasPrefix(lines[i], ">"); i++ {
				text = append(text, strings.TrimSpace(strings.TrimPrefix(lines[i], ">")))
			}
			blocks = append(blocks, block{kind: quote, text: strings.Join(text, " ")})

		case strings.HasPrefix(l, "|"):
			b := block{kind: table}
			for ; i < len(lines) && strings.HasPrefix(lines[i], "|"); i++ {
				b.items = append(b.items, lines[i])
			}
			blocks = append(blocks, b)

		case itemRe.MatchString(l):
			b := block{kind: list}
			i = parseList(lines, i, &b)
			blocks = append(blocks, b)

		default:
			var text []string
			for ; i < len(lines) && startsParagraph(lines[i]); i++ {
				text = append(text, strings.TrimSpace(lines[i]))
			}
			if len(text) == 0 { // an indented line out of any context
				i++
				continue
			}
			blocks = append(blocks, block{kind: paragraph, text: strings.Join(text, " ")})
		}
	}
	return blocks
}

// startsParagraph reports whether l continues a paragraph.
func startsParagraph(l string) bool {
	return strings.TrimSpace(l) != "" && !isFence(l) && !headingRe.MatchString(l) &&
		!strings.HasPrefix(l, ">") && !strings.HasPrefix(l, "|") && !itemRe.MatchString(l)
}

// parseList reads a list starting at lines[i] and returns the index of the
// first line after it. Only the first line (or lines) of each first-level item
// is kept, up to the first nested element.
func parseList(lines []string, i int, b *block) int {
	inFence := false
	keep := false // still in the first paragraph of the current item
	for i < len(lines) {
		l := lines[i]
		switch {
		case inFence:
			if isFence(l) {
				inFence = false
			}
		case isFence(l):
			inFence, keep = true, false
		case itemRe.MatchString(l): // new first-level item
			b.items = append(b.items, itemRe.FindStringSubmatch(l)[2])
			keep = true
		case strings.TrimSpace(l) == "":
			keep = false
			// The list goes on if the next non-blank line is indented or is
			// another item.
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || !strings.HasPrefix(lines[j], " ") && !itemRe.MatchString(lines[j]) {
				return j
			}
		case strings.HasPrefix(l, " "):
			t := strings.TrimSpace(l)
			if keep && !itemRe.MatchString(t) {
				b.items[len(b.items)-1] += " " + t
			} else {
				keep = false // nested list or second paragraph
			}
		default:
			return i
		}
		i++
	}
	return i
}
