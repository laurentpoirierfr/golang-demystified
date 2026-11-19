---
title: "Packages et modules"
weight: 120
---

## Les packages

Un **package** est un répertoire de fichiers `.go` qui déclarent tous le même
nom de package. C'est l'unité de compilation, d'encapsulation et de
réutilisation.

```
monservice/
├── go.mod                 module example.com/monservice
├── main.go                package main
└── stock/
    ├── stock.go           package stock
    └── stock_test.go      package stock
```

Un package s'importe par son **chemin**, formé du chemin du module et du
répertoire : `example.com/monservice/stock`. On l'utilise ensuite par son
**nom**, le dernier élément du chemin :

```go
import (
	"fmt"
	"net/http"

	"example.com/monservice/stock"
)

func main() {
	s := stock.Nouveau()
	fmt.Println(s.Quantite("vis"))
}
```

`goimports`, intégré à gopls, ajoute et range les imports automatiquement : la
bibliothèque standard d'abord, puis les autres modules.

Un import peut être renommé (`import str "strings"`) pour éviter un conflit de
noms. L'import `_` (`import _ "image/png"`) exécute seulement les fonctions
`init` du package, par exemple pour enregistrer un format ou un pilote de base
de données.

### Nommer un package

Un nom de package est court, en minuscules, sans souligné ni majuscule, et
décrit ce que le package **fournit** : `stock`, `http`, `json`. Évitez les noms
fourre-tout comme `utils`, `common` ou `helpers`.

Le nom du package fait partie de la façon dont on lit ses symboles : on écrit
`stock.Nouveau()` et non `stock.NouveauStock()`, `http.Client` et non
`http.HTTPClient`.

## La visibilité

Un identifiant est **exporté** (visible depuis les autres packages) si et
seulement s'il commence par une **majuscule**. Cette règle s'applique aux
fonctions, aux types, aux variables, aux constantes, aux champs et aux
méthodes.

```go
package stock

type Stock struct {
	articles map[string]int // privé au package
}

// Nouveau crée un stock vide.
func Nouveau() *Stock {
	return &Stock{articles: make(map[string]int)}
}

// Quantite renvoie la quantité disponible d'un article.
func (s *Stock) Quantite(article string) int { return s.articles[article] }
```

Les fichiers d'un même package se partagent tout, y compris les identifiants
privés.

### Le répertoire internal

Un package situé sous un répertoire `internal/` ne peut être importé que par
les packages de l'arborescence parente de ce répertoire. C'est le moyen de
partager du code entre les packages d'un module sans l'exposer à ses
utilisateurs.

```
monservice/
├── internal/
│   └── auth/          importable par monservice/… uniquement
└── api/
```

## Documenter un package

Les commentaires placés juste avant une déclaration sont sa documentation. Ils
commencent par le nom de l'élément documenté, et forment des phrases
complètes :

```go
// Package stock gère l'inventaire des articles d'un entrepôt.
package stock

// Quantite renvoie la quantité disponible d'un article,
// ou 0 si l'article est inconnu.
func (s *Stock) Quantite(article string) int
```

`go doc` et [pkg.go.dev](https://pkg.go.dev) affichent ces commentaires. Depuis
**Go 1.19**, ils acceptent des titres (`# Titre`), des listes et des liens
(`[strings.Cut]`).

Les **exemples** sont des fonctions `ExampleXxx` placées dans les fichiers de
test. Ils apparaissent dans la documentation, et `go test` vérifie que leur
sortie correspond au commentaire `// Output:`. Tous les exemples de ce livre
sont écrits ainsi.

## Les modules

Un **module** est un ensemble de packages versionnés ensemble, décrit par un
fichier `go.mod` à sa racine. Le chapitre « L'outil go » détaille la gestion
des dépendances. Rappelons les principes :

- un module a un **chemin** (`github.com/utilisateur/projet`), qui est aussi le
  préfixe des chemins d'importation de ses packages ;
- les versions suivent le **versionnage sémantique** et sont des tags Git :
  `v1.4.2` ;
- une version majeure à partir de v2 change le chemin du module :
  `github.com/utilisateur/projet/v2` ;
- les versions `v0.x.y` n'offrent aucune garantie de compatibilité.

### Publier un module

Publier un module, c'est simplement pousser un tag sur un dépôt public :

```bash
git tag v1.0.0
git push origin v1.0.0
```

Le proxy `proxy.golang.org` le récupère dès qu'un utilisateur le demande, et
la documentation apparaît sur pkg.go.dev.

## Organiser un projet

Il n'existe pas de structure imposée. Quelques conventions sont largement
partagées :

- un petit projet tient dans un seul package, à la racine du module ;
- les exécutables se placent dans `cmd/<nom>/main.go` lorsqu'il y en a
  plusieurs ;
- le code non destiné à être importé par d'autres modules va dans `internal/` ;
- on découpe en packages selon les **responsabilités** métier (`commande`,
  `facturation`), plutôt que selon les couches techniques (`models`,
  `controllers`).

Les cycles d'imports sont interdits : si deux packages ont besoin l'un de
l'autre, c'est le signe qu'une responsabilité est mal découpée. Une interface
définie côté utilisateur permet souvent de casser le cycle.

La partie Approfondissements présente des exemples complets d'organisation :
un service web complet et une architecture hexagonale.
