// Package context contient les solutions des exercices du chapitre
// « Annulation et délais : context ».
package context

import (
	"context"
	"errors"
	"time"
)

// Exercice 1 ★ : une attente annulable.

// Attends attend la durée d, sauf si ctx est annulé avant : elle renvoie
// alors la cause, context.Canceled ou context.DeadlineExceeded.
func Attends(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Exercice 2 ★★ : une valeur typée.

// cleUtilisateur est un type non exporté : aucun autre package ne peut
// créer la même clé, donc aucune collision.
type cleUtilisateur struct{}

// AvecUtilisateur renvoie un contexte qui porte le nom de l'utilisateur.
func AvecUtilisateur(ctx context.Context, nom string) context.Context {
	return context.WithValue(ctx, cleUtilisateur{}, nom)
}

// Utilisateur lit le nom porté par le contexte.
func Utilisateur(ctx context.Context) (string, bool) {
	nom, ok := ctx.Value(cleUtilisateur{}).(string)
	return nom, ok
}

// Exercice 3 ★★★ : le premier qui réussit.

// Premier lance toutes les sources en parallèle, renvoie le premier résultat
// réussi, et annule les autres. Si toutes échouent, il renvoie leurs erreurs
// réunies.
func Premier(ctx context.Context, sources ...func(context.Context) (string, error)) (string, error) {
	ctx, annule := context.WithCancel(ctx)
	defer annule() // arrête les sources encore en cours

	type resultat struct {
		valeur string
		err    error
	}
	resultats := make(chan resultat, len(sources)) // tamponné : aucune goroutine ne reste bloquée
	for _, s := range sources {
		go func() {
			v, err := s(ctx)
			resultats <- resultat{v, err}
		}()
	}
	var errs []error
	for range sources {
		r := <-resultats
		if r.err == nil {
			return r.valeur, nil
		}
		errs = append(errs, r.err)
	}
	return "", errors.Join(errs...)
}
