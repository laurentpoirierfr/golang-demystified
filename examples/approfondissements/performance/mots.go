// Package performance illustre le chapitre « Profilage et performance » :
// deux versions d'un même calcul, la naïve et l'optimisée, mesurées par des
// benchmarks.
package performance

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Frequence associe un mot à son nombre d'occurrences.
type Frequence struct {
	Mot    string
	Compte int
}

var separateurs = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// MotsFrequentsNaif lit tout le texte en mémoire, le découpe avec une
// expression régulière, et met chaque mot en minuscules : simple et correct,
// mais coûteux.
func MotsFrequentsNaif(r io.Reader, n int) []Frequence {
	data, _ := io.ReadAll(r)
	compte := map[string]int{}
	for _, mot := range separateurs.Split(string(data), -1) {
		if mot != "" {
			compte[strings.ToLower(mot)]++
		}
	}
	return lesPlusFrequents(compte, n)
}

// MotsFrequents lit le texte en flux et travaille sur des octets : il n'alloue
// qu'une chaîne par mot distinct, et non par occurrence.
func MotsFrequents(r io.Reader, n int) []Frequence {
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)
	compte := make(map[string]*int, 1024) // capacité initiale : moins de réallocations
	var minuscules []byte                 // tampon réutilisé pour chaque mot
	for sc.Scan() {
		for mot := range bytes.FieldsFuncSeq(sc.Bytes(), separateur) {
			minuscules = minuscules[:0]
			for _, r := range string(mot) { // conversion sans allocation
				minuscules = utf8.AppendRune(minuscules, unicode.ToLower(r))
			}
			// m[string(b)] en lecture n'alloue pas : le compilateur évite la copie.
			if c, ok := compte[string(minuscules)]; ok {
				*c++
				continue
			}
			un := 1
			compte[string(minuscules)] = &un // nouveau mot : une seule allocation
		}
	}
	valeurs := make(map[string]int, len(compte))
	for mot, c := range compte {
		valeurs[mot] = *c
	}
	return lesPlusFrequents(valeurs, n)
}

func separateur(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }

func lesPlusFrequents(compte map[string]int, n int) []Frequence {
	freqs := make([]Frequence, 0, len(compte))
	for mot, c := range compte {
		freqs = append(freqs, Frequence{mot, c})
	}
	slices.SortFunc(freqs, func(a, b Frequence) int {
		if a.Compte != b.Compte {
			return b.Compte - a.Compte
		}
		return strings.Compare(a.Mot, b.Mot)
	})
	return freqs[:min(n, len(freqs))]
}

// ConcateneNaif construit une chaîne par concaténations successives : chaque
// « + » recopie tout ce qui précède.
func ConcateneNaif(parts []string) string {
	s := ""
	for _, p := range parts {
		s += p + ","
	}
	return s
}

// Concatene calcule la taille finale, puis remplit un seul tampon.
func Concatene(parts []string) string {
	taille := 0
	for _, p := range parts {
		taille += len(p) + 1
	}
	var b strings.Builder
	b.Grow(taille)
	for _, p := range parts {
		b.WriteString(p)
		b.WriteByte(',')
	}
	return b.String()
}
