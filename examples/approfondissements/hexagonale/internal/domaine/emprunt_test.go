package domaine

import (
	"errors"
	"testing"
	"time"
)

func TestPeutEmprunter(t *testing.T) {
	cas := []struct {
		nom        string
		membre     Membre
		enCours    int
		disponible bool
		attendu    error
	}{
		{"autorisé", Membre{}, 2, true, nil},
		{"pénalités", Membre{Penalites: 20}, 0, true, ErrMembreSuspendu},
		{"quota", Membre{}, 3, true, ErrQuotaAtteint},
		{"indisponible", Membre{}, 0, false, ErrLivreIndisponible},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if err := PeutEmprunter(c.membre, c.enCours, c.disponible); !errors.Is(err, c.attendu) {
				t.Errorf("erreur = %v, attendu %v", err, c.attendu)
			}
		})
	}
}

func TestPenalites(t *testing.T) {
	debut := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	cas := []struct {
		nom     string
		retour  time.Duration // après l'échéance
		attendu int
	}{
		{"en avance", -24 * time.Hour, 0},
		{"pile à l'échéance", 0, 0},
		{"une heure de retard", time.Hour, 20},
		{"trois jours", 3 * 24 * time.Hour, 60},
		{"plafond", 100 * 24 * time.Hour, PenaliteMaximale},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			e := NouvelEmprunt("e1", "isbn", "m1", debut)
			penalite, err := e.Rend(e.Echeance.Add(c.retour))
			if err != nil || penalite != c.attendu {
				t.Errorf("Rend = %d, %v ; attendu %d", penalite, err, c.attendu)
			}
		})
	}
}

func TestDoubleRetour(t *testing.T) {
	e := NouvelEmprunt("e1", "isbn", "m1", time.Now())
	e.Rend(time.Now())
	if _, err := e.Rend(time.Now()); !errors.Is(err, ErrDejaRendu) {
		t.Errorf("erreur = %v, attendu ErrDejaRendu", err)
	}
}
