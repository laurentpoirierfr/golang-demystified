// Package fonctions contient les solutions des exercices du chapitre
// « Fonctions ».
package fonctions

// Exercice 1 ★ : une fonction variadique.

// Moyenne renvoie la moyenne de ses arguments, et faux s'il n'y en a pas.
func Moyenne(nombres ...float64) (float64, bool) {
	if len(nombres) == 0 {
		return 0, false
	}
	total := 0.0
	for _, n := range nombres {
		total += n
	}
	return total / float64(len(nombres)), true
}

// Exercice 2 ★★ : une fermeture.

// Compteur renvoie une fonction qui, à chaque appel, renvoie le nombre
// suivant à partir de depart. Deux compteurs sont indépendants.
func Compteur(depart int) func() int {
	n := depart - 1
	return func() int {
		n++
		return n
	}
}

// Exercice 3 ★★★ : mémoïser.

// Memoise renvoie une version de f qui garde ses résultats : f n'est appelée
// qu'une fois par argument.
func Memoise(f func(int) int) func(int) int {
	cache := map[int]int{}
	return func(n int) int {
		if v, ok := cache[n]; ok {
			return v
		}
		v := f(n)
		cache[n] = v
		return v
	}
}
