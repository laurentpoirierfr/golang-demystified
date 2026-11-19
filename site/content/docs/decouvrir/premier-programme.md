---
title: "Premier programme"
weight: 30
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/hello`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/hello).

## Créer un module

Tout projet Go est un **module** : un ensemble de packages versionnés ensemble,
décrit par un fichier `go.mod`. Créez un répertoire, puis initialisez le module
avec son **chemin d'importation**, en général l'adresse du dépôt où il sera
publié :

```bash
mkdir bonjour && cd bonjour
go mod init example.com/bonjour
```

Le fichier `go.mod` produit est minimal :

```go
module example.com/bonjour

go 1.27.4
```

## Le code

Créez le fichier `main.go` :

```go
// Le premier programme du livre.
package main

import "fmt"

func main() {
	fmt.Println(greeting("Gopher"))
}

func greeting(name string) string {
	return "Bonjour, " + name + " !"
}
```

Quelques remarques sur ces lignes :

- **`package main`** : chaque fichier appartient à un package. Le package
  `main` est spécial : il produit un exécutable, dont le point d'entrée est la
  fonction `main`.
- **`import "fmt"`** : importe le package de formatage de la bibliothèque
  standard. Un import inutilisé est une erreur de compilation, tout comme une
  variable locale inutilisée.
- **`func greeting(name string) string`** : le type se place *après* le nom,
  aussi bien pour les paramètres que pour la valeur de retour.
- **`fmt.Println`** : seuls les noms qui commencent par une majuscule sont
  exportés, c'est-à-dire visibles hors de leur package. `greeting` est donc
  privé au package `main`.
- **Pas de point-virgule** : le compilateur les insère automatiquement en fin
  de ligne. C'est pour cela que l'accolade ouvrante doit rester sur la même
  ligne que `func`, `if` ou `for`.
- **L'indentation se fait avec des tabulations**, comme l'impose `gofmt`.

## Exécuter et compiler

`go run` compile le programme dans un répertoire temporaire et l'exécute :

```bash
$ go run .
Bonjour, Gopher !
```

`go build` produit un exécutable autonome, sans dépendance à une machine
virtuelle ni à une bibliothèque partagée :

```bash
$ go build -o bonjour .
$ ./bonjour
Bonjour, Gopher !
```

La compilation croisée ne demande que deux variables d'environnement. Pour
produire un exécutable Windows ou un binaire pour un Raspberry Pi depuis Linux :

```bash
GOOS=windows GOARCH=amd64 go build -o bonjour.exe .
GOOS=linux GOARCH=arm64 go build -o bonjour-arm64 .
```

`go tool dist list` affiche toutes les plateformes prises en charge.

## Un premier test

Les tests sont intégrés à l'outil. Un fichier qui se termine par `_test.go`
contient des fonctions `TestXxx` qui reçoivent un `*testing.T`. Créez
`main_test.go` :

```go
package main

import "testing"

func TestGreeting(t *testing.T) {
	if got, want := greeting("Gopher"), "Bonjour, Gopher !"; got != want {
		t.Errorf("greeting() = %q, attendu %q", got, want)
	}
}
```

```bash
$ go test ./...
ok  	example.com/bonjour	0.002s
```

Le motif `./...` désigne tous les packages du module. Les tests sont détaillés
dans le chapitre consacré au package `testing`.

## Ajouter une dépendance

Les dépendances sont des modules, référencés par leur chemin d'importation.
Il suffit de les importer dans le code, puis de laisser `go mod tidy` mettre à
jour `go.mod` :

```go
import "golang.org/x/text/language"
```

```bash
$ go mod tidy
go: finding module for package golang.org/x/text/language
go: found golang.org/x/text/language in golang.org/x/text v0.44.0
```

`go.mod` contient alors une directive `require`, et un fichier `go.sum`
apparaît. Il contient les empreintes cryptographiques de chaque dépendance :
une dépendance modifiée après sa publication est détectée. Ces deux fichiers
doivent être versionnés avec le code.

> [!TIP]
> Le [Go Playground](https://go.dev/play/) permet d'exécuter du code Go dans le
> navigateur, sans rien installer. C'est pratique pour essayer un exemple de ce
> livre.
