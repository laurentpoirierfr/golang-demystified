// Couverture du livre et identité visuelle (couleurs, polices).
// Ce module est importé par typst-show.typ pour le PDF, et compilé seul
// par couverture-epub.typ pour produire l'image de couverture de l'EPUB.

// Palette tirée de l'identité visuelle de Go.
#let go-blue = rgb("#00ADD8")
#let go-blue-dark = rgb("#007D9C")
#let go-ink = rgb("#202224")
#let code-bg = rgb("#F3F8FA")

#let font-text = "Source Serif 4"
#let font-titles = "Source Sans 3"
#let font-code = "Go Mono"

// Frise des versions marquantes, de Go 1.0 à la version décrite par le livre.
#let frise(largeur: 10cm) = {
  let versions = ("1.0", "1.5", "1.11", "1.18", "1.21", "1.23", "1.27")
  let n = versions.len() - 1
  block(width: largeur, height: 1.6cm, {
    place(top + left, dy: 0.4cm, line(length: largeur, stroke: 1.5pt + go-blue))
    for (i, v) in versions.enumerate() {
      let x = largeur * i / n
      let actuelle = i == n
      place(top + left, dx: x - 5pt, dy: 0.4cm - 5pt, circle(
        radius: 5pt,
        fill: if actuelle { go-blue } else { white },
        stroke: 1.5pt + go-blue,
      ))
      place(top + left, dx: x - 0.7cm, dy: 0.75cm, box(width: 1.4cm, align(center, text(
        size: 9pt,
        weight: if actuelle { "bold" } else { "regular" },
        fill: if actuelle { go-blue-dark } else { luma(100) },
        "Go " + v,
      ))))
    }
  })
}

// Page de couverture, pleine page, avant le modèle orange-book.
#let couverture() = page(margin: 0pt, header: none, footer: none, numbering: none, {
  set text(font: font-titles)
  set par(justify: false)

  // Les triangles en bas de page.
  place(top + left, image("assets/background.png", width: 100%, height: 100%))

  // Le bandeau bleu en biais, dans l'angle des triangles.
  place(top + left, polygon(
    fill: gradient.linear(go-blue, go-blue-dark, angle: 30deg),
    (0cm, 0cm), (21cm, 0cm), (21cm, 10.5cm), (0cm, 14.5cm),
  ))

  // Le badge de l'édition.
  place(top + right, dx: -1.8cm, dy: 1.8cm, box(
    fill: white, radius: 14pt, inset: (x: 11pt, y: 6pt),
    text(size: 10pt, weight: "bold", fill: go-blue-dark)[Édition Go 1.27],
  ))

  // Le titre.
  place(top + left, dx: 2cm, dy: 3.4cm, block(width: 12cm, {
    text(size: 10pt, weight: "semibold", tracking: 2.5pt, fill: white.transparentize(15%))[
      LE LANGAGE GO EXPLIQUÉ
    ]
    v(0.5cm)
    text(size: 50pt, weight: "bold", fill: white, top-edge: "cap-height")[Golang]
    linebreak()
    text(size: 50pt, weight: "bold", fill: white)[Demystified]
    v(0.6cm)
    text(size: 15pt, fill: white)[Le langage, ses outils et son évolution]
  }))

  // Le gopher, assis sur la diagonale.
  place(top + right, dx: -1.6cm, dy: 7.6cm, image("assets/cover.png", width: 6.8cm))

  // La frise des versions, dans la zone blanche.
  place(top + left, dx: 2cm, dy: 16.6cm, {
    frise(largeur: 10cm)
    v(0.25cm)
    text(size: 10pt, fill: luma(90))[Quatorze ans d'évolution, version par version]
  })

  // L'auteur, sur les triangles.
  place(bottom + right, dx: -2cm, dy: -2cm,
    text(size: 15pt, weight: "semibold", fill: white)[The Gang of Three])
})

