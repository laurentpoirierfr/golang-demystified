// Remplace le partial « typst-show.typ » de Quarto pour personnaliser le
// modèle orange-book : couverture, couleurs, polices, blocs de code.
// Les variables du modèle sont remplies par Quarto.
#import "@preview/orange-book:0.7.1": book, part, chapter, appendices

// Palette tirée de l'identité visuelle de Go.
#let go-blue = rgb("#00ADD8")
#let go-blue-dark = rgb("#007D9C")
#let go-ink = rgb("#202224")
#let code-bg = rgb("#F3F8FA")

#let font-text = "Source Serif 4"
#let font-titles = "Source Sans 3"
#let font-code = "Go Mono"

// Typographie : texte à empattements, titres et tableaux sans empattements.
// Ces règles précèdent le modèle pour s'appliquer aussi à la couverture et
// à la table des matières.
#set text(font: font-text, fill: go-ink)
#set par(justify: true, leading: 0.62em)
#show heading: set text(font: font-titles)
#show outline: set text(font: font-titles)
#show table: set text(font: font-titles, size: 9.5pt)
// DejaVu Sans Mono complète Go Mono pour les caractères de dessin (├──).
#show raw: set text(font: (font-code, "DejaVu Sans Mono"), size: 8.8pt)
#show link: set text(fill: go-blue-dark)

#show: book.with(
$if(title)$
  title: text(font: font-titles, weight: "bold", fill: go-ink)[$title$],
$endif$
$if(subtitle)$
  subtitle: text(font: font-titles, fill: go-blue-dark)[$subtitle$],
$endif$
$if(by-author)$
  author: "$for(by-author)$$it.name.literal$$sep$, $endfor$",
$endif$
$if(date)$
  date: "$date$",
$endif$
  lang: "fr",
  main-color: go-blue,
  font-size: 10.5pt,
  cover: image("assets/background.pdf"),
  cover-background: white.transparentize(10%),
  logo: image("assets/cover.png", width: 5.5cm),
  copyright: [
    #set text(size: 9pt)
    *Golang Demystified* — édition à jour pour Go 1.27.

    Le livre est aussi disponible en ligne :
    #link("https://laurentpoirierfr.github.io/golang-demystified/").

    Le code de tous les exemples est compilé et testé à chaque modification :
    #link("https://github.com/laurentpoirierfr/golang-demystified").

    Texte composé en Source Serif 4 et Source Sans 3 (Adobe, licence SIL OFL),
    code en Go Mono (Bigelow & Holmes, licence BSD).
  ],
$if(toc-depth)$
  outline-depth: $toc-depth$,
$endif$
  supplement-chapter: "Chapitre",
  supplement-part: "Partie",
)

// Puces disponibles dans Source Serif 4 (le modèle utilise « ◦ », absent).
#set list(marker: ([•], [–], [·]))

// Tableaux : en-tête souligné en bleu Go, lignes fines.
#set table(
  stroke: (x, y) => (
    top: if y == 1 { 0.8pt + go-blue } else { 0pt },
    bottom: 0.4pt + luma(220),
  ),
  inset: (x: 6pt, y: 5pt),
)

// Blocs de code : fond clair et filet bleu Go à gauche. Cette fonction
// remplace celle que Quarto génère pour la coloration syntaxique.
#let Skylighting(fill: none, number: false, start: 1, sourcelines) = {
  let blocks = []
  let lnum = start - 1
  for ln in sourcelines {
    if number {
      lnum = lnum + 1
      blocks = blocks + box(width: 24pt, text(fill: luma(160), [#lnum]))
    }
    blocks = blocks + ln + EndLine()
  }
  block(
    fill: code-bg,
    stroke: (left: 2.5pt + go-blue),
    width: 100%,
    inset: (left: 10pt, right: 8pt, y: 8pt),
    radius: (right: 3pt),
    breakable: true,
    blocks,
  )
}
