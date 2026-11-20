package testing

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// Exercice 1 ★ : un test de table.
func TestSlug(t *testing.T) {
	tests := []struct {
		nom, titre, attendu string
	}{
		{"simple", "Bonjour", "bonjour"},
		{"espaces", "Go et Rust", "go-et-rust"},
		{"accents", "L'été à Paris !", "l-ete-a-paris"},
		{"ponctuation en bord", "  -- Go 1.27 -- ", "go-1-27"},
		{"vide", "", ""},
		{"rien d'utilisable", "?!", ""},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			if got := Slug(tt.titre); got != tt.attendu {
				t.Errorf("Slug(%q) = %q, attendu %q", tt.titre, got, tt.attendu)
			}
		})
	}
}

// Exercice 2 ★★ : un test de fuzzing. Une propriété, plutôt que des
// exemples : déchiffrer ce qu'on a chiffré redonne le texte. go test lance
// les exemples du corpus ; go test -fuzz=FuzzChiffre en génère d'autres.
func FuzzChiffre(f *testing.F) {
	f.Add("Bonjour, Gopher !", 3)
	f.Add("", 0)
	f.Add("zZ", -27)
	f.Fuzz(func(t *testing.T, texte string, decalage int) {
		if got := Chiffre(Chiffre(texte, decalage), -decalage); got != texte {
			t.Errorf("aller-retour de %q avec %d : %q", texte, decalage, got)
		}
	})
}

// Exercice 3 ★★★ : tester le temps avec synctest. Dans la bulle, l'horloge
// est fictive : les 7 secondes d'attente passent instantanément, et sont
// mesurées exactement.
func TestReessaie(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		appels := 0
		debut := time.Now()
		err := Reessaie(context.Background(), 4, time.Second, func() error {
			if appels++; appels < 4 {
				return errors.New("pas encore")
			}
			return nil
		})
		if err != nil || appels != 4 {
			t.Fatalf("%v après %d appels", err, appels)
		}
		if d := time.Since(debut); d != 7*time.Second { // 1 s + 2 s + 4 s
			t.Errorf("attente totale %v, attendu 7s", d)
		}
	})
}

func TestReessaieAnnule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, annule := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer annule()
		appels := 0
		err := Reessaie(ctx, 10, time.Second, func() error { appels++; return errors.New("non") })
		if !errors.Is(err, context.DeadlineExceeded) || appels != 2 {
			t.Errorf("%v après %d appels", err, appels)
		}
	})
}
