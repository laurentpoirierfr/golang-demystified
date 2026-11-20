---
title: "Fonctions"
linkTitle: "5. Fonctions"
weight: 50
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/fonctions`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/fonctions).

## Déclarer une fonction

```go
func nom(paramètre type, …) typeDeRetour {
	…
}
```

Les paramètres consécutifs de même type peuvent partager leur type :
`func add(a, b int) int`. Les arguments sont **toujours passés par valeur** :
la fonction reçoit une copie. Pour modifier une variable de l'appelant, on
passe un pointeur. Les slices, les maps et les channels contiennent déjà une
référence vers leurs données : leur copie est peu coûteuse, et la fonction voit
les mêmes éléments.

Go n'a ni paramètres par défaut, ni paramètres nommés, ni surcharge : deux
fonctions d'un même package ne peuvent pas porter le même nom.

## Plusieurs valeurs de retour

Une fonction peut renvoyer plusieurs valeurs. C'est ainsi que Go signale les
erreurs : la dernière valeur de retour est une `error`, `nil` en cas de succès.

```go
func divise(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division par zéro")
	}
	return a / b, nil
}

q, err := divise(7, 2)
fmt.Println(q, err)
// 3 <nil>
```

### Les retours nommés

Les valeurs de retour peuvent être nommées. Elles sont alors déclarées en
début de fonction, initialisées à leur valeur zéro, et documentent ce que la
fonction renvoie :

```go
func statistiques(nombres []float64) (moyenne, ecart float64) {
	if len(nombres) == 0 {
		return // renvoie les valeurs zéro
	}
	var somme float64
	for _, n := range nombres {
		somme += n
	}
	moyenne = somme / float64(len(nombres))
	ecart = slices.Max(nombres) - slices.Min(nombres)
	return moyenne, ecart
}
```

Un `return` sans valeur (*naked return*) renvoie les variables nommées. Il est
à réserver aux fonctions courtes, car il nuit à la lisibilité. Les retours
nommés sont en revanche indispensables pour qu'un `defer` modifie la valeur
renvoyée, comme on le verra avec `recover`.

## Les fonctions variadiques

Le dernier paramètre peut accepter un nombre variable d'arguments, reçus sous
forme de slice. Une slice existante se passe en la faisant suivre de `...` :

```go
func somme(nombres ...int) int {
	total := 0
	for _, n := range nombres {
		total += n
	}
	return total
}

fmt.Println(somme(), somme(1, 2, 3)) // 0 6
valeurs := []int{4, 5, 6}
fmt.Println(somme(valeurs...))       // 15
```

`fmt.Println` et `append` sont des fonctions variadiques.

## Les fonctions sont des valeurs

Une fonction peut être affectée à une variable, passée en argument ou renvoyée
par une autre fonction. Son type s'écrit `func(paramètres) retours` :

```go
func applique(s []string, f func(string) string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		out = append(out, f(x))
	}
	return out
}

crier := func(s string) string { return strings.ToUpper(s) + " !" }
fmt.Println(applique([]string{"go", "zig"}, crier))
// [GO ! ZIG !]
fmt.Println(applique([]string{"Go"}, strings.ToLower))
// [go]
```

La valeur zéro d'un type fonction est `nil`. Appeler une fonction nil provoque
un `panic`.

## Les closures

Une fonction anonyme peut **capturer** les variables de son environnement.
Elle en garde une référence, et non une copie : les variables capturées
survivent tant que la closure existe.

```go
func compteur() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

c1 := compteur()
c2 := compteur()
fmt.Println(c1(), c1(), c1(), c2())
// 1 2 3 1
```

Chaque appel à `compteur` crée une nouvelle variable `n` : `c1` et `c2` sont
indépendants.

## Les itérateurs

Depuis **Go 1.23**, `range` accepte des fonctions. Un itérateur est une
fonction qui reçoit une fonction `yield` et l'appelle pour chaque élément.
Si `yield` renvoie `false`, la boucle de l'appelant s'est arrêtée (`break`,
`return`…) et l'itérateur doit s'arrêter aussi.

Le package `iter` définit les deux formes standard :

```go
type Seq[V any]     func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)
```

```go
// Pairs renvoie un itérateur sur les nombres pairs de 0 à max inclus.
func Pairs(max int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i <= max; i += 2 {
			if !yield(i) {
				return
			}
		}
	}
}

for n := range Pairs(10) {
	if n > 6 {
		break
	}
	fmt.Print(n, ";")
}
// 0;2;4;6;

fmt.Println(slices.Collect(Pairs(8)))
// [0 2 4 6 8]
```

Un `iter.Seq2` produit des paires, comme l'indice et la valeur d'une slice :

```go
func Indexe(texte string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, mot := range strings.Fields(texte) {
			if !yield(i+1, mot) {
				return
			}
		}
	}
}

for pos, mot := range Indexe("le gopher creuse") {
	fmt.Println(pos, mot)
}
// 1 le
// 2 gopher
// 3 creuse
```

Les itérateurs évitent de construire une slice intermédiaire, et permettent de
parcourir des séquences infinies ou coûteuses à produire. La bibliothèque
standard en propose de nombreux :

```go
for part := range strings.SplitSeq("a,b,c", ",") { … }   // Go 1.24
for i, v := range slices.Backward(s) { … }              // parcours à l'envers
for k := range maps.Keys(m) { … }
for k, v := range maps.All(m) { … }
lignes := slices.Collect(strings.Lines(texte))           // Go 1.24
```

> [!TIP]
> Avant Go 1.23, chaque bibliothèque inventait sa propre convention de
> parcours (`Next()`, `Each(func)`, channels…). Les itérateurs standard
> rendent ces API interchangeables : n'importe quel `iter.Seq` peut être
> collecté avec `slices.Collect`, trié avec `slices.Sorted`, etc.

## Les méthodes

Une méthode est une fonction attachée à un type, grâce à un **récepteur**
déclaré avant son nom :

```go
type Rectangle struct{ Largeur, Hauteur float64 }

func (r Rectangle) Aire() float64 { return r.Largeur * r.Hauteur }
```

Les méthodes, et le choix entre récepteur valeur et récepteur pointeur, sont
présentés dans le chapitre sur les structures.

## init

Chaque package peut déclarer une ou plusieurs fonctions `init()`, sans
paramètre ni retour. Elles sont exécutées automatiquement une seule fois, après
l'initialisation des variables du package et avant `main`. Elles sont utiles
pour enregistrer un pilote ou vérifier une configuration, mais à utiliser avec
modération : leur effet est implicite et difficile à tester.
