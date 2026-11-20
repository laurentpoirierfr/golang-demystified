---
title: "Pointeurs et mémoire"
linkTitle: "10. Pointeurs et mémoire"
weight: 100
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/pointeurs`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/pointeurs).

## Les pointeurs

Un pointeur contient l'**adresse** d'une variable. Le type `*T` désigne un
pointeur vers une valeur de type `T`. L'opérateur `&` prend l'adresse d'une
variable, et `*` accède à la valeur pointée :

```go
x := 10
p := &x // p est de type *int
*p = 20
fmt.Println(x, *p, p == &x)
// 20 20 true
```

La valeur zéro d'un pointeur est `nil`. Déréférencer un pointeur nil provoque
un `panic`.

Contrairement au C, il n'y a **pas d'arithmétique de pointeurs** (`p++` ne
compile pas), et il est impossible d'obtenir un pointeur invalide hors du
package `unsafe`. Les pointeurs de Go sont sûrs.

## Pourquoi utiliser un pointeur ?

Go passe tous les arguments **par valeur**. Une fonction qui doit modifier une
variable de l'appelant reçoit son adresse :

```go
func doubleValeur(n int)    { n *= 2 }
func doublePointeur(n *int) { *n *= 2 }

a := 5
doubleValeur(a)
fmt.Println(a) // 5 : la fonction a modifié une copie
doublePointeur(&a)
fmt.Println(a) // 10
```

On utilise un pointeur pour :

- **modifier** une valeur depuis une fonction ou une méthode ;
- **éviter la copie** d'une grande structure ;
- **partager** une valeur entre plusieurs parties du programme ;
- exprimer une valeur **optionnelle** : `nil` signifie « absente » ;
- construire des **structures chaînées** (listes, arbres, graphes).

```go
type Noeud struct {
	Valeur  int
	Suivant *Noeud
}

var tete *Noeud
for i := range 3 {
	tete = &Noeud{Valeur: i, Suivant: tete}
}
for n := tete; n != nil; n = n.Suivant {
	fmt.Print(n.Valeur)
}
// 210
```

Pour les petites structures et les types de base, une copie est souvent plus
rapide qu'un pointeur : elle évite une allocation et une indirection.
N'utilisez pas les pointeurs « par réflexe ».

## new

`new(T)` alloue une variable de type `T`, initialisée à sa valeur zéro, et
renvoie son adresse. En pratique, on écrit plus souvent `&T{…}` pour une
structure.

```go
p := new(int) // pointeur vers un int initialisé à 0
*p = 3
```

> [!NOTE]
> **Go 1.26** — `new` accepte une **expression**, qui initialise la variable.
>
> ```go
> q := new(42) // *int qui pointe vers 42
> ```

Cette évolution simplifie les champs optionnels, très fréquents dans les
structures qui représentent du JSON ou des messages protobuf. Il n'est plus
nécessaire de passer par une variable temporaire ou une fonction utilitaire
`ptr(v)` :

```go
type Utilisateur struct {
	Nom        string
	Age        *int       // optionnel
	Expiration *time.Time // optionnel
}

u := Utilisateur{
	Nom: "Alice",
	Age: new(anneesDepuis(naissance)),
}
```

## La pile, le tas et l'analyse d'échappement

En Go, le développeur ne choisit pas où une variable est allouée. Le
compilateur décide, par l'**analyse d'échappement** :

- une variable qui ne survit pas à sa fonction est placée sur la **pile**.
  C'est très rapide, et elle est libérée automatiquement au retour de la
  fonction ;
- une variable dont l'adresse « s'échappe » (renvoyée, stockée dans une
  structure qui survit, capturée par une closure ou une goroutine…) est placée
  sur le **tas**, géré par le ramasse-miettes.

Il est donc parfaitement correct de renvoyer l'adresse d'une variable locale :

```go
func nouveauCompteur() *int {
	n := 0
	return &n // n s'échappe : il est alloué sur le tas
}
```

L'option `-gcflags=-m` affiche les décisions du compilateur :

```bash
$ go build -gcflags=-m .
./main.go:4:2: moved to heap: n
```

Réduire les allocations sur le tas est l'une des principales optimisations en
Go. Le chapitre sur le profilage montre comment les mesurer.

## Le ramasse-miettes

La mémoire du tas est libérée par un **ramasse-miettes** (*garbage collector*)
concurrent : il fonctionne en parallèle du programme, avec des pauses de
l'ordre de quelques dizaines de microsecondes.

Deux réglages permettent de l'ajuster :

| Variable     | Rôle                                                                   |
|--------------|------------------------------------------------------------------------|
| `GOGC`       | Croissance du tas qui déclenche un cycle (100 % par défaut)            |
| `GOMEMLIMIT` | Limite de mémoire souple : le GC travaille plus dur à son approche (Go 1.19) |

Dans un conteneur dont la mémoire est limitée, fixer `GOMEMLIMIT` un peu
en dessous de la limite évite les arrêts pour manque de mémoire.

> [!NOTE]
> **Go 1.26** — le ramasse-miettes **Green Tea** est activé par défaut. Il
> parcourt la mémoire par blocs plutôt qu'objet par objet, ce qui réduit de
> 10 à 40 % le temps passé dans le GC sur des programmes réels.

> [!NOTE]
> **Go 1.27** — les petites allocations (moins de 80 octets) sont jusqu'à
> 30 % moins coûteuses grâce à des fonctions d'allocation spécialisées par
> taille.
