# Golang Demystified

Un livre en français qui présente Go dans son ensemble : le langage, ses outils,
sa bibliothèque standard et son évolution au fil des versions.

- 🌐 [Lire en ligne](https://laurentpoirierfr.github.io/golang-demystified/)
- 📄 [Télécharger le PDF](./book/golang-demystified.pdf)
- 📱 [Télécharger l'EPUB](./book/golang-demystified.epub) (liseuses)

## Organisation du dépôt

| Répertoire          | Rôle                                                         |
|---------------------|--------------------------------------------------------------|
| `site/`             | Site Hugo (thème [Hextra](https://imfing.github.io/hextra/)) |
| `site/content/docs` | Contenu du livre, source unique du site, du PDF et de l'EPUB |
| `docs/`             | Site généré, publié par GitHub Pages                         |
| `book/quarto/`      | Configuration Quarto, modèle Typst et polices du livre       |
| `book/`             | Couverture, PDF et EPUB générés                              |
| `examples/`         | Code des exemples du livre, compilé et testé                 |
| `tools/mkbook`      | Transforme le contenu du site en projet de livre Quarto      |

## Prérequis

- Go ≥ 1.27
- Hugo extended ≥ 0.150
- Docker (ou Podman) pour générer le livre avec [Quarto](https://quarto.org)

## Commandes

```sh
make serve   # prévisualisation sur http://localhost:1313/golang-demystified/
make site    # génère le site dans docs/
make book    # génère book/golang-demystified.pdf et .epub (Quarto + Typst)
make check   # gofmt, go vet et go test sur les exemples et les outils
make all     # tout à la fois
```

## Règles d'écriture

Le même Markdown alimente le site, le PDF et l'EPUB :

- l'ordre des parties et des chapitres est donné par le champ `weight` du front matter ;
- le titre vient du front matter : le contenu commence au niveau `##` ;
- pas de shortcodes Hugo dans `site/content/docs` (`make book` échoue sinon) ;
  pour les encadrés, utiliser les alertes GitHub (`> [!NOTE]`, `> [!TIP]`,
  `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`), converties en callouts
  Quarto dans le livre ;
- les images sont placées à côté de la page (page bundle) et référencées en relatif ;
- une page peut être exclue du livre avec `pdf: false`.

## Gestion des versions

Le dépôt suit [git flow](https://nvie.com/posts/a-successful-git-branching-model/) :
le travail se fait sur des branches `feature/*` issues de `develop`, et chaque
release (`release/x.y.z`) régénère le site et le livre avant d'être fusionnée
dans `main`, qui est publiée par GitHub Pages.

## Licences des polices

Le livre est composé en Source Serif 4 et Source Sans 3 (Adobe, SIL Open Font
License) et en Go Mono (Bigelow & Holmes, licence BSD). Les licences sont dans
`book/quarto/fonts/`.
