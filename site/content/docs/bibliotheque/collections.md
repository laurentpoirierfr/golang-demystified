---
title: "Collections : slices, maps, iter, cmp"
weight: 20
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/collections`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/collections).

Les packages `slices` et `maps` (**Go 1.21**), `cmp` (**Go 1.21**) et `iter`
(**Go 1.23**) forment la boîte à outils générique des collections. Le chapitre
« Tableaux, slices et maps » a présenté les bases ; celui-ci va plus loin.

## Trier

`slices.Sort` trie les types ordonnés. `slices.SortFunc` accepte une fonction
de comparaison qui renvoie un entier négatif, nul ou positif. `cmp.Compare`
fournit cette comparaison pour les types ordonnés, et `cmp.Or` permet de
chaîner les critères : il renvoie le premier résultat non nul.

```go
type Personne struct {
	Nom string
	Age int
}

slices.SortFunc(equipe, func(a, b Personne) int {
	return cmp.Or(
		cmp.Compare(b.Age, a.Age),      // âge décroissant
		strings.Compare(a.Nom, b.Nom),  // puis nom croissant
	)
})
// [{Dave 45} {Alice 31} {Carol 31} {Bob 27}]
```

`slices.SortStableFunc` préserve l'ordre des éléments égaux. Le package
`sort`, plus ancien, reste disponible mais n'est plus nécessaire.

## Chercher

```go
i := slices.IndexFunc(equipe, func(p Personne) bool { return p.Age > 40 })
// 3
slices.ContainsFunc(equipe, func(p Personne) bool { return p.Nom == "Eve" })
// false
slices.MaxFunc(equipe, func(a, b Personne) int { return cmp.Compare(a.Age, b.Age) })
// {Dave 45}
```

Sur une slice triée, `slices.BinarySearch` et `slices.BinarySearchFunc`
trouvent un élément en temps logarithmique.

## Modifier

```go
s := []int{1, 2, 3, 4, 5}
s = slices.Insert(s, 1, 10, 11)      // [1 10 11 2 3 4 5]
s = slices.Delete(s, 0, 2)           // [11 2 3 4 5]
s = slices.DeleteFunc(s, func(n int) bool { return n%2 == 0 })
                                     // [11 3 5]
slices.Compact([]int{1, 1, 2, 2, 2, 3, 1})
                                     // [1 2 3 1] : doublons consécutifs
```

Comme `append`, ces fonctions renvoient la slice modifiée : réaffectez
toujours le résultat. Depuis **Go 1.22**, `Delete`, `DeleteFunc` et `Compact`
remettent à zéro les éléments libérés en fin de tableau, ce qui permet au
ramasse-miettes de récupérer les objets qu'ils référençaient.

`slices.Chunk` (**Go 1.23**) découpe une slice en lots, par exemple pour des
insertions en base de données :

```go
for lot := range slices.Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3) {
	fmt.Print(lot)
}
// [1 2 3][4 5 6][7]
```

## Le package maps

```go
stock := map[string]int{"vis": 120, "écrou": 80, "clou": 0}
copie := maps.Clone(stock)
maps.DeleteFunc(copie, func(_ string, q int) bool { return q == 0 })
slices.Sorted(maps.Keys(copie))   // [vis écrou]
maps.Equal(stock, copie)          // false
```

`maps.Keys`, `maps.Values` et `maps.All` renvoient des itérateurs.
`maps.Collect` construit une map à partir d'un `iter.Seq2`, et `maps.Insert`
ajoute les paires d'un itérateur à une map existante.

## Itérateurs : les combiner

Les fonctions qui produisent ou consomment des itérateurs se combinent
librement :

| Produire                             | Consommer                       |
|--------------------------------------|---------------------------------|
| `slices.Values(s)`, `slices.All(s)`  | `slices.Collect(seq)`           |
| `slices.Backward(s)`                 | `slices.Sorted(seq)`            |
| `maps.Keys(m)`, `maps.Values(m)`, `maps.All(m)` | `slices.AppendSeq(s, seq)` |
| `strings.SplitSeq`, `strings.Lines`… | `maps.Collect(seq2)`            |
| `slices.Chunk(s, n)`                 | `for … range seq`               |

La bibliothèque standard ne fournit pas (encore) de `Filter` ou de `Map` sur
les itérateurs, mais ils s'écrivent en quelques lignes :

```go
// Filtre ne garde que les éléments qui satisfont garde.
func Filtre[V any](seq iter.Seq[V], garde func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if garde(v) && !yield(v) {
				return
			}
		}
	}
}

// Transforme applique f à chaque élément.
func Transforme[V, W any](seq iter.Seq[V], f func(V) W) iter.Seq[W] {
	return func(yield func(W) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

nombres := slices.Values([]int{1, 2, 3, 4, 5, 6})
pairs := Filtre(nombres, func(n int) bool { return n%2 == 0 })
carres := Transforme(pairs, func(n int) int { return n * n })
fmt.Println(slices.Collect(carres))
// [4 16 36]
```

Rien n'est calculé avant la consommation finale par `slices.Collect` : chaque
élément traverse toute la chaîne, sans slice intermédiaire.

### Itérateurs « pull »

Un `iter.Seq` *pousse* ses valeurs vers la boucle. `iter.Pull` le convertit en
une fonction `suivant()` que l'on appelle à la demande, ce qui est utile pour
parcourir deux séquences en parallèle :

```go
suivant, stop := iter.Pull(slices.Values([]string{"a", "b"}))
defer stop()
for {
	v, ok := suivant()
	if !ok {
		break
	}
	fmt.Print(v)
}
// ab
```

## cmp

Le package `cmp` définit la contrainte `cmp.Ordered`, ainsi que :

- `cmp.Compare(a, b)` : renvoie -1, 0 ou +1 ;
- `cmp.Less(a, b)` : `a < b`, en traitant correctement `NaN` ;
- `cmp.Or(valeurs...)` (**Go 1.22**) : la première valeur non nulle. C'est
  l'idiome pour les valeurs par défaut :

```go
hote := cmp.Or(config["hote"], "localhost")
port := cmp.Or(os.Getenv("PORT"), "8080")
```

## Autres structures

- `container/heap` : tas binaire (file de priorité) ;
- `container/list` : liste doublement chaînée ;
- `container/ring` : liste circulaire ;
- `unique` (**Go 1.23**) : `unique.Make(v)` *interne* une valeur comparable.
  Deux handles de valeurs égales sont identiques, ce qui économise la mémoire
  et accélère les comparaisons ;
- `weak` (**Go 1.24**) : pointeurs faibles, qui n'empêchent pas le
  ramasse-miettes de libérer un objet. Ils servent à construire des caches.
