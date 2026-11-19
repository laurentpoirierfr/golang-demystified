package store

import (
	"errors"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

// nouveauStore ouvre une base en mémoire, propre à chaque test.
func nouveauStore(t *testing.T) *SQLite {
	t.Helper()
	s, err := Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	s.maintenant = func() time.Time { return time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC) }
	return s
}

func TestCycleDeVie(t *testing.T) {
	s := nouveauStore(t)
	ctx := t.Context()

	creee, err := s.Cree(ctx, "Écrire le chapitre")
	if err != nil {
		t.Fatal(err)
	}
	if creee.ID != 1 || creee.Creation.Hour() != 10 {
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
}

func TestListeVide(t *testing.T) {
	taches, err := nouveauStore(t).Liste(t.Context())
	if err != nil || taches == nil || len(taches) != 0 {
		t.Errorf("Liste = %#v, %v ; attendu une liste vide non nil", taches, err)
	}
}

func TestMigrationsIdempotentes(t *testing.T) {
	s := nouveauStore(t)
	if err := s.migre(t.Context()); err != nil {
		t.Fatalf("seconde application des migrations : %v", err)
	}
}
