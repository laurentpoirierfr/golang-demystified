// Package texte contient les solutions des exercices du chapitre « Texte :
// fmt, strings, strconv ».
package texte

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"unicode"
)

// Exercice 1 ★ : des initiales.

// Initiales renvoie les initiales d'un nom, en majuscules et suivies d'un
// point : « ada lovelace » → « A.L. ». Elle travaille sur des runes, pour
// accepter « Émile Zola ».
func Initiales(nom string) string {
	var b strings.Builder
	for _, mot := range strings.Fields(nom) {
		for _, r := range mot {
			b.WriteRune(unicode.ToUpper(r))
			b.WriteByte('.')
			break // seule la première rune compte
		}
	}
	return b.String()
}

// Exercice 2 ★★ : un palindrome.

// EstPalindrome indique si s se lit dans les deux sens, en ignorant la
// casse, les espaces et la ponctuation.
func EstPalindrome(s string) bool {
	var lettres []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			lettres = append(lettres, unicode.ToLower(r))
		}
	}
	for i, j := 0, len(lettres)-1; i < j; i, j = i+1, j-1 {
		if lettres[i] != lettres[j] {
			return false
		}
	}
	return true
}

// Exercice 3 ★★★ : aligner des colonnes.

// Tableau aligne des lignes en colonnes, à gauche, séparées par deux espaces
// au moins.
func Tableau(lignes [][]string) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, l := range lignes {
		fmt.Fprintln(w, strings.Join(l, "\t")) // une tabulation entre les cellules
	}
	w.Flush()
	return b.String()
}
