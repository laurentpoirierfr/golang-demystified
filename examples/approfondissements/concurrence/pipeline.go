// Package concurrence illustre le chapitre « Concurrence en pratique ».
package concurrence

import (
	"context"
	"sync"
)

// Genere émet les nombres de 1 à n, et s'arrête si ctx est annulé.
func Genere(ctx context.Context, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 1; i <= n; i++ {
			select {
			case out <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Etape applique f à chaque valeur reçue : c'est un étage de pipeline.
func Etape[E, S any](ctx context.Context, in <-chan E, f func(E) S) <-chan S {
	out := make(chan S)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- f(v):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Fusionne regroupe plusieurs channels en un seul (fan-in).
func Fusionne[T any](ctx context.Context, entrees ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	for _, in := range entrees {
		wg.Go(func() {
			for v := range in {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Resultat associe une tâche à son résultat ou à son erreur.
type Resultat[E, S any] struct {
	Entree E
	Sortie S
	Err    error
}

// Pool traite les tâches avec n workers (fan-out), et renvoie les résultats
// dans l'ordre où ils sont produits.
func Pool[E, S any](ctx context.Context, n int, taches []E, f func(context.Context, E) (S, error)) <-chan Resultat[E, S] {
	jobs := make(chan E)
	resultats := make(chan Resultat[E, S])

	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			for t := range jobs {
				s, err := f(ctx, t)
				select {
				case resultats <- Resultat[E, S]{t, s, err}:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	go func() {
		defer close(jobs)
		for _, t := range taches {
			select {
			case jobs <- t:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultats)
	}()
	return resultats
}
