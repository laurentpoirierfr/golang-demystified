package context

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAttends(t *testing.T) {
	if err := Attends(context.Background(), time.Millisecond); err != nil {
		t.Error(err)
	}
	ctx, annule := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer annule()
	debut := time.Now()
	if err := Attends(ctx, time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v", err)
	}
	if time.Since(debut) > time.Second {
		t.Error("l'attente aurait dû s'arrêter au délai")
	}
}

func TestUtilisateur(t *testing.T) {
	ctx := AvecUtilisateur(context.Background(), "ada")
	ctx = context.WithValue(ctx, "cleUtilisateur", "intrus") //nolint:staticcheck // une clé string ne doit pas entrer en collision
	if nom, ok := Utilisateur(ctx); !ok || nom != "ada" {
		t.Errorf("%q, %v", nom, ok)
	}
	if _, ok := Utilisateur(context.Background()); ok {
		t.Error("aucun utilisateur dans un contexte vide")
	}
}

func TestPremier(t *testing.T) {
	var annulees atomic.Int32
	lente := func(ctx context.Context) (string, error) {
		select {
		case <-time.After(time.Second):
			return "lente", nil
		case <-ctx.Done():
			annulees.Add(1)
			return "", ctx.Err()
		}
	}
	rapide := func(context.Context) (string, error) { return "rapide", nil }
	echec := func(context.Context) (string, error) { return "", errors.New("hors service") }

	v, err := Premier(context.Background(), lente, echec, rapide, lente)
	if err != nil || v != "rapide" {
		t.Errorf("%q, %v", v, err)
	}
	for i := 0; annulees.Load() < 2 && i < 100; i++ {
		time.Sleep(time.Millisecond)
	}
	if annulees.Load() != 2 {
		t.Errorf("%d sources lentes annulées, attendu 2", annulees.Load())
	}
	if _, err := Premier(context.Background(), echec, echec); err == nil || err.Error() != "hors service\nhors service" {
		t.Errorf("toutes en échec : %v", err)
	}
}
