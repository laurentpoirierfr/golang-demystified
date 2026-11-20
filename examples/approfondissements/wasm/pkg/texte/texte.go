// Package texte analyse un texte : mots, phrases, temps de lecture, mots les
// plus fréquents. Il n'a aucune dépendance : le même code tourne sur le
// serveur, dans le navigateur (WebAssembly) et dans un greffon WASI, compilé
// par Go ou par TinyGo.
package texte

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MotsParMinute est la vitesse de lecture moyenne retenue.
const MotsParMinute = 230

// Frequence est un mot et son nombre d'occurrences.
type Frequence struct {
	Mot    string `json:"mot"`
	Nombre int    `json:"nombre"`
}

// Stats résume un texte.
type Stats struct {
	Caracteres     int         `json:"caracteres"`
	Mots           int         `json:"mots"`
	Phrases        int         `json:"phrases"`
	MinutesLecture int         `json:"minutes_lecture"`
	Frequents      []Frequence `json:"frequents"`
}

// motsVides ne comptent pas parmi les mots fréquents.
var motsVides = map[string]bool{
	"le": true, "la": true, "les": true, "un": true, "une": true, "des": true, "de": true, "du": true,
	"et": true, "ou": true, "en": true, "à": true, "a": true, "au": true, "aux": true, "est": true,
	"il": true, "elle": true, "on": true, "que": true, "qui": true, "ne": true, "pas": true, "pour": true,
	"par": true, "sur": true, "dans": true, "se": true, "ce": true, "son": true, "sa": true, "ses": true,
	"the": true, "and": true, "of": true, "to": true, "is": true, "in": true,
}

// Analyse calcule les statistiques d'un texte. n est le nombre de mots
// fréquents à renvoyer.
func Analyse(s string, n int) Stats {
	st := Stats{Caracteres: utf8.RuneCountInString(s)}
	compte := map[string]int{}
	dansPhrase := false
	// Un mot est une suite de lettres, de chiffres, d'apostrophes ou de
	// traits d'union : « aujourd'hui » et « peut-être » comptent pour un.
	for mot := range strings.FieldsFuncSeq(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '’' && r != '-'
	}) {
		mot = strings.Trim(mot, "'’-")
		if mot == "" {
			continue
		}
		st.Mots++
		if m := strings.ToLower(mot); !motsVides[m] && utf8.RuneCountInString(m) > 2 {
			compte[m]++
		}
	}
	for _, r := range s {
		switch {
		case r == '.' || r == '!' || r == '?' || r == '…':
			if dansPhrase {
				st.Phrases++
				dansPhrase = false
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			dansPhrase = true
		}
	}
	if dansPhrase { // une dernière phrase sans ponctuation
		st.Phrases++
	}
	if st.Mots > 0 {
		st.MinutesLecture = max(1, (st.Mots+MotsParMinute-1)/MotsParMinute)
	}

	for mot, nb := range compte {
		st.Frequents = append(st.Frequents, Frequence{mot, nb})
	}
	slices.SortFunc(st.Frequents, func(a, b Frequence) int {
		return cmp.Or(cmp.Compare(b.Nombre, a.Nombre), strings.Compare(a.Mot, b.Mot))
	})
	if len(st.Frequents) > n {
		st.Frequents = st.Frequents[:n]
	}
	if st.Frequents == nil {
		st.Frequents = []Frequence{} // [] en JSON, pas null
	}
	return st
}
