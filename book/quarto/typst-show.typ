// Remplace le partial « typst-show.typ » de Quarto pour personnaliser le
// modèle orange-book : couverture, logo, couleur principale et page de
// copyright. Les variables du modèle sont remplies par Quarto.
#import "@preview/orange-book:0.7.1": book, part, chapter, appendices

#let go-blue = rgb("#00ADD8")

#show: book.with(
$if(title)$
  title: [$title$],
$endif$
$if(subtitle)$
  subtitle: [$subtitle$],
$endif$
$if(by-author)$
  author: "$for(by-author)$$it.name.literal$$sep$, $endfor$",
$endif$
$if(date)$
  date: "$date$",
$endif$
  lang: "fr",
  main-color: go-blue,
  cover: image("assets/background.pdf"),
  cover-background: white.transparentize(15%),
  logo: image("assets/cover.png", width: 7cm),
  copyright: [
    #set text(size: 9pt)
    *Golang Demystified* — édition à jour pour Go 1.27.

    Le livre est aussi disponible en ligne :
    #link("https://laurentpoirierfr.github.io/golang-demystified/").

    Le code de tous les exemples est compilé et testé :
    #link("https://github.com/laurentpoirierfr/golang-demystified").
  ],
$if(toc-depth)$
  outline-depth: $toc-depth$,
$endif$
  supplement-chapter: "Chapitre",
  supplement-part: "Partie",
)
