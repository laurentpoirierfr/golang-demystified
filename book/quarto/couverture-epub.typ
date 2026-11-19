// Compile la couverture seule, pour l'image de couverture de l'EPUB :
//   typst compile couverture-epub.typ assets/couverture.png --font-path fonts
#import "couverture.typ": *

#set page(paper: "a4")
#couverture()
