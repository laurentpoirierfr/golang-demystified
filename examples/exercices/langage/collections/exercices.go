// Package collections contient les solutions des exercices du chapitre
// « Tableaux, slices et maps ».
package collections

// Exercice 1 ★ : inverser une slice.

// Inverse renvoie une nouvelle slice, dans l'ordre inverse. L'originale
// n'est pas modifiée.
func Inverse(s []int) []int {
	r := make([]int, len(s))
	for i, v := range s {
		r[len(s)-1-i] = v
	}
	return r
}

// Exercice 2 ★★ : compter des mots.

// Frequences compte les occurrences de chaque mot.
func Frequences(mots []string) map[string]int {
	f := make(map[string]int)
	for _, m := range mots {
		f[m]++ // la valeur zéro d'un int absent est 0
	}
	return f
}

// Exercice 3 ★★★ : dédoublonner en gardant l'ordre.

// Uniques renvoie les éléments de s sans doublon, dans l'ordre de leur
// première apparition.
func Uniques(s []string) []string {
	vus := make(map[string]bool, len(s))
	r := make([]string, 0, len(s))
	for _, v := range s {
		if !vus[v] {
			vus[v] = true
			r = append(r, v)
		}
	}
	return r
}
