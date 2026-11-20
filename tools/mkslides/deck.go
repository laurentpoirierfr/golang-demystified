package main

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

// Limits that keep a slide readable, measured on the rendered slides : the
// content area holds about budget lines of text of 70 characters ; a line of
// code weighs 0.62 line of text, a subtitle one line. Marp shrinks code that
// is too wide (auto-scaling), but never code that is too long.
const (
	budget       = 13
	maxItems     = 7
	maxCodeLines = 14 // in the deck of a chapter
	maxKeyLines  = 11 // in the key slides of a rubrique, under a subtitle and a sentence
	maxTableRows = 8
	maxCell      = 70
	maxSentence  = 190
)

var (
	linkRe     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	examplesRe = regexp.MustCompile(`https://github\.com/laurentpoirierfr/golang-demystified/tree/main/examples/[^)\s]+`)
	numberedRe = regexp.MustCompile(`^(\d+)\. (.*)$`)
	labelRe    = regexp.MustCompile(`^\*\*[^*]+\*\*:?$`)
)

var alertLabels = map[string]string{
	"NOTE": "Note", "TIP": "Astuce", "IMPORTANT": "Important", "WARNING": "Attention", "CAUTION": "Prudence",
}

// chapter describes the chapter a deck is made from.
type chapter struct {
	Part, Title string
	URL         string // the chapter on the site
	Edition     string // « Go 1.27 »
}

// slide is one slide: a title, a subtitle and Markdown content.
type slide struct {
	title, subtitle string
	class           string
	body            []string // Markdown fragments, separated by blank lines
	weight          float64
}

// deck builds the Marp Markdown of a chapter.
func deck(ch chapter, md string) string {
	blocks := parse(md)
	var slides []slide

	slides = append(slides, slide{class: "titre", title: ch.Title, subtitle: ch.Part,
		body: []string{"Golang Demystified · Édition " + ch.Edition}})

	// The introduction: what comes before the first section.
	first := len(blocks)
	for i, b := range blocks {
		if b.kind == heading && b.level == 2 {
			first = i
			break
		}
	}
	if intro := summarize("En bref", "", blocks[:first]); len(intro) > 0 {
		slides = append(slides, intro...)
	}

	// The outline, then the sections.
	var outline []string
	var sections [][]block
	for i := first; i < len(blocks); i++ {
		if b := blocks[i]; b.kind == heading && b.level == 2 {
			outline = append(outline, "- "+inline(b.text))
			sections = append(sections, nil)
			continue
		}
		sections[len(sections)-1] = append(sections[len(sections)-1], blocks[i])
	}
	if len(outline) > 1 {
		slides = append(slides, slide{title: "Au programme", class: "programme", body: []string{strings.Join(outline, "\n")}})
	}
	for i, sec := range sections {
		title := strings.TrimPrefix(outline[i], "- ")
		// Each subsection (###) gets its own slides.
		start := 0
		sub := ""
		for j := 0; j <= len(sec); j++ {
			if j < len(sec) && (sec[j].kind != heading || sec[j].level != 3) {
				continue
			}
			slides = append(slides, summarize(title, sub, sec[start:j])...)
			if j < len(sec) {
				sub, start = inline(sec[j].text), j+1
			}
		}
	}

	// To go further: the chapter, its examples, the book.
	more := []string{fmt.Sprintf("- Le chapitre complet : [%s](%s)", ch.Title, ch.URL)}
	if ex := examplesRe.FindString(md); ex != "" {
		more = append(more, fmt.Sprintf("- Le code des exemples : [%s](%s)", strings.TrimPrefix(ex, "https://github.com/laurentpoirierfr/golang-demystified/tree/main/"), ex))
	}
	more = append(more, "- Le livre en PDF et en EPUB : [github.com/laurentpoirierfr/golang-demystified](https://github.com/laurentpoirierfr/golang-demystified/releases/latest)")
	slides = append(slides, slide{title: "Pour aller plus loin", class: "fin", body: []string{strings.Join(more, "\n")}})

	return render(ch, slides)
}

// summarize turns the blocks of a section into one or more slides: the
// first sentence of the text, the lists, the first table, the first code
// block and the first alert, cut into slides that fit the budget.
func summarize(title, sub string, blocks []block) []slide {
	type part struct {
		md     string
		weight float64
	}
	var parts []part
	sentence, tables, codes, alerts := false, 0, 0, 0
	for _, b := range blocks {
		switch b.kind {
		case paragraph:
			// Un paragraphe tout en gras est une étiquette (« **Construire un
			// service** ») : il reste au-dessus de la liste qui le suit.
			if labelRe.MatchString(strings.TrimSpace(b.text)) {
				parts = append(parts, part{inline(strings.TrimSpace(b.text)), 1.1})
				continue
			}
			if !sentence {
				s := firstSentence(inline(b.text))
				parts = append(parts, part{s, lines(s)})
				sentence = true
			}
		case list:
			// A long list is cut into chunks, which the pagination below
			// spreads over several slides.
			items := b.items
			more := len(items) > maxItems
			if more {
				items = items[:maxItems]
			}
			var chunk []string
			flush := func() {
				if len(chunk) > 0 {
					l, w := shrinkList(chunk)
					parts = append(parts, part{l, w})
					chunk = nil
				}
			}
			for _, it := range items {
				if _, w := shrinkList(append(chunk, it)); w > budget-subtitleWeight-1 {
					flush()
				}
				chunk = append(chunk, it)
			}
			flush()
			if more {
				parts[len(parts)-1].md += "\n- …"
				parts[len(parts)-1].weight += 1.3
			}
		case table:
			// The first table, in full : a long one is spread over up to
			// three slides, its header repeated.
			if tables == 0 {
				for _, rows := range chunkTable(b.items, maxTableRows, 3) {
					t, w := shrinkTable(rows, maxTableRows)
					parts = append(parts, part{t, w})
				}
			}
			tables++
		case code:
			if codes == 0 && len(b.items) > 0 {
				c, w := shrinkCode(b, maxCodeLines)
				parts = append(parts, part{c, w})
			}
			codes++
		case alert:
			if alerts == 0 {
				s := "**" + alertLabels[b.alert] + "** : " + firstSentence(inline(b.text))
				parts = append(parts, part{"> " + s, alertWeight(s)})
			}
			alerts++
		}
	}
	// Une étiquette reste avec ce qui la suit : jamais seule en bas d'une
	// diapositive.
	for i := 0; i < len(parts)-1; i++ {
		if labelRe.MatchString(parts[i].md) {
			parts[i] = part{parts[i].md + "\n\n" + parts[i+1].md, parts[i].weight + parts[i+1].weight}
			parts = slices.Delete(parts, i+1, i+2)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	var slides []slide
	cur := slide{title: title, subtitle: sub}
	if sub != "" {
		cur.weight = subtitleWeight
	}
	for _, p := range parts {
		if len(cur.body) > 0 && cur.weight+p.weight > budget {
			slides = append(slides, cur)
			cur = slide{title: title, subtitle: strings.TrimSpace(sub + " (suite)"), weight: subtitleWeight}
		}
		cur.body = append(cur.body, p.md)
		cur.weight += p.weight
	}
	return append(slides, cur)
}

// Heights, in lines of text, measured on the rendered slides.
const subtitleWeight = 1.1

// lines estimates the height of a paragraph : 85 characters per line.
func lines(s string) float64 { return math.Ceil(float64(utf8.RuneCountInString(s)) / 85) }

// alertWeight : smaller text, with a margin and a padding.
func alertWeight(s string) float64 {
	return 0.85*math.Ceil(float64(utf8.RuneCountInString(s))/100) + 0.8
}

// firstSentence keeps the first sentence of a text (two if the first one is
// very short), without the colon that introduces a code block.
func firstSentence(s string) string {
	end := -1
	for i := 0; i < len(s); i++ {
		if (s[i] == '.' || s[i] == '!' || s[i] == '?') && (i+1 == len(s) || s[i+1] == ' ') && !inCode(s, i) {
			if i >= 40 || end >= 0 {
				end = i + 1
				break
			}
			end = i + 1
		}
	}
	if end > 0 {
		s = s[:end]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), " :")
	s = strings.TrimSuffix(s, ":")
	if utf8.RuneCountInString(s) > maxSentence {
		s = cut(s, maxSentence)
	}
	return balance(s)
}

// inCode reports whether s[i] is inside an inline code span.
func inCode(s string, i int) bool { return strings.Count(s[:i], "`")%2 == 1 }

// cut shortens s to about n characters, at a word boundary, never inside a
// link : a link that straddles the limit is kept whole if it ends soon
// enough, and dropped otherwise.
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	t := string(r[:n])
	for _, l := range linkRe.FindAllStringIndex(s, -1) {
		if l[0] < len(t) && len(t) < l[1] {
			if utf8.RuneCountInString(s[:l[1]]) <= n+100 {
				return strings.TrimRight(s[:l[1]], " ,;:") + "…"
			}
			t = s[:l[0]]
			break
		}
	}
	if k := strings.LastIndex(t, " "); k > n/2 {
		t = t[:k]
	}
	return strings.TrimRight(t, " ,;:") + "…"
}

// balance closes the code spans and bold markers a cut left open.
func balance(s string) string {
	if strings.Count(s, "`")%2 == 1 {
		s += "`"
	}
	if strings.Count(s, "**")%2 == 1 {
		s += "**"
	}
	return s
}

// inline cleans inline Markdown for a slide: internal links become their text,
// and dollar signs are escaped outside code (Marp would read them as math).
func inline(s string) string {
	s = linkRe.ReplaceAllStringFunc(s, func(l string) string {
		m := linkRe.FindStringSubmatch(l)
		if strings.Contains(m[2], "://") {
			return l
		}
		return m[1]
	})
	parts := strings.Split(s, "`")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = strings.ReplaceAll(parts[i], "$", `\$`)
	}
	return strings.Join(parts, "`")
}

func shrinkCode(b block, max int) (string, float64) {
	src := b.items
	if len(src) > max {
		comment := "// …"
		switch b.lang {
		case "yaml", "bash", "sh", "toml", "dockerfile", "console", "makefile":
			comment = "# …"
		case "sql":
			comment = "-- …"
		case "", "text", "json":
			comment = "…"
		}
		src = append(src[:max:max], comment)
	}
	fence := "```"
	if strings.Contains(strings.Join(src, "\n"), "```") {
		fence = "````"
	}
	return fence + b.lang + "\n" + strings.Join(src, "\n") + "\n" + fence, 1 + 0.62*float64(len(src))
}

func shrinkTable(rows []string, max int) (string, float64) {
	var out []string
	w := 0.6 // margins
	cols := max1(strings.Count(rows[0], "|") - 1)
	perCol := 118 / cols // characters per column at the size of tables
	for k, r := range rows {
		if k == max+2 { // header, separator, rows
			out = append(out, strings.Repeat("| … ", strings.Count(rows[0], "|")-1)+"|")
			w++
			break
		}
		cells := strings.Split(strings.Trim(r, "|"), "|")
		for c := range cells {
			cells[c] = " " + balance(cut(inline(strings.TrimSpace(cells[c])), maxCell)) + " "
			if k == 1 { // separator row
				cells[c] = strings.ReplaceAll(cells[c], "…", "")
			}
		}
		out = append(out, "|"+strings.Join(cells, "|")+"|")
		if k != 1 { // a row is as high as its longest cell
			wrap := 1.0
			for _, c := range cells {
				wrap = math.Max(wrap, math.Ceil(float64(utf8.RuneCountInString(strings.TrimSpace(c)))/float64(perCol)))
			}
			w += 0.3 + 0.7*wrap
		}
	}
	return strings.Join(out, "\n"), w
}

// render writes the Marp Markdown.
func render(ch chapter, slides []slide) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nmarp: true\ntheme: golang\npaginate: true\nlang: fr\ntitle: %q\nheader: %q\nfooter: %q\n---\n",
		ch.Title+" · Golang Demystified", "Golang Demystified · "+ch.Part, ch.Title)
	for i, s := range slides {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		b.WriteString("\n")
		if s.class != "" {
			fmt.Fprintf(&b, "<!-- _class: %s -->\n", s.class)
		}
		if s.class == "titre" {
			b.WriteString("<!-- _paginate: skip -->\n<!-- _header: \"\" -->\n<!-- _footer: \"\" -->\n\n")
			fmt.Fprintf(&b, "# %s\n\n## %s\n\n%s\n", inline(s.title), inline(s.subtitle), strings.Join(s.body, "\n\n"))
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", s.title)
		if s.subtitle != "" {
			fmt.Fprintf(&b, "### %s\n\n", s.subtitle)
		}
		b.WriteString(strings.Join(s.body, "\n\n"))
		b.WriteString("\n")
	}
	return b.String()
}

// section is a level-2 section of a chapter: its title and its blocks.
type section struct {
	title  string
	blocks []block
}

// sections cuts a chapter into its introduction and its sections.
func sections(blocks []block) (intro []block, secs []section) {
	for _, b := range blocks {
		switch {
		case b.kind == heading && b.level == 2:
			secs = append(secs, section{title: inline(b.text)})
		case len(secs) == 0:
			intro = append(intro, b)
		default:
			secs[len(secs)-1].blocks = append(secs[len(secs)-1].blocks, b)
		}
	}
	return intro, secs
}

// part describes a rubrique and its chapters, for its deck.
type rubrique struct {
	Number   int
	Title    string
	URL      string
	Edition  string
	Intro    string // Markdown of the rubrique's index page
	Chapters []rubriqueChapter
}

type rubriqueChapter struct {
	Number int
	Title  string
	Body   string
}

// rubriqueDeck builds the deck of a rubrique: the rubrique, its outline, then
// for each chapter a divider slide and up to two key slides. Short enough to
// be presented, it complements the detailed deck of each chapter.
func rubriqueDeck(r rubrique) string {
	ch := chapter{Part: content.Romain(r.Number) + " - " + r.Title, Title: r.Title, URL: r.URL, Edition: r.Edition}
	slides := []slide{{class: "titre", title: r.Title, subtitle: "Rubrique " + content.Romain(r.Number),
		body: []string{"Golang Demystified · Édition " + r.Edition}}}
	slides = append(slides, summarize("En bref", "", parse(r.Intro))...)

	var outline []string
	for _, c := range r.Chapters {
		outline = append(outline, fmt.Sprintf("%d. %s", c.Number, inline(c.Title)))
	}
	slides = append(slides, slide{title: "Au programme", class: "programme", body: []string{strings.Join(outline, "\n")}})

	for _, c := range r.Chapters {
		intro, secs := sections(parse(c.Body))
		// The divider: the chapter, its first sentence, its sections.
		divider := slide{class: "chapitre", title: fmt.Sprintf("%d. %s", c.Number, inline(c.Title))}
		if s := firstParagraph(intro, secs); s != "" {
			divider.body = append(divider.body, s)
		}
		var names []string
		for k, s := range secs {
			if k == 6 {
				names = append(names, "- …")
				break
			}
			names = append(names, "- "+s.title)
		}
		if len(names) > 0 {
			divider.body = append(divider.body, strings.Join(names, "\n"))
		}
		slides = append(slides, divider)
		slides = append(slides, keySlides(c.Title, secs)...)
	}

	more := []string{
		fmt.Sprintf("- La rubrique sur le site : [%s](%s)", r.Title, r.URL),
		"- Une présentation détaillée par chapitre : le répertoire voisin, du même numéro",
		"- Le livre en PDF et en EPUB : [github.com/laurentpoirierfr/golang-demystified](https://github.com/laurentpoirierfr/golang-demystified/releases/latest)",
	}
	slides = append(slides, slide{title: "Pour aller plus loin", class: "fin", body: []string{strings.Join(more, "\n")}})
	return render(ch, slides)
}

// firstParagraph returns the first sentence of the chapter: its
// introduction, or else its first section.
func firstParagraph(intro []block, secs []section) string {
	all := intro
	if len(secs) > 0 {
		all = append(all[:len(all):len(all)], secs[0].blocks...)
	}
	for _, b := range all {
		if b.kind == paragraph {
			return firstSentence(inline(b.text))
		}
	}
	return ""
}

// keySlides picks up to two slides that show the chapter at a glance: its
// first table and its first code example, or, failing that, its first list.
func keySlides(title string, secs []section) []slide {
	var out []slide
	used := map[int]bool{}
	pick := func(k kind) {
		if len(out) == 2 {
			return
		}
		for i, s := range secs {
			if used[i] {
				continue
			}
			for _, b := range s.blocks {
				if b.kind != k || (k == code && len(b.items) == 0) {
					continue
				}
				sentence := ""
				for _, p := range s.blocks {
					if p.kind == paragraph {
						sentence = firstSentence(inline(p.text))
						break
					}
				}
				out = append(out, slide{title: inline(title), subtitle: s.title, body: fit(b, sentence)})
				used[i] = true
				return
			}
		}
	}
	pick(table)
	pick(code)
	if len(out) < 2 {
		pick(list)
	}
	return out
}

// shrinkList keeps the first sentence of the first items of a list.
func shrinkList(items []string) (string, float64) {
	var out []string
	w := 0.0
	for k, it := range items {
		if k == maxItems {
			out = append(out, "- …")
			w++
			break
		}
		puce := "- "
		if m := numberedRe.FindStringSubmatch(it); m != nil { // « 3. texte » : liste numérotée
			puce, it = m[1]+". ", m[2]
		}
		s := firstSentence(inline(it))
		out = append(out, puce+s)
		w += math.Ceil(float64(utf8.RuneCountInString(s))/80) + 0.3 // indented, with a margin
	}
	return strings.Join(out, "\n"), w
}

func max1(n int) int { return max(n, 1) }

// fit builds the body of a key slide, under its title and subtitle: a
// sentence, then a table, a code block or a list. If it is too high, the
// sentence goes first, then rows, lines or items, until it fits.
func fit(b block, sentence string) []string {
	room := budget - subtitleWeight
	limit := map[kind]int{table: maxTableRows, code: maxKeyLines, list: maxItems}[b.kind]
	for {
		var md string
		var w float64
		switch b.kind {
		case table:
			md, w = shrinkTable(b.items, limit)
		case code:
			md, w = shrinkCode(b, limit)
		case list:
			items := b.items
			if len(items) > limit {
				items = items[:limit]
			}
			md, w = shrinkList(items)
		}
		switch {
		case sentence != "" && w+lines(sentence) <= room:
			return []string{sentence, md}
		case w <= room || limit <= 2:
			return []string{md}
		case sentence != "":
			sentence = "" // the example matters more than the sentence
		default:
			limit--
		}
	}
}

// chunkTable cuts a table into tables of at most size rows, each with the
// header and separator, at most max of them : the last one then ends with
// the ellipsis row of shrinkTable.
func chunkTable(rows []string, size, max int) [][]string {
	if len(rows) < 2 {
		return [][]string{rows}
	}
	head, body := rows[:2], rows[2:]
	// Balanced chunks : 9 rows make 5 + 4, not 8 + 1.
	if chunks := (len(body) + size - 1) / size; chunks > 1 && chunks <= max {
		size = (len(body) + chunks - 1) / chunks
	}
	var out [][]string
	for len(body) > 0 && len(out) < max {
		n := min(size, len(body))
		if len(out) == max-1 {
			n = len(body) // the last chunk keeps the rest, shrinkTable cuts it
		}
		out = append(out, append(append([]string{}, head...), body[:n]...))
		body = body[n:]
	}
	return out
}
