// Package temps contient les solutions des exercices du chapitre « Le
// temps : time ».
package temps

import (
	"time"
	_ "time/tzdata" // la base des fuseaux, embarquée : le programme ne dépend pas du système
)

// Exercice 1 ★ : un âge.

// Age renvoie l'âge en années révolues, à la date aujourdhui.
func Age(naissance, aujourdhui time.Time) int {
	age := aujourdhui.Year() - naissance.Year()
	// L'anniversaire de cette année n'est pas encore passé.
	if aujourdhui.Month() < naissance.Month() ||
		aujourdhui.Month() == naissance.Month() && aujourdhui.Day() < naissance.Day() {
		age--
	}
	return age
}

// Exercice 2 ★★ : des jours ouvrés.

// JoursOuvres compte les jours du lundi au vendredi entre debut inclus et fin
// exclue.
func JoursOuvres(debut, fin time.Time) int {
	n := 0
	for j := debut; j.Before(fin); j = j.AddDate(0, 0, 1) { // AddDate, pas Add(24h) : les changements d'heure
		if w := j.Weekday(); w != time.Saturday && w != time.Sunday {
			n++
		}
	}
	return n
}

// Exercice 3 ★★★ : une date française, en UTC.

// LitDateParis lit une date au format « 20/11/2026 14:30 », exprimée à
// l'heure de Paris, et la renvoie en UTC.
func LitDateParis(s string) (time.Time, error) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.ParseInLocation("02/01/2006 15:04", s, paris)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
