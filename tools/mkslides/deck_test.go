package main

import (
	"fmt"
	"strings"
	"testing"
)

const exemple = `> [!NOTE]
> Les exemples sont testés : [` + "`examples/langage/interfaces`" + `](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/interfaces).

## Un contrat de comportement

Une interface décrit un **ensemble de méthodes**. Tout type qui les possède la
satisfait, implicitement.

` + "```go" + `
type Forme interface {
	Aire() float64
}
` + "```" + `

## Les petites interfaces

| Interface | Méthode |
|-----------|---------|
| ` + "`io.Reader`" + ` | ` + "`Read`" + ` |
| ` + "`fmt.Stringer`" + ` | ` + "`String`" + ` |

### Composer

- **io.ReadWriter** : un lecteur et un écrivain. La suite est ignorée.
- Voir [le chapitre](../structures/) sur l'intégration.
`

func Example_deck() {
	md := deck(chapter{Part: "2. Le langage", Title: "Interfaces", URL: "https://exemple.org/docs/langage/interfaces/", Edition: "Go 1.27"}, exemple)
	fmt.Print(md)
	// Output:
	// ---
	// marp: true
	// theme: golang
	// paginate: true
	// lang: fr
	// title: "Interfaces · Golang Demystified"
	// header: "Golang Demystified · 2. Le langage"
	// footer: "Interfaces"
	// ---
	//
	// <!-- _class: titre -->
	// <!-- _paginate: skip -->
	// <!-- _header: "" -->
	// <!-- _footer: "" -->
	//
	// # Interfaces
	//
	// ## 2. Le langage
	//
	// Golang Demystified · Édition Go 1.27
	//
	// ---
	//
	// ## En bref
	//
	// > **Note** : Les exemples sont testés : [`examples/langage/interfaces`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/interfaces).
	//
	// ---
	//
	// <!-- _class: programme -->
	// ## Au programme
	//
	// - Un contrat de comportement
	// - Les petites interfaces
	//
	// ---
	//
	// ## Un contrat de comportement
	//
	// Une interface décrit un **ensemble de méthodes**.
	//
	// ```go
	// type Forme interface {
	// 	Aire() float64
	// }
	// ```
	//
	// ---
	//
	// ## Les petites interfaces
	//
	// | Interface | Méthode |
	// | ----------- | --------- |
	// | `io.Reader` | `Read` |
	// | `fmt.Stringer` | `String` |
	//
	// ---
	//
	// ## Les petites interfaces
	//
	// ### Composer
	//
	// - **io.ReadWriter** : un lecteur et un écrivain.
	// - Voir le chapitre sur l'intégration.
	//
	// ---
	//
	// <!-- _class: fin -->
	// ## Pour aller plus loin
	//
	// - Le chapitre complet : [Interfaces](https://exemple.org/docs/langage/interfaces/)
	// - Le code des exemples : [examples/langage/interfaces](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/interfaces)
	// - Le livre en PDF et en EPUB : [github.com/laurentpoirierfr/golang-demystified](https://github.com/laurentpoirierfr/golang-demystified/releases/latest)

}

func TestFirstSentence(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Une phrase complète. Une autre.", "Une phrase complète. Une autre."}, // trop courte : deux phrases
		{"Le package `fmt` formate des valeurs selon des verbes. La suite.", "Le package `fmt` formate des valeurs selon des verbes."},
		{"La version 1.18 a introduit les génériques, longtemps attendus. Suite.", "La version 1.18 a introduit les génériques, longtemps attendus."},
		{"On écrit `x.y. z` puis on continue la phrase jusqu'au bout. Fin.", "On écrit `x.y. z` puis on continue la phrase jusqu'au bout."},
		{"Ce principe résume l'usage des interfaces en Go :", "Ce principe résume l'usage des interfaces en Go"},
	}
	for _, tt := range tests {
		if got := firstSentence(tt.in); got != tt.want {
			t.Errorf("firstSentence(%q)\n = %q\nwant %q", tt.in, got, tt.want)
		}
	}
}

func TestCutKeepsLinks(t *testing.T) {
	s := strings.Repeat("mot ", 40) + "[un lien](https://exemple.org/une/adresse/assez/longue) " + strings.Repeat("fin ", 30)
	got := cut(s, 170)
	if strings.Count(got, "[") != strings.Count(got, "](") || strings.Count(got, "(") != strings.Count(got, ")") {
		t.Errorf("lien coupé : %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("pas de points de suspension : %q", got)
	}
}

func TestInline(t *testing.T) {
	got := inline("Coût : 5 $ et 10 $, voir [la page](../autre/) et `$1` ou [Go](https://go.dev)")
	want := "Coût : 5 \\$ et 10 \\$, voir la page et `$1` ou [Go](https://go.dev)"
	if got != want {
		t.Errorf("inline\n = %q\nwant %q", got, want)
	}
}

func TestLongListSplit(t *testing.T) {
	var b strings.Builder
	b.WriteString("## Résumé\n\n")
	for i := range 7 {
		fmt.Fprintf(&b, "- Point %d : %s.\n", i, strings.Repeat("une explication assez longue ", 4))
	}
	md := deck(chapter{Title: "T"}, b.String())
	if n := strings.Count(md, "## Résumé"); n < 2 {
		t.Errorf("la liste aurait dû s'étaler sur plusieurs diapositives : %d", n)
	}
	if n := strings.Count(md, "- Point "); n != 7 {
		t.Errorf("%d points, attendu 7", n)
	}
}

func TestChunkTable(t *testing.T) {
	rows := []string{"| A |", "|---|"}
	for i := range 9 {
		rows = append(rows, fmt.Sprintf("| %d |", i))
	}
	var tailles []int
	for _, c := range chunkTable(rows, 8, 3) {
		if c[0] != "| A |" || c[1] != "|---|" {
			t.Errorf("en-tête manquant : %v", c)
		}
		tailles = append(tailles, len(c)-2)
	}
	if fmt.Sprint(tailles) != "[5 4]" {
		t.Errorf("morceaux : %v, attendu [5 4]", tailles)
	}
}

func TestParseListWithNestedCode(t *testing.T) {
	md := "- **Un** : premier\n  point.\n\n  ```go\n  code()\n  ```\n\n- Deux\n  - imbriqué\n\nAprès la liste.\n"
	blocks := parse(md)
	if len(blocks) != 2 || blocks[0].kind != list || blocks[1].kind != paragraph {
		t.Fatalf("blocs : %+v", blocks)
	}
	if fmt.Sprint(blocks[0].items) != "[**Un** : premier point. Deux]" {
		t.Errorf("éléments : %q", blocks[0].items)
	}
}

func TestRubriqueDeck(t *testing.T) {
	md := rubriqueDeck(rubrique{Number: 2, Title: "Le langage", URL: "https://exemple.org/docs/langage/", Edition: "Go 1.27",
		Intro: "Les bases du langage.",
		Chapters: []rubriqueChapter{
			{Number: 1, Title: "Interfaces", Body: exemple},
			{Number: 2, Title: "Vide", Body: "Rien à dire.\n"},
		}})
	for _, attendu := range []string{
		"## Rubrique II",
		"1. Interfaces\n2. Vide",                      // le plan
		"<!-- _class: chapitre -->\n## 1. Interfaces", // l'intercalaire
		"### Les petites interfaces",                  // diapositive clé : le tableau
		"### Un contrat de comportement",              // diapositive clé : le code
		"<!-- _class: chapitre -->\n## 2. Vide",
	} {
		if !strings.Contains(md, attendu) {
			t.Errorf("%q absent", attendu)
		}
	}
}

// Une étiquette en gras et une liste numérotée gardent leur forme.
func TestEtiquettesEtNumeros(t *testing.T) {
	md := deck(chapter{Title: "T"}, "## Parcours\n\nIntroduction du parcours.\n\n**Construire un service**\n\n3. [Une API](api/) : la base.\n4. Le client : en ligne de commande.\n\n- une puce\n")
	for _, attendu := range []string{
		"**Construire un service**\n\n3. Une API : la base.\n4. Le client : en ligne de commande.",
		"- une puce",
	} {
		if !strings.Contains(md, attendu) {
			t.Errorf("%q absent de :\n%s", attendu, md)
		}
	}
}

// Une étiquette n'est jamais séparée de sa liste par un changement de page.
func TestEtiquetteJamaisSeule(t *testing.T) {
	var b strings.Builder
	b.WriteString("## Parcours\n\n")
	for g := range 4 {
		fmt.Fprintf(&b, "**Groupe %d**\n\n", g)
		for i := range 4 {
			fmt.Fprintf(&b, "%d. Un élément assez long pour occuper une bonne partie de la ligne, voire davantage.\n", g*4+i+1)
		}
		b.WriteString("\n")
	}
	for diapo := range strings.SplitSeq(deck(chapter{Title: "T"}, b.String()), "\n---\n") {
		lignes := strings.Split(strings.TrimSpace(diapo), "\n")
		if dernier := lignes[len(lignes)-1]; strings.HasPrefix(dernier, "**Groupe") {
			t.Errorf("étiquette seule en bas de diapositive :\n%s", diapo)
		}
	}
}
