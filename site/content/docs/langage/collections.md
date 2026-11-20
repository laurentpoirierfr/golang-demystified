---
title: "Tableaux, slices et maps"
linkTitle: "4. Tableaux, slices et maps"
weight: 40
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/collections`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/collections).

## Les tableaux

Un tableau a une **taille fixe**, qui fait partie de son type : `[3]int` et
`[4]int` sont deux types différents. Un tableau est une **valeur** : l'affecter
ou le passer à une fonction le copie entièrement.

```go
var a [3]int
b := [...]string{"Go", "Rust", "Zig"} // taille déduite : 3
a[1] = 42

c := a // copie complète
c[0] = 7
fmt.Println(a, c, len(b))
// [0 42 0] [7 42 0] 3
```

Les tableaux sont rarement manipulés directement. On les rencontre comme clés
de map (ils sont comparables), pour des données de taille fixe (une empreinte
SHA-256 est un `[32]byte`) et, surtout, sous les slices.

## Les slices

Une slice est une **vue** sur un tableau. C'est une petite structure de trois
champs : un pointeur vers le tableau sous-jacent, une longueur (`len`) et une
capacité (`cap`).

```go
s := []int{1, 2, 3}
s = append(s, 4, 5)
fmt.Println(s, len(s))
// [1 2 3 4 5] 5
```

L'expression `s[début:fin]` crée une nouvelle slice qui **partage** le même
tableau : modifier l'une modifie l'autre.

```go
t := s[1:3] // éléments d'indice 1 et 2
t[0] = 20
fmt.Println(t, s)
// [20 3] [1 20 3 4 5]
```

Les bornes peuvent être omises : `s[:2]`, `s[2:]`, `s[:]`.

### append et la capacité

`append` ajoute des éléments à la fin. Si la capacité est suffisante, il écrit
dans le tableau existant. Sinon, il alloue un tableau plus grand, y copie les
éléments, et renvoie une slice qui pointe vers ce nouveau tableau. C'est
pourquoi il faut **toujours** réaffecter son résultat : `s = append(s, x)`.

Lorsque la taille finale est connue, `make` permet de réserver la capacité à
l'avance et d'éviter les réallocations :

```go
s := make([]int, 0, 10) // longueur 0, capacité 10
for i := range 4 {
	s = append(s, i*i)
}
fmt.Println(s, len(s), cap(s))
// [0 1 4 9] 4 10
```

> [!WARNING]
> Deux `append` sur une même slice qui a de la capacité libre écrivent **au
> même emplacement** :
>
> ```go
> base := make([]int, 3, 10)
> a := append(base, 1)
> b := append(base, 2) // écrase a[3] !
> fmt.Println(a[3], b[3])
> // 2 2
> ```
>
> L'expression à trois indices `base[:3:3]` limite la capacité à 3 et force
> `append` à copier : `a` et `b` deviennent indépendants.

### Copier une slice

`copy(dst, src)` copie les éléments et renvoie le nombre d'éléments copiés.
`slices.Clone` crée une copie indépendante en une ligne :

```go
src := []int{1, 2, 3}
dst := make([]int, len(src))
n := copy(dst, src)
dst[0] = 100
fmt.Println(n, src, dst)
// 3 [1 2 3] [100 2 3]

clone := slices.Clone(src)
```

### La slice nil

La valeur zéro d'une slice est `nil`. Une slice nil a une longueur de 0, se
parcourt avec `range` et accepte `append` : il est inutile de l'initialiser.

```go
var resultats []string // nil
for _, x := range donnees {
	if garder(x) {
		resultats = append(resultats, x)
	}
}
```

### Le package slices

Depuis **Go 1.21**, le package générique `slices` regroupe les opérations
courantes, qu'il fallait auparavant écrire à la main :

```go
s := []int{5, 2, 8, 1}
slices.Sort(s)
i, trouve := slices.BinarySearch(s, 5)
fmt.Println(s, i, trouve)
// [1 2 5 8] 2 true
fmt.Println(slices.Contains(s, 8), slices.Index(s, 2), slices.Max(s))
// true 1 8
```

On y trouve aussi `Equal`, `Reverse`, `Insert`, `Delete`, `Compact`,
`SortFunc`, `IndexFunc`, `Chunk`, `Collect`… Le chapitre sur le package
`slices` les détaille.

## Les maps

Une map associe des clés à des valeurs. Les clés doivent être d'un type
**comparable** avec `==` : nombres, chaînes, booléens, pointeurs, tableaux,
structures composées de champs comparables… mais pas les slices ni les maps.

```go
ages := map[string]int{
	"alice": 31,
	"bob":   27,
}
ages["carol"] = 45
delete(ages, "bob")
```

Lire une clé absente renvoie la valeur zéro. La forme à deux valeurs
(*comma ok*) indique si la clé existe :

```go
age, ok := ages["bob"]
fmt.Println(age, ok, len(ages))
// 0 false 2
```

La valeur zéro permet d'écrire un compteur sans tester l'existence de la clé :

```go
mots := []string{"go", "rust", "go", "zig", "go"}
compte := make(map[string]int)
for _, m := range mots {
	compte[m]++
}
fmt.Println(compte)
// map[go:3 rust:1 zig:1]
```

### L'ordre de parcours

L'ordre de parcours d'une map avec `range` n'est **pas défini** : il change
d'une exécution à l'autre, volontairement. Pour un ordre stable, triez les
clés :

```go
for _, nom := range slices.Sorted(maps.Keys(ages)) {
	fmt.Println(nom, ages[nom])
}
// alice 31
// carol 45
```

> [!NOTE]
> **Go 1.23** — `maps.Keys` et `maps.Values` renvoient des itérateurs, et
> `slices.Sorted` les collecte dans une slice triée.

### La map nil

Une map nil peut être lue, mais pas écrite : une écriture provoque un `panic`.
Initialisez-la avec `make` ou un littéral.

```go
var m map[string]int
fmt.Println(m["absent"], len(m)) // 0 0
// m["x"] = 1 → panic: assignment to entry in nil map
m = make(map[string]int)
m["x"] = 1
```

> [!CAUTION]
> Une map n'est pas protégée contre les accès concurrents. Si plusieurs
> goroutines l'écrivent, protégez-la par un `sync.Mutex`, ou utilisez
> `sync.Map` pour certains cas particuliers. Le runtime détecte souvent ces
> accès et arrête le programme (`fatal error: concurrent map writes`).

### Ensembles

Go n'a pas de type ensemble. On utilise une map dont la valeur est une
structure vide, qui n'occupe aucune mémoire :

```go
vus := make(map[string]struct{})
vus["go"] = struct{}{}
if _, ok := vus["go"]; ok {
	fmt.Println("déjà vu")
}
```

## clear, min et max

Depuis **Go 1.21**, trois fonctions intégrées simplifient le code courant :

```go
s := []int{1, 2, 3}
m := map[string]int{"a": 1}
clear(s) // remet chaque élément à sa valeur zéro : [0 0 0]
clear(m) // supprime toutes les entrées

fmt.Println(min(3, 1, 2), max(2.5, 1))
// 1 2.5
```
