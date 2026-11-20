// Command mots affiche les dix mots les plus fréquents d'un fichier. Le même
// programme est écrit en Rust dans ../../rust/mots, pour comparer les deux
// langages.
package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode"
)

// Frequence est un mot et son nombre d'occurrences.
type Frequence struct {
	Mot    string
	Nombre int
}

// Compte renvoie les n mots les plus fréquents du texte.
func Compte(texte string, n int) []Frequence {
	compte := map[string]int{}
	for mot := range strings.FieldsFuncSeq(texte, func(r rune) bool { return !unicode.IsLetter(r) }) {
		compte[strings.ToLower(mot)]++
	}
	frequences := make([]Frequence, 0, len(compte))
	for mot, nombre := range compte {
		frequences = append(frequences, Frequence{mot, nombre})
	}
	slices.SortFunc(frequences, func(a, b Frequence) int {
		return cmp.Or(cmp.Compare(b.Nombre, a.Nombre), strings.Compare(a.Mot, b.Mot))
	})
	return frequences[:min(n, len(frequences))]
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage : mots FICHIER")
		os.Exit(2)
	}
	texte, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, f := range Compte(string(texte), 10) {
		fmt.Printf("%8d %s\n", f.Nombre, f.Mot)
	}
}
