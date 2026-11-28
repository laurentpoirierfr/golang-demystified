---
title: "II.9 Génériques"
weight: 90
---

# Chapitre II.9 — Génériques

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/generiques`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/generiques).

Les génériques, ou **paramètres de type**, sont arrivés avec **Go 1.18**, après
plus de dix ans de débats. Ils permettent d'écrire une fonction ou un type une
seule fois pour plusieurs types, en gardant la vérification à la compilation.

## Les fonctions génériques

Les paramètres de type se déclarent entre crochets, avant les paramètres
ordinaires. Chacun a une **contrainte**, qui indique les types acceptés :

```go
// Map applique f à chaque élément de s.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

longueurs := Map([]string{"go", "rust", "zig"}, func(s string) int { return len(s) })
textes := Map([]int{1, 2, 3}, strconv.Itoa)
fmt.Println(longueurs, strings.Join(textes, "-"))
// [2 4 3] 1-2-3
```

Le plus souvent, le compilateur **déduit** les types à partir des arguments. On
peut aussi les préciser : `Map[string, int](…)`.

## Les contraintes

Une contrainte est une interface. Elle limite les types acceptés, et donc les
opérations autorisées dans la fonction :

| Contrainte    | Types acceptés                     | Opérations permises       |
|---------------|------------------------------------|---------------------------|
| `any`         | tous                               | affectation, passage      |
| `comparable`  | les types comparables              | `==`, `!=`, clé de map    |
| `cmp.Ordered` | entiers, flottants, chaînes        | `<`, `<=`, `>`, `>=`      |

```go
func PlusGrand[T cmp.Ordered](s []T) T {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

fmt.Println(PlusGrand([]int{3, 9, 2}), PlusGrand([]string{"b", "c", "a"}))
// 9 c
```

### Les ensembles de types

Une interface utilisée comme contrainte peut lister des types avec `|`. Le
préfixe `~` accepte aussi tous les types dont le **type sous-jacent** est
celui indiqué, comme `type Euros float64` :

```go
type Nombre interface {
	~int | ~int64 | ~float64
}

func Somme[T Nombre](s []T) T {
	var total T // valeur zéro de T
	for _, v := range s {
		total += v
	}
	return total
}

type Euros float64

fmt.Println(Somme([]int{1, 2, 3}))         // 6
fmt.Println(Somme([]Euros{9.99, 0.01}))    // 10
```

Une interface qui contient un ensemble de types ne peut servir **que** de
contrainte, pas de type de variable.

## Les types génériques

Un type peut lui aussi avoir des paramètres de type. C'est le cas des
structures de données : piles, files, arbres, caches…

```go
type Pile[T any] struct {
	elements []T
}

func (p *Pile[T]) Empile(v T) { p.elements = append(p.elements, v) }

func (p *Pile[T]) Depile() (T, bool) {
	var zero T
	if len(p.elements) == 0 {
		return zero, false
	}
	v := p.elements[len(p.elements)-1]
	p.elements = p.elements[:len(p.elements)-1]
	return v, true
}

var p Pile[string]
p.Empile("a")
p.Empile("b")
v, ok := p.Depile()
fmt.Println(v, ok)
// b true
```

`var zero T` est la façon idiomatique d'obtenir la valeur zéro d'un paramètre
de type.

> [!NOTE]
> **Go 1.24** — les alias de types acceptent des paramètres de type :
>
> ```go
> type Ensemble[T comparable] = map[T]struct{}
> ```

## Les méthodes génériques

> [!NOTE]
> **Go 1.27** — une méthode peut déclarer **ses propres** paramètres de type,
> en plus de ceux de son type.

```go
type Liste[T any] struct{ elements []T }

// Transforme déclare son propre paramètre de type U.
func (l Liste[T]) Transforme[U any](f func(T) U) Liste[U] {
	return Liste[U]{elements: Map(l.elements, f)}
}

l := Liste[int]{elements: []int{1, 2, 3}}
carres := l.Transforme(func(n int) string { return strconv.Itoa(n * n) })
fmt.Println(carres.elements)
// [1 4 9]
```

Avant Go 1.27, il fallait écrire une fonction séparée,
`Transforme(l, f)`, ce qui empêchait d'enchaîner les appels. Une limite
demeure : une méthode générique ne peut pas servir à satisfaire une interface,
et une interface ne peut pas déclarer de méthode générique.

## Les contraintes récursives

> [!NOTE]
> **Go 1.26** — un type générique peut se référencer lui-même dans sa liste de
> paramètres de type.

Cela permet d'exprimer « un type qui sait s'additionner à lui-même » :

```go
type Additionnable[A Additionnable[A]] interface {
	Ajoute(A) A
}

type Vecteur struct{ X, Y int }

func (v Vecteur) Ajoute(w Vecteur) Vecteur { return Vecteur{v.X + w.X, v.Y + w.Y} }

func Total[A Additionnable[A]](valeurs ...A) A {
	var total A
	for _, v := range valeurs {
		total = total.Ajoute(v)
	}
	return total
}

fmt.Println(Total(Vecteur{1, 2}, Vecteur{3, 4}, Vecteur{5, 6}))
// {9 12}
```

## Quand utiliser les génériques ?

Les génériques sont utiles lorsque le **même code** s'applique à plusieurs
types :

- les fonctions sur les slices, les maps et les channels (voir les packages
  `slices`, `maps` et `iter`) ;
- les structures de données génériques ;
- les fonctions utilitaires qui ne dépendent pas du type manipulé.

Ils sont inutiles lorsque :

- une **interface** suffit : si la fonction appelle seulement des méthodes,
  acceptez l'interface directement ;
- l'implémentation **diffère** selon le type : écrivez des fonctions
  distinctes ou utilisez un type switch.

> « Write code, don't design types. » — Ian Lance Taylor, co-concepteur des
> génériques de Go

Une bonne règle : commencez par écrire le code pour un type concret. Lorsque
vous le dupliquez pour un second type, il est temps de le rendre générique.

## Exercices

Les solutions sont dans
[`examples/exercices/langage/generiques`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/generiques),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Un maximum ★

```go
func Max[T cmp.Ordered](s []T) (T, bool)
```

La même fonction sert pour des entiers, des flottants et des textes. Pour une
slice vide, renvoyez la valeur zéro de `T` et `false`.

### Transformer et filtrer ★★

```go
func Transforme[T, U any](s []T, f func(T) U) []U
func Filtre[T any](s []T, garde func(T) bool) []T
```

`Transforme([]int{1, 2, 3}, strconv.Itoa)` renvoie `["1" "2" "3"]` : les deux
paramètres de type sont déduits des arguments.

### Un ensemble générique ★★★

```go
type Ensemble[T comparable] struct{ /* … */ }

func NouvelEnsemble[T comparable](elements ...T) *Ensemble[T]
func (e *Ensemble[T]) Ajoute(v T)
func (e *Ensemble[T]) Contient(v T) bool
func (e *Ensemble[T]) Taille() int
func (e *Ensemble[T]) Union(autre *Ensemble[T]) *Ensemble[T]
func Trie[T cmp.Ordered](e *Ensemble[T]) []T
```

Pourquoi `Trie` est-elle une fonction, et non une méthode ? (Une méthode ne
peut pas ajouter de contrainte à celles du type : `comparable` ne permet pas
de trier.)
