// Package interfaces contient les solutions des exercices du chapitre
// « Interfaces ».
package interfaces

import (
	"fmt"
	"io"
	"math"
	"unicode"
	"unicode/utf8"
)

// Exercice 1 ★ : une interface et deux types.

// Forme est satisfaite implicitement par tout type qui a une méthode Aire.
type Forme interface {
	Aire() float64
}

// Carre et Cercle satisfont Forme.
type (
	Carre  struct{ Cote float64 }
	Cercle struct{ Rayon float64 }
)

func (c Carre) Aire() float64  { return c.Cote * c.Cote }
func (c Cercle) Aire() float64 { return math.Pi * c.Rayon * c.Rayon }

// AireTotale additionne les aires de formes de types quelconques.
func AireTotale(formes ...Forme) float64 {
	total := 0.0
	for _, f := range formes {
		total += f.Aire()
	}
	return total
}

// Exercice 2 ★★ : fmt.Stringer.

// Duree est un nombre de secondes, qui s'affiche « 1h02m05s ».
type Duree int

// String satisfait fmt.Stringer : fmt.Println et %v l'utilisent.
func (d Duree) String() string {
	h, m, s := int(d)/3600, int(d)%3600/60, int(d)%60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// Exercice 3 ★★★ : décorer un io.Reader.

// majuscules enveloppe un io.Reader, et met en majuscules ce qu'il lit.
type majuscules struct {
	source io.Reader
}

// Majuscules renvoie un io.Reader qui lit source en majuscules. Le texte
// est supposé en ASCII ou en UTF-8 sans caractère coupé entre deux lectures.
func Majuscules(source io.Reader) io.Reader { return majuscules{source} }

func (m majuscules) Read(p []byte) (int, error) {
	n, err := m.source.Read(p)
	for i := 0; i < n; {
		r, taille := utf8.DecodeRune(p[i:n])
		if haut := unicode.ToUpper(r); haut != r && utf8.RuneLen(haut) == taille {
			utf8.EncodeRune(p[i:], haut) // même longueur : on remplace sur place
		}
		i += taille
	}
	return n, err
}
