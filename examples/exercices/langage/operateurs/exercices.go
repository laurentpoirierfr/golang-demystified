// Package operateurs contient les solutions des exercices du chapitre
// « Les opérateurs ».
package operateurs

// Exercice 1 ★ : la parité.

// EstPair indique si n est pair, y compris pour un nombre négatif.
func EstPair(n int) bool {
	return n%2 == 0 // -3 % 2 vaut -1 en Go : comparer à 1 serait faux
}

// Exercice 2 ★★ : des droits en bits.

// Droit est un ensemble de droits, un bit par droit.
type Droit uint8

const (
	Lecture Droit = 1 << iota
	Ecriture
	Execution
)

// A indique si d contient tous les droits demandés.
func A(d, demandes Droit) bool { return d&demandes == demandes }

// Retire enlève des droits.
func Retire(d, retires Droit) Droit { return d &^ retires }

// Exercice 3 ★★★ : compter les bits.

// Bits compte les bits à 1 de n, sans le package math/bits.
func Bits(n uint64) int {
	c := 0
	for n != 0 {
		n &= n - 1 // efface le bit à 1 le plus faible
		c++
	}
	return c
}
