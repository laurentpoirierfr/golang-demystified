// Package tests illustre le chapitre « Tests : testing ».
package tests

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
)

// Slug transforme un titre en identifiant d'URL : « Go 1.27 ! » → « go-1-27 ».
func Slug(titre string) string {
	var b strings.Builder
	tiret := false
	for _, r := range strings.ToLower(titre) {
		switch {
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

// ErrDelai est renvoyée lorsque l'attente dépasse le délai.
var ErrDelai = errors.New("délai dépassé")

// AttendAvecDelai attend la fin de travail, au plus pendant delai.
func AttendAvecDelai(ctx context.Context, travail <-chan struct{}, delai time.Duration) error {
	select {
	case <-travail:
		return nil
	case <-time.After(delai):
		return ErrDelai
	case <-ctx.Done():
		return ctx.Err()
	}
}
