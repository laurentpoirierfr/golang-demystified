---
title: "CI/CD d'un projet Go"
linkTitle: "14. CI/CD"
weight: 140
---

L'outillage de Go se prête particulièrement bien à l'automatisation : une
seule commande compile, teste, analyse et formate, sans configuration. Ce
chapitre montre comment vérifier chaque modification et livrer chaque version
automatiquement, avec [GitHub Actions](https://docs.github.com/actions).

L'exemple n'est pas fictif : c'est l'intégration continue **de ce livre**. Ses
fichiers sont dans le dépôt, dans `.github/workflows/`, et s'exécutent à chaque
modification.

## Ce que l'on automatise

| Moment | Vérification ou action | Outil |
|--------|------------------------|-------|
| Chaque push | format du code | `gofmt` |
| | analyse statique | `go vet`, [golangci-lint](https://golangci-lint.run) |
| | tests, avec le détecteur de courses | `go test -race` |
| | vulnérabilités connues | [`govulncheck`](https://go.dev/doc/security/vuln/) |
| | tests d'intégration avec de vrais services | [Testcontainers](https://golang.testcontainers.org) |
| | code généré à jour | `go generate`, `git diff` |
| | livraison possible | [GoReleaser](https://goreleaser.com), à blanc |
| Chaque semaine | dépendances à jour | [Dependabot](https://docs.github.com/code-security/dependabot) |
| Chaque version | exécutables, archives, release | GoReleaser |
| | site et documentation | Hugo, GitHub Pages |

## GitHub Actions en bref

Un **workflow** est un fichier YAML placé dans `.github/workflows/`. Il est
déclenché par des événements (`push`, `pull_request`, une planification…) et
contient des **jobs**, exécutés en parallèle sur des machines virtuelles. Chaque
job enchaîne des **étapes** : une commande (`run`) ou une action réutilisable
(`uses`).

L'action `actions/setup-go` installe Go. L'option `go-version-file` lit la
version dans `go.mod` : la CI utilise toujours la version déclarée par le
projet, sans la dupliquer. Elle met aussi en cache les modules téléchargés et
le cache de compilation.

```yaml
name: CI

on:
  push:
    branches: [main, develop, "feature/**", "release/**", "hotfix/**"]
  pull_request:

permissions:
  contents: read # droits minimaux : lecture seule

# Un nouveau push sur la même branche annule la vérification en cours.
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
```

Les **permissions** limitent ce que le jeton d'accès du workflow peut faire.
Accordez le strict nécessaire : en lecture seule pour vérifier, en écriture
seulement pour publier.

## Vérifier chaque modification

### Format, analyse, tests et vulnérabilités

Le premier job regroupe les vérifications rapides :

```yaml
  verifications:
    name: Format, analyse, tests et vulnérabilités
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod

      - name: Formatage (gofmt)
        run: |
          fichiers=$(gofmt -l tools examples)
          if [ -n "$fichiers" ]; then
            echo "Fichiers à formater avec gofmt :"
            echo "$fichiers"
            exit 1
          fi

      - name: Analyse statique (go vet)
        run: go vet ./...

      - name: Tests, avec le détecteur de courses
        run: go test -race -short ./...

      - name: Vulnérabilités (govulncheck)
        run: go tool govulncheck ./...
```

- `gofmt -l` liste les fichiers mal formatés sans les modifier : la CI échoue
  au lieu de corriger, et le développeur lance `gofmt` lui-même ;
- `-race` active le détecteur de courses. Il demande un compilateur C,
  présent sur les machines de GitHub : c'est souvent là qu'on l'exécute
  plutôt qu'en local ;
- `-short` saute les tests longs, comme les tests d'intégration, confiés à un
  job séparé ;
- `govulncheck` est déclaré comme **outil du module** (directive `tool` de Go
  1.24) : la CI et les développeurs utilisent la même version, figée dans
  `go.mod`.

`govulncheck` ne signale que les vulnérabilités dont le code est
**réellement appelé**. Sur ce dépôt, il trouve deux vulnérabilités dans des
dépendances, mais dans des fonctions que le code n'utilise pas : il ne fait
donc pas échouer la CI.

```
Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 2
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
```

### golangci-lint

`go vet` détecte les erreurs certaines. [golangci-lint](https://golangci-lint.run)
exécute en une passe des dizaines d'analyseurs supplémentaires : erreurs non
vérifiées (`errcheck`), affectations inutiles (`ineffassign`), simplifications
et bugs subtils (`staticcheck`), code mort (`unused`)…

```yaml
# .golangci.yml
version: "2"

linters:
  # errcheck, govet, ineffassign, staticcheck, unused.
  default: standard
  enable:
    - bodyclose  # corps de réponse HTTP non fermé
    - errorlint  # comparaison d'erreurs sans errors.Is / errors.As
  exclusions:
    presets:
      # Erreurs habituellement ignorées : Close, Flush, fmt.Fprint…
      - std-error-handling
      - common-false-positives
    rules:
      # Dans les tests, une erreur inattendue fait échouer le test par ailleurs.
      - path: _test\.go
        linters: [errcheck]
      # Les exemples de la partie « Le langage » contiennent volontairement du
      # code à ne pas imiter, pour l'expliquer.
      - path: examples/langage/
        linters: [errorlint, ineffassign, staticcheck]

formatters:
  enable:
    - gofmt
    - goimports
```

```yaml
  lint:
    name: Linters (golangci-lint)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v9
        with:
          version: v2.15.0
```

Sa première exécution sur le code de ce livre a signalé 56 points. La plupart
étaient volontaires ou bénins, mais elle a trouvé de **vrais oublis**, corrigés
depuis :

- dans l'opérateur Kubernetes, les erreurs de `AddHealthzCheck` et
  `AddReadyzCheck` étaient ignorées ;
- dans un middleware, la valeur d'une panique était comparée à
  `http.ErrAbortHandler` avec `==` au lieu de `errors.Is` ;
- une erreur d'annulation de transaction était perdue ; elle est désormais
  combinée à l'erreur d'origine :

  ```go
  if err := f(tx); err != nil {
  	return errors.Join(err, tx.Rollback()) // l'erreur d'annulation n'est pas perdue
  }
  ```

Lorsqu'une erreur est ignorée volontairement, rendez-le **explicite** avec
`_ =` et une explication, plutôt que de désactiver l'analyseur :

```go
// Les en-têtes sont déjà envoyés : une erreur d'encodage ne peut plus être
// signalée au client.
_ = json.NewEncoder(w).Encode(v)
```

Sur un projet existant, l'option `--new-from-rev=main` ne signale que les
problèmes introduits par les modifications en cours : on améliore le code
progressivement, sans tout corriger d'un coup.

### Les tests d'intégration

Les machines Ubuntu de GitHub disposent de Docker : Testcontainers y démarre
PostgreSQL sans aucune configuration.

```yaml
  integration:
    name: Tests d'intégration (Testcontainers)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - name: Tests avec PostgreSQL
        run: go test -count=1 -run Postgres -v ./examples/...
```

```
🔔 Container is ready: c241370f3985
--- PASS: TestPostgres (14.39s)
--- PASS: TestPostgresCatalogue (14.64s)
--- PASS: TestPostgresMigrationsReversibles (2.70s)
```

### Le code généré à jour

Le code généré (sqlc, oapi-codegen, controller-gen) est versionné, pour que le
projet compile sans les générateurs. Il faut donc vérifier qu'il correspond
bien à ses sources : un développeur peut modifier le contrat OpenAPI et oublier
de régénérer le code. La CI régénère tout, et échoue si le résultat diffère de
ce qui est versionné :

```yaml
  code-genere:
    name: Code généré à jour
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - name: Régénération
        run: make generate
      - name: Aucune différence avec le code versionné
        run: |
          if ! git diff --exit-code; then
            echo "::error::Le code généré n'est pas à jour : lancez « make generate » et versionnez le résultat."
            exit 1
          fi
```

La syntaxe `::error::` est une **commande de workflow** : GitHub affiche le
message en évidence dans l'interface.

## Garder les dépendances à jour : Dependabot

Une dépendance non mise à jour finit par accumuler des vulnérabilités, et des
mises à jour d'autant plus difficiles qu'elles sont tardives. Dependabot
propose chaque semaine des pull requests de mise à jour, que la CI vérifie
automatiquement :

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
    groups:
      # Une seule pull request pour les modules Kubernetes, qui évoluent ensemble.
      kubernetes:
        patterns: ["k8s.io/*", "sigs.k8s.io/*"]
      golang-x:
        patterns: ["golang.org/x/*"]

  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
```

Les **groupes** évitent une avalanche de pull requests pour des modules qui
doivent être mis à jour ensemble. [Renovate](https://docs.renovatebot.com)
est une alternative plus configurable, qui fonctionne aussi hors de GitHub.

## Livrer un exécutable : GoReleaser

Publier un outil en ligne de commande, c'est le compiler pour chaque
plateforme, l'archiver, calculer des sommes de contrôle, et publier le tout.
[GoReleaser](https://goreleaser.com) automatise ces étapes à partir d'un
fichier de configuration. Voici celui de `taches-cli`, le client du chapitre
« Ligne de commande » :

```yaml
# .goreleaser.yaml
version: 2
project_name: taches-cli

builds:
  - id: taches-cli
    main: ./cmd/taches-cli
    binary: taches-cli
    env:
      - CGO_ENABLED=0 # exécutables statiques
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    flags:
      - -trimpath # chemins de compilation retirés : builds reproductibles
    ldflags:
      - -s -w
      - -X github.com/…/internal/cli.Version={{ .Version }}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]

checksum:
  name_template: checksums.txt
```

```
$ goreleaser release --snapshot --clean --skip=publish
  • building     target=linux_amd64_v1
  • building     target=darwin_arm64_v8.0
  …
  • archiving    name=dist/taches-cli_1.8.0-SNAPSHOT-bd69b3f_linux_amd64.tar.gz
  • archiving    name=dist/taches-cli_1.8.0-SNAPSHOT-bd69b3f_windows_amd64.zip
  …
  • calculating checksums
  • release succeeded after 33s

$ ./dist/taches-cli_linux_amd64_v1/taches-cli --version
taches-cli version 1.8.0-SNAPSHOT-bd69b3f
```

La version est **injectée à la compilation** grâce à `-ldflags -X`, qui
modifie la valeur d'une variable de type `string` : aucun fichier à mettre à
jour avant chaque release.

Dans la CI, une exécution « à blanc » (`--snapshot --skip=publish`) vérifie à
chaque push que la livraison fonctionnerait :

```yaml
  goreleaser:
    name: Livraison de taches-cli (GoReleaser, à blanc)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0 # GoReleaser lit les tags pour nommer la version
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: v2.20.1
          args: release --snapshot --clean --skip=publish
          workdir: examples/approfondissements/api
```

Pour une vraie publication, un workflow déclenché par un tag lance
`goreleaser release` : les archives sont attachées à la release GitHub.
GoReleaser sait aussi produire des paquets (deb, rpm, apk), des formules
[Homebrew](https://brew.sh), des images Docker, et signer les artefacts.

## Les images de conteneurs

Un exécutable Go statique tient dans une image minimale. La construction en
deux étapes vue dans le chapitre « Une API web complète » produit une image de
quelques mégaoctets. Pour publier une image pour plusieurs architectures
(amd64 pour les serveurs, arm64 pour Graviton ou les Mac récents), deux
approches existent :

- **Docker Buildx**, qui construit pour plusieurs plateformes à partir du même
  `Dockerfile` :

  ```bash
  docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/exemple/taches:1.0 --push .
  ```

- **[ko](https://ko.build)**, conçu pour Go : il compile directement l'image,
  sans `Dockerfile` ni démon Docker, grâce à la compilation croisée de Go :

  ```bash
  KO_DOCKER_REPO=ghcr.io/exemple ko build --platform=linux/amd64,linux/arm64 ./cmd/taches
  ```

## Savoir ce que contient un exécutable : SBOM

Une **SBOM** (*Software Bill of Materials*) est l'inventaire des composants
d'un logiciel. Elle permet de savoir, le jour où une vulnérabilité est publiée,
quels exécutables sont concernés. Go facilite l'exercice : chaque exécutable
**embarque** la liste exacte de ses modules et de ses options de compilation.

```
$ go version -m taches-cli
taches-cli: go1.27.3
	path	github.com/…/examples/approfondissements/api/cmd/taches-cli
	dep	github.com/fsnotify/fsnotify	v1.9.0	h1:2Ml+OJNzbYCTzsxtv8vKSFD9PbJjmhYF14k/jKC7S9k=
	dep	github.com/oapi-codegen/runtime	v1.8.0	h1:RJzEgQSJQADEPaG1cYuSCKYai/cCVRE2xnkrqhJs/3c=
	…
	build	-trimpath=true
	build	CGO_ENABLED=0
	build	GOARCH=amd64
	build	GOOS=linux
```

`govulncheck -mode=binary taches-cli` analyse directement un exécutable, sans
son code source. Pour produire une SBOM aux formats standard (SPDX,
CycloneDX), [Syft](https://github.com/anchore/syft) lit ces informations, et
GoReleaser peut l'appeler à chaque release.

## Publier un site : l'exemple de ce livre

Le site et le livre que vous lisez sont publiés par un second workflow,
déclenché à chaque push sur `main`, c'est-à-dire à la fin de chaque release :

```yaml
name: Publication

on:
  push:
    branches: [main]

jobs:
  construction:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      tag: ${{ steps.version.outputs.tag }}
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0 # pour lire les tags
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - name: Installation de Hugo
        run: …
      - name: Version publiée
        id: version
        run: echo "tag=$(git describe --exact-match --tags HEAD 2>/dev/null || true)" >> "$GITHUB_OUTPUT"
      - name: Livre (PDF et EPUB)
        run: make book
      - name: Site, avec le livre téléchargeable
        run: make publication
      - uses: actions/upload-pages-artifact@v5
        with:
          path: site/public
      …

  deploiement:
    needs: construction
    permissions:
      pages: write
      id-token: write
    environment:
      name: github-pages
    steps:
      - uses: actions/deploy-pages@v5

  release:
    needs: construction
    if: needs.construction.outputs.tag != ''
    permissions:
      contents: write
    steps:
      - uses: actions/download-artifact@v8
        with:
          name: livre
      - run: gh release create "$TAG" golang-demystified.pdf golang-demystified.epub …
```

Trois jobs, trois responsabilités, trois niveaux de permissions : construire
(lecture), déployer le site (Pages), créer la release (écriture). Les fichiers
générés ne sont plus versionnés dans le dépôt : ils sont reconstruits à chaque
publication.

## En résumé

- Automatisez **tout** ce qui peut l'être : format, analyse, tests,
  vulnérabilités, code généré, livraison. Un oubli humain devient une CI rouge.
- Figez les versions des outils (`tool` dans `go.mod`, versions explicites des
  actions et des images) : la CI et les postes de développement donnent les
  mêmes résultats.
- Donnez aux workflows les **permissions minimales**.
- Faites de la CI la seule voie de publication : une release ne dépend plus du
  poste de quelqu'un.
