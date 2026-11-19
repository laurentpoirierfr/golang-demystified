package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf8"
)

func TestSlug(t *testing.T) {
	cas := []struct {
		nom     string
		entree  string
		attendu string
	}{
		{"simple", "Bonjour", "bonjour"},
		{"espaces", "Le gopher creuse", "le-gopher-creuse"},
		{"ponctuation", "Go 1.27 !", "go-1-27"},
		{"accents ignorés", "Été à Paris", "t-paris"},
		{"vide", "", ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			t.Parallel()
			if got := Slug(c.entree); got != c.attendu {
				t.Errorf("Slug(%q) = %q, attendu %q", c.entree, got, c.attendu)
			}
		})
	}
}

func FuzzSlug(f *testing.F) {
	f.Add("Go 1.27 !")
	f.Add("Été à Paris")
	f.Fuzz(func(t *testing.T, titre string) {
		s := Slug(titre)
		if !utf8.ValidString(s) {
			t.Fatalf("slug invalide : %q", s)
		}
		if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
			t.Errorf("Slug(%q) = %q : tiret mal placé", titre, s)
		}
	})
}

func BenchmarkSlug(b *testing.B) {
	titre := strings.Repeat("Le gopher creuse ", 10)
	for b.Loop() {
		Slug(titre)
	}
}

func TestAttendAvecDelai(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		travail := make(chan struct{})
		debut := time.Now()

		err := AttendAvecDelai(t.Context(), travail, time.Hour)

		if !errors.Is(err, ErrDelai) {
			t.Fatalf("erreur = %v, attendu ErrDelai", err)
		}
		if ecoule := time.Since(debut); ecoule != time.Hour {
			t.Errorf("écoulé = %v, attendu 1h", ecoule)
		}
	})
}

func TestAide(t *testing.T) {
	verifie := func(t *testing.T, entree, attendu string) {
		t.Helper() // l'erreur pointe vers l'appelant
		if got := Slug(entree); got != attendu {
			t.Errorf("Slug(%q) = %q, attendu %q", entree, got, attendu)
		}
	}
	verifie(t, "A B", "a-b")
	verifie(t, "  A  ", "a")
}

func ExampleSlug() {
	fmt.Println(Slug("Golang Demystified : le livre"))
	// Output: golang-demystified-le-livre
}
