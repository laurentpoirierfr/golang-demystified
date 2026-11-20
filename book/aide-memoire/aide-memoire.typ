// Mise en page de l'aide-mémoire : A4 à l'italienne, en deux colonnes, dense.
#set page(flipped: true, numbering: "1 / 1")
#set par(justify: false, leading: 0.5em, spacing: 0.7em)
#show heading: set text(font: "Source Sans 3", fill: rgb("#007D9C"))
#show heading.where(level: 2): set block(above: 1em, below: 0.5em)
#show raw: set text(font: ("Go Mono", "DejaVu Sans Mono"), size: 7pt)
#show raw.where(block: true): set block(fill: luma(245), inset: 4pt, radius: 2pt, width: 100%)
#show table: set text(size: 7pt)
#show link: set text(fill: rgb("#007D9C"))
