# Golang Demystified

Un livre en français qui présente Go dans son ensemble : le langage, ses outils,
sa bibliothèque standard et son évolution au fil des versions.

- 🌐 [Lire en ligne](https://laurentpoirierfr.github.io/golang-demystified/)
- 📄 [Télécharger le PDF](./book/golang-demystified.pdf)

## Organisation du dépôt

| Répertoire        | Rôle                                                         |
|-------------------|--------------------------------------------------------------|
| `site/`           | Site Hugo (thème [Hextra](https://imfing.github.io/hextra/)) |
| `site/content/docs` | Contenu du livre, source unique du site et du PDF          |
| `docs/`           | Site généré, publié par GitHub Pages                         |
| `book/`           | Métadonnées, ressources et PDF généré                        |
| `examples/`       | Code des exemples du livre, compilé et testé                 |
| `tools/mkbook`    | Assemble les pages en un seul Markdown pour pandoc           |

## Prérequis

- Go ≥ 1.27
- Hugo extended ≥ 0.150
- Docker (ou Podman) pour générer le PDF

## Commandes

```sh
make serve   # prévisualisation sur http://localhost:1313/golang-demystified/
make site    # génère le site dans docs/
make pdf     # génère book/golang-demystified.pdf
make check   # gofmt, go vet et go test sur les exemples et les outils
make all     # tout à la fois
```

## Règles d'écriture

Le même Markdown alimente le site et le PDF :

- l'ordre des parties et des chapitres est donné par le champ `weight` du front matter ;
- le titre vient du front matter : le contenu commence au niveau `##` ;
- pas de shortcodes Hugo dans `site/content/docs` (`make pdf` échoue sinon) ;
  pour les encadrés, utiliser les alertes GitHub (`> [!NOTE]`, `> [!TIP]`, `> [!WARNING]`) ;
- les images sont placées à côté de la page (page bundle) et référencées en relatif ;
- une page peut être exclue du PDF avec `pdf: false`.
