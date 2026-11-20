// Package erreurs contient les solutions des exercices du chapitre
// « Gestion des erreurs ».
package erreurs

import (
	"errors"
	"fmt"
	"strconv"
)

// Exercice 1 ★ : envelopper une erreur.

// ErrHorsLimites signale un âge impossible.
var ErrHorsLimites = errors.New("âge hors limites")

// LitAge convertit un texte en âge (de 0 à 150). Les erreurs gardent leur
// cause : errors.Is les retrouve.
func LitAge(s string) (int, error) {
	age, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("âge %q : %w", s, err)
	}
	if age < 0 || age > 150 {
		return 0, fmt.Errorf("âge %d : %w", age, ErrHorsLimites)
	}
	return age, nil
}

// Exercice 2 ★★ : un type d'erreur.

// ErrChamp décrit un champ invalide.
type ErrChamp struct {
	Champ, Raison string
}

func (e *ErrChamp) Error() string { return e.Champ + " : " + e.Raison }

// Utilisateur est la donnée à valider.
type Utilisateur struct {
	Nom   string
	Email string
}

// Valide renvoie toutes les erreurs, réunies par errors.Join.
func Valide(u Utilisateur) error {
	var errs []error
	if u.Nom == "" {
		errs = append(errs, &ErrChamp{"nom", "obligatoire"})
	}
	if !contientArobase(u.Email) {
		errs = append(errs, &ErrChamp{"email", "adresse invalide"})
	}
	return errors.Join(errs...) // nil s'il n'y a aucune erreur
}

func contientArobase(s string) bool {
	for _, r := range s {
		if r == '@' {
			return true
		}
	}
	return false
}

// Exercice 3 ★★★ : transformer une panique en erreur.

// Protege appelle f, et renvoie une erreur si f panique.
func Protege(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panique : %v", r) // la valeur de retour nommée est modifiée
		}
	}()
	f()
	return nil
}
