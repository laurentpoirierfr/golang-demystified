// Package collections contient les solutions des exercices du chapitre
// « Collections : slices, maps, iter, cmp ».
package collections

import (
	"cmp"
	"iter"
	"slices"
)

// Exercice 1 ★ : trier sur deux critères.

// Personne a un nom et un âge.
type Personne struct {
	Nom string
	Age int
}

// TrieParAge trie par âge croissant puis, à âge égal, par nom. Le tri est
// fait sur place.
func TrieParAge(p []Personne) {
	slices.SortFunc(p, func(a, b Personne) int {
		return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Nom, b.Nom))
	})
}

// Exercice 2 ★★ : regrouper.

// Regroupe range les éléments par clé, en gardant leur ordre.
func Regroupe[T any, K comparable](s []T, cle func(T) K) map[K][]T {
	groupes := make(map[K][]T)
	for _, v := range s {
		k := cle(v)
		groupes[k] = append(groupes[k], v)
	}
	return groupes
}

// Exercice 3 ★★★ : combiner des itérateurs.

// Naturels produit 1, 2, 3… sans fin : c'est celui qui consomme qui
// s'arrête.
func Naturels() iter.Seq[int] {
	return func(yield func(int) bool) {
		for n := 1; ; n++ {
			if !yield(n) {
				return
			}
		}
	}
}

// Filtre ne laisse passer que les valeurs pour lesquelles garde est vrai.
func Filtre[T any](seq iter.Seq[T], garde func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if garde(v) && !yield(v) {
				return
			}
		}
	}
}

// Premiers s'arrête après n valeurs.
func Premiers[T any](seq iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		i := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			if i++; i == n {
				return
			}
		}
	}
}
