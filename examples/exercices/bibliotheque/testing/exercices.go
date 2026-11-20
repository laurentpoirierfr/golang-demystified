// Package testing contient le code à tester des exercices du chapitre
// « Tests : testing ». Ici, l'exercice inverse les rôles : le code est donné,
// et ce sont les tests (exercices_test.go) qui sont la solution.
package testing

import (
	"context"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Slug transforme un titre en identifiant d'URL : « L'été à Paris ! » →
// « l-ete-a-paris ».
func Slug(titre string) string {
	var b strings.Builder
	tiret := false
	for _, r := range norm.NFD.String(strings.ToLower(titre)) {
		switch {
		case unicode.Is(unicode.Mn, r): // accent séparé de sa lettre par NFD
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			if tiret && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			tiret = false
		default:
			tiret = true
		}
	}
	return b.String()
}

// Chiffre applique le chiffre de César : chaque lettre ASCII avance de
// decalage places dans l'alphabet. Les autres caractères ne changent pas.
func Chiffre(texte string, decalage int) string {
	// Octet par octet, et non rune par rune : un texte qui n'est pas de l'UTF-8
	// valide traverse la fonction intact (le fuzzing l'a montré).
	d := byte((decalage%26 + 26) % 26)
	b := []byte(texte)
	for i, c := range b {
		switch {
		case c >= 'a' && c <= 'z':
			b[i] = 'a' + (c-'a'+d)%26
		case c >= 'A' && c <= 'Z':
			b[i] = 'A' + (c-'A'+d)%26
		}
	}
	return string(b)
}

// Reessaie appelle f jusqu'à ce qu'elle réussisse, au plus essais fois, en
// doublant l'attente entre deux essais (attente, 2×attente, 4×attente…).
func Reessaie(ctx context.Context, essais int, attente time.Duration, f func() error) error {
	var err error
	for i := range essais {
		if err = f(); err == nil {
			return nil
		}
		if i == essais-1 {
			break
		}
		select {
		case <-time.After(attente):
			attente *= 2
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
