// Package controle contient les solutions des exercices du chapitre
// « Contrôle de flux ».
package controle

import "strconv"

// Exercice 1 ★ : FizzBuzz.

// FizzBuzz renvoie le mot de n : « Fizz » pour un multiple de 3, « Buzz »
// pour un multiple de 5, « FizzBuzz » pour les deux, n sinon.
func FizzBuzz(n int) string {
	switch {
	case n%15 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	}
	return strconv.Itoa(n)
}

// Exercice 2 ★★ : une mention.

// Mention renvoie la mention d'une note sur 20, ou « invalide » hors de
// [0, 20].
func Mention(note float64) string {
	switch {
	case note < 0 || note > 20:
		return "invalide"
	case note >= 16:
		return "très bien"
	case note >= 14:
		return "bien"
	case note >= 12:
		return "assez bien"
	case note >= 10:
		return "passable"
	default:
		return "ajourné"
	}
}

// Exercice 3 ★★★ : la suite de Syracuse.

// Syracuse compte les étapes pour atteindre 1 depuis n (n > 0) : n/2 si n
// est pair, 3n+1 sinon. Elle renvoie -1 si n <= 0.
func Syracuse(n int) int {
	if n <= 0 {
		return -1
	}
	etapes := 0
	for n != 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		etapes++
	}
	return etapes
}
