---
title: "I.6 L'outil go"
weight: 40
---

# Chapitre I.6 — L'outil go

La commande `go` est le point d'entrée unique de tout l'outillage : elle
compile, teste, formate, analyse, documente et gère les dépendances. Ce
chapitre en fait le tour. `go help <commande>` affiche l'aide détaillée de
chacune.

## Les commandes essentielles

| Commande             | Rôle                                                              |
|----------------------|-------------------------------------------------------------------|
| `go run .`           | Compile et exécute le package courant                             |
| `go build ./...`     | Compile tous les packages du module                               |
| `go test ./...`      | Exécute les tests                                                 |
| `go fmt ./...`       | Formate le code (appelle `gofmt`)                                 |
| `go vet ./...`       | Détecte les erreurs probables que le compilateur laisse passer    |
| `go fix ./...`       | Modernise le code (Go 1.26)                                       |
| `go doc <symbole>`   | Affiche la documentation d'un package ou d'un symbole             |
| `go mod tidy`        | Synchronise `go.mod` et `go.sum` avec les imports du code         |
| `go get <module>`    | Ajoute ou met à jour une dépendance                               |
| `go install <outil>` | Compile et installe un exécutable dans `GOBIN`                    |
| `go tool <outil>`    | Lance un outil (`pprof`, `trace`, `cover`… ou un outil du module) |
| `go env`             | Affiche ou modifie la configuration                               |
| `go clean -cache`    | Vide le cache de compilation                                      |

## Les variables d'environnement

`go env` affiche la configuration complète. `go env -w VAR=valeur` enregistre
une valeur dans le fichier de configuration de l'utilisateur, sans toucher au
profil du shell.

| Variable      | Rôle                                                                   | Valeur par défaut                   |
|---------------|------------------------------------------------------------------------|-------------------------------------|
| `GOROOT`      | Emplacement de l'installation de Go                                    | Déduit de l'exécutable `go`         |
| `GOPATH`      | Répertoire de travail de Go                                            | `$HOME/go`                          |
| `GOMODCACHE`  | Cache des modules téléchargés                                          | `$GOPATH/pkg/mod`                   |
| `GOBIN`       | Destination de `go install`                                            | `$GOPATH/bin`                       |
| `GOCACHE`     | Cache de compilation et de tests                                       | `~/.cache/go-build` (Linux)         |
| `GOOS`, `GOARCH` | Système et architecture cibles                                      | Ceux de la machine                  |
| `GOPROXY`     | Proxy de téléchargement des modules                                    | `https://proxy.golang.org,direct`   |
| `GOPRIVATE`   | Modules privés, à récupérer sans proxy ni base de sommes de contrôle   | vide                                |
| `GOTOOLCHAIN` | Choix de la version de Go (voir « Installation »)                      | `auto`                              |
| `GOFLAGS`     | Options ajoutées à chaque commande                                     | vide                                |
| `CGO_ENABLED` | Autorise l'appel de code C                                             | `1` si un compilateur C est présent |

### GOROOT : l'installation de Go

`GOROOT` contient la distribution de Go :

- `bin/` : les commandes `go` et `gofmt` ;
- `src/` : le code source de la bibliothèque standard, écrit en Go. Le lire est
  un excellent moyen d'apprendre ;
- `pkg/tool/` : les outils internes (compilateur, éditeur de liens, `vet`,
  `pprof`…) ;
- `lib/`, `misc/`, `test/` : données annexes et tests de la distribution.

Il n'est jamais nécessaire de le définir à la main.

### GOPATH : l'héritage d'avant les modules

Avant Go 1.11, tout le code devait se trouver dans `$GOPATH/src`, et une seule
version de chaque dépendance pouvait exister sur la machine. Les modules ont
supprimé cette contrainte. `GOPATH` ne sert plus qu'à deux choses :

- `$GOPATH/pkg/mod` : le cache des modules téléchargés, partagé entre tous les
  projets et en lecture seule ;
- `$GOPATH/bin` : les exécutables installés par `go install`.

### Accéder à des modules privés

Les modules publics sont téléchargés via `proxy.golang.org`, et leur contenu est
vérifié auprès de la base de sommes de contrôle `sum.golang.org`. Pour un dépôt
privé, indiquez à Go de le contacter directement :

```bash
go env -w GOPRIVATE=gitlab.monentreprise.fr/*
```

## Gérer les dépendances

### go.mod

```go
module example.com/monservice

go 1.27.4

require (
	github.com/jackc/pgx/v5 v5.13.0
	golang.org/x/sync v0.25.0
)

require golang.org/x/text v0.44.0 // indirect

tool golang.org/x/tools/cmd/stringer

replace example.com/mabib => ../mabib
```

- **`module`** : le chemin d'importation du module.
- **`go`** : la version minimale du langage. Elle détermine aussi certains
  comportements, par exemple la portée des variables de boucle depuis Go 1.22.
- **`require`** : les dépendances et leur version minimale. Le commentaire
  `// indirect` signale une dépendance de dépendance. Depuis Go 1.27, `go mod
  tidy` regroupe toutes les dépendances en deux blocs, l'un direct et l'autre
  indirect.
- **`tool`** (Go 1.24) : les outils de développement utilisés par le projet.
- **`replace`** : remplace un module par une autre version ou par un
  répertoire local, pratique pour tester une modification.

À partir de v2, la version majeure fait partie du chemin d'importation
(`github.com/jackc/pgx/v5`). Deux versions majeures d'un même module peuvent
ainsi coexister dans un programme.

Go sélectionne pour chaque dépendance **la version minimale** qui satisfait
toutes les exigences (*Minimal Version Selection*). Les builds sont donc
reproductibles sans fichier de verrouillage : une nouvelle version d'une
dépendance n'est jamais utilisée sans que `go.mod` le dise.

### Les commandes du quotidien

```bash
go get github.com/jackc/pgx/v5@latest   # ajouter ou mettre à jour
go get github.com/jackc/pgx/v5@v5.12.0   # choisir une version précise
go get -u ./...                         # mettre à jour toutes les dépendances
go get github.com/old/dep@none          # retirer une dépendance
go mod tidy                             # nettoyer go.mod et go.sum
go list -m -u all                       # lister les mises à jour disponibles
go mod why golang.org/x/text            # pourquoi cette dépendance ?
go mod graph                            # le graphe des dépendances
```

### Les outils d'un projet

Depuis **Go 1.24**, les outils de développement (générateurs de code, linters…)
se déclarent dans `go.mod` avec la directive `tool`. Leur version est alors
figée comme celle de n'importe quelle dépendance :

```bash
go get -tool golang.org/x/tools/cmd/stringer
go tool stringer -type=Couleur
```

Avant Go 1.24, la convention était un fichier `tools.go` qui importait ces outils
avec une contrainte de build `//go:build tools`.

### Les workspaces

Depuis **Go 1.18**, un fichier `go.work` permet de travailler sur plusieurs
modules à la fois, sans ajouter de directives `replace` :

```bash
go work init ./monservice ./mabib
```

Les modifications de `mabib` sont alors immédiatement visibles dans
`monservice`. Le fichier `go.work` est en général local et n'est pas versionné.

## Analyser et moderniser le code

### go vet

`go vet` détecte des erreurs que le compilateur accepte : format de `Printf`
incohérent, copie d'un `sync.Mutex`, code inaccessible, balise de structure
mal formée… Une partie de ces vérifications est lancée automatiquement par
`go test`.

```bash
$ go vet ./...
./main.go:8:14: fmt.Printf format %d has arg name of wrong type string
```

### go fix et les « modernizers »

Réécrit en **Go 1.26**, `go fix` met le code à jour pour utiliser les
fonctionnalités récentes du langage et de la bibliothèque standard. Par
exemple, `interface{}` devient `any`, une boucle `for i := 0; i < n; i++`
devient `for i := range n`, `errors.As` devient `errors.AsType`.

```bash
go fix -diff ./...   # afficher les modifications proposées
go fix ./...         # les appliquer
go tool fix help     # lister les correcteurs disponibles
```

Une bibliothèque peut aussi marquer une fonction obsolète avec la directive
`//go:fix inline` : `go fix` remplace alors ses appels par son nouveau code
chez tous ses utilisateurs.

### govulncheck

[`govulncheck`](https://go.dev/doc/security/vuln/) compare les dépendances à la base de vulnérabilités de Go. Il ne
signale que les vulnérabilités dont le code est réellement appelé par le
programme :

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Lire la documentation

`go doc` affiche la documentation de n'importe quel package, type ou fonction,
y compris pour les dépendances du module :

```bash
go doc strings.Cut                 # une fonction
go doc -all net/http.Client        # un type et toutes ses méthodes
go doc -src slices.Sort            # le code source
go doc -ex bytes                   # les exemples exécutables (Go 1.27)
go doc -http                       # la documentation dans le navigateur
```

La documentation de tous les modules publics est aussi consultable sur
[pkg.go.dev](https://pkg.go.dev).
