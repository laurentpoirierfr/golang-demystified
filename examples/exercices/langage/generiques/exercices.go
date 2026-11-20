// Package generiques contient les solutions des exercices du chapitre
// « Génériques ».
package generiques

import (
	"cmp"
	"maps"
	"slices"
)

// Exercice 1 ★ : un maximum générique.

// Max renvoie le plus grand élément, et faux si s est vide.
func Max[T cmp.Ordered](s []T) (T, bool) {
	var max T
	if len(s) == 0 {
		return max, false
	}
	max = s[0]
	for _, v := range s[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// Exercice 2 ★★ : transformer et filtrer.

// Transforme applique f à chaque élément.
func Transforme[T, U any](s []T, f func(T) U) []U {
	r := make([]U, 0, len(s))
	for _, v := range s {
		r = append(r, f(v))
	}
	return r
}

// Filtre garde les éléments pour lesquels garde est vrai.
func Filtre[T any](s []T, garde func(T) bool) []T {
	var r []T
	for _, v := range s {
		if garde(v) {
			r = append(r, v)
		}
	}
	return r
}

// Exercice 3 ★★★ : un type générique.

// Ensemble est un ensemble d'éléments comparables.
type Ensemble[T comparable] struct {
	elements map[T]struct{}
}

// NouvelEnsemble crée un ensemble contenant les éléments donnés.
func NouvelEnsemble[T comparable](elements ...T) *Ensemble[T] {
	e := &Ensemble[T]{elements: make(map[T]struct{}, len(elements))}
	for _, v := range elements {
		e.Ajoute(v)
	}
	return e
}

func (e *Ensemble[T]) Ajoute(v T)        { e.elements[v] = struct{}{} }
func (e *Ensemble[T]) Contient(v T) bool { _, ok := e.elements[v]; return ok }
func (e *Ensemble[T]) Taille() int       { return len(e.elements) }

// Union renvoie un nouvel ensemble, avec les éléments des deux.
func (e *Ensemble[T]) Union(autre *Ensemble[T]) *Ensemble[T] {
	u := NouvelEnsemble[T]()
	maps.Copy(u.elements, e.elements)
	maps.Copy(u.elements, autre.elements)
	return u
}

// Trie renvoie les éléments d'un ensemble, dans l'ordre croissant. C'est une
// fonction, non une méthode : elle demande une contrainte plus forte
// (cmp.Ordered) que le type (comparable).
func Trie[T cmp.Ordered](e *Ensemble[T]) []T {
	return slices.Sorted(maps.Keys(e.elements))
}
