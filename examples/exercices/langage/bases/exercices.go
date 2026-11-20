// Package bases contient les solutions des exercices du chapitre
// « Variables, constantes et types ». Les tests (exercices_test.go) servent
// de spécification : écrivez votre version, puis lancez go test.
package bases

import "fmt"

// Exercice 1 ★ : convertir une température.

// Fahrenheit convertit des degrés Celsius en degrés Fahrenheit.
func Fahrenheit(celsius float64) float64 {
	return celsius*9/5 + 32
}

// Exercice 2 ★★ : des constantes avec iota.

// Unités de taille : chaque unité vaut 1 024 fois la précédente.
const (
	Octet int64 = 1 << (10 * iota)
	Kio
	Mio
	Gio
	Tio
)

// FormateTaille écrit une taille avec l'unité la plus grande qui convient,
// et une décimale : 1536 → « 1.5 Kio ».
func FormateTaille(n int64) string {
	switch {
	case n >= Tio:
		return fmt.Sprintf("%.1f Tio", float64(n)/float64(Tio))
	case n >= Gio:
		return fmt.Sprintf("%.1f Gio", float64(n)/float64(Gio))
	case n >= Mio:
		return fmt.Sprintf("%.1f Mio", float64(n)/float64(Mio))
	case n >= Kio:
		return fmt.Sprintf("%.1f Kio", float64(n)/float64(Kio))
	}
	return fmt.Sprintf("%d o", n)
}

// Exercice 3 ★★★ : détecter un dépassement.

// AdditionneInt8 additionne deux int8, et indique si le résultat dépasse ce
// que peut contenir un int8 (de -128 à 127).
func AdditionneInt8(a, b int8) (int8, bool) {
	somme := int16(a) + int16(b) // le calcul dans un type plus grand ne déborde pas
	return int8(somme), somme > 127 || somme < -128
}
