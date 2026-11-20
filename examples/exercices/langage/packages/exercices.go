// Package packages contient les solutions des exercices du chapitre
// « Packages et modules ».
package packages

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Exercice 1 ★ : encapsuler.

// Temperature garde une valeur en kelvins, jamais négative : le champ n'est
// pas exporté, seul le constructeur peut le fixer.
type Temperature struct {
	kelvins float64
}

// DepuisCelsius crée une température, ou renvoie une erreur sous le zéro
// absolu.
func DepuisCelsius(c float64) (Temperature, error) {
	if c < -273.15 {
		return Temperature{}, fmt.Errorf("%v °C : sous le zéro absolu", c)
	}
	return Temperature{kelvins: c + 273.15}, nil
}

// Celsius renvoie la température en degrés Celsius.
func (t Temperature) Celsius() float64 { return t.kelvins - 273.15 }

// Exercice 2 ★★ : un registre, comme les pilotes de database/sql.

var (
	mu       sync.Mutex
	registre = map[string]func(string) string{}
)

// Enregistre ajoute un format sous un nom. Elle panique si le nom est déjà
// pris : c'est une erreur de programmation, détectée au démarrage (les
// enregistrements se font dans des fonctions init).
func Enregistre(nom string, f func(string) string) {
	mu.Lock()
	defer mu.Unlock()
	if _, existe := registre[nom]; existe {
		panic("format déjà enregistré : " + nom)
	}
	registre[nom] = f
}

// Formate applique le format enregistré sous ce nom.
func Formate(nom, texte string) (string, error) {
	mu.Lock()
	f, ok := registre[nom]
	mu.Unlock()
	if !ok {
		return "", fmt.Errorf("format inconnu : %s", nom)
	}
	return f(texte), nil
}

// Formats renvoie les noms enregistrés, triés.
func Formats() []string {
	mu.Lock()
	defer mu.Unlock()
	return slices.Sorted(maps.Keys(registre))
}
