// Package concurrence contient les solutions des exercices du chapitre
// « Goroutines et channels ».
package concurrence

import "sync"

// Exercice 1 ★ : une somme en parallèle.

// SommeParallele découpe nombres en parts morceaux, les additionne chacun
// dans une goroutine, puis réunit les sommes partielles.
func SommeParallele(nombres []int, parts int) int {
	if parts < 1 {
		parts = 1
	}
	sommes := make([]int, parts) // une case par goroutine : aucun partage
	taille := (len(nombres) + parts - 1) / parts
	var wg sync.WaitGroup
	for i := range parts {
		debut, fin := min(i*taille, len(nombres)), min((i+1)*taille, len(nombres))
		wg.Go(func() {
			for _, n := range nombres[debut:fin] {
				sommes[i] += n
			}
		})
	}
	wg.Wait()
	total := 0
	for _, s := range sommes {
		total += s
	}
	return total
}

// Exercice 2 ★★ : un pipeline.

// Genere envoie 1, 2… n sur le channel renvoyé, puis le ferme.
func Genere(n int) <-chan int {
	sortie := make(chan int)
	go func() {
		defer close(sortie)
		for i := 1; i <= n; i++ {
			sortie <- i
		}
	}()
	return sortie
}

// Carres envoie le carré de chaque valeur reçue, et ferme sa sortie quand
// l'entrée est fermée.
func Carres(entree <-chan int) <-chan int {
	sortie := make(chan int)
	go func() {
		defer close(sortie)
		for v := range entree {
			sortie <- v * v
		}
	}()
	return sortie
}

// Exercice 3 ★★★ : un groupe de travailleurs.

// Traite applique f à chaque tâche, avec au plus n appels simultanés, et
// renvoie les résultats dans l'ordre des tâches.
func Traite[T, R any](taches []T, n int, f func(T) R) []R {
	resultats := make([]R, len(taches))
	indices := make(chan int)
	var wg sync.WaitGroup
	for range max(n, 1) {
		wg.Go(func() {
			for i := range indices {
				resultats[i] = f(taches[i]) // chaque case n'est écrite qu'une fois
			}
		})
	}
	for i := range taches {
		indices <- i
	}
	close(indices)
	wg.Wait()
	return resultats
}
