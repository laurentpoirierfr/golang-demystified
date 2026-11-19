package store

import (
	"errors"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

// ouvre ouvre une base pour un test, et la ferme à la fin du test.
func ouvre(t *testing.T, dsn string) *Store {
	t.Helper()
	s, err := Ouvre(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	s.maintenant = func() time.Time { return time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC) }
	return s
}

// suite regroupe les tests de comportement du stockage. Elle s'exécute à
// l'identique sur chaque base : SQLite ici, PostgreSQL dans postgres_test.go.
func suite(t *testing.T, s *Store) {
	t.Run("liste vide", func(t *testing.T) {
		taches, err := s.Liste(t.Context())
		if err != nil || taches == nil || len(taches) != 0 {
			t.Errorf("Liste = %#v, %v ; attendu une liste vide non nil", taches, err)
		}
	})

	t.Run("cycle de vie", func(t *testing.T) {
		ctx := t.Context()
		creee, err := s.Cree(ctx, "Écrire le chapitre")
		if err != nil {
			t.Fatal(err)
		}
		if creee.ID == 0 || creee.Creation.Hour() != 10 {
			t.Errorf("tâche créée inattendue : %+v", creee)
		}

		faite, err := s.MarqueFaite(ctx, creee.ID, true)
		if err != nil || !faite.Faite {
			t.Fatalf("MarqueFaite = %+v, %v", faite, err)
		}

		taches, err := s.Liste(ctx)
		if err != nil || len(taches) != 1 || taches[0] != faite {
			t.Fatalf("Liste = %+v, %v", taches, err)
		}

		if err := s.Supprime(ctx, creee.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Trouve(ctx, creee.ID); !errors.Is(err, tache.ErrIntrouvable) {
			t.Errorf("Trouve après suppression : %v, attendu ErrIntrouvable", err)
		}
		if err := s.Supprime(ctx, creee.ID); !errors.Is(err, tache.ErrIntrouvable) {
			t.Errorf("double suppression : %v, attendu ErrIntrouvable", err)
		}
	})

	t.Run("migrations idempotentes", func(t *testing.T) {
		if err := s.migre(t.Context()); err != nil {
			t.Fatalf("seconde application des migrations : %v", err)
		}
	})
}

func TestSQLite(t *testing.T) {
	suite(t, ouvre(t, ":memory:"))
}

func TestParametres(t *testing.T) {
	pg := &Store{d: postgres}
	if got := pg.q(`UPDATE t SET a = ? WHERE id = ?`); got != `UPDATE t SET a = $1 WHERE id = $2` {
		t.Errorf("q = %q", got)
	}
	lite := &Store{d: sqlite}
	if got := lite.q(`WHERE id = ?`); got != `WHERE id = ?` {
		t.Errorf("q = %q", got)
	}
}
