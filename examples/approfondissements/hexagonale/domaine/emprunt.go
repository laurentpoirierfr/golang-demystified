// Package domaine contient le cœur métier de la bibliothèque : les règles
// d'emprunt. Il n'importe que la bibliothèque standard, et aucun autre
// package du projet.
package domaine

import (
	"errors"
	"fmt"
	"time"
)

// Règles de la bibliothèque.
const (
	EmpruntsMax      = 3
	DureeEmprunt     = 21 * 24 * time.Hour
	PenaliteParJour  = 20 // en centimes
	PenaliteMaximale = 500
)

// Erreurs métier.
var (
	ErrLivreIndisponible = errors.New("livre déjà emprunté")
	ErrQuotaAtteint      = fmt.Errorf("%d emprunts au plus", EmpruntsMax)
	ErrMembreSuspendu    = errors.New("membre suspendu : pénalités impayées")
	ErrDejaRendu         = errors.New("emprunt déjà rendu")
	ErrIntrouvable       = errors.New("introuvable")
)

// Livre est un exemplaire du catalogue.
type Livre struct {
	ISBN  string
	Titre string
}

// Membre est un adhérent de la bibliothèque.
type Membre struct {
	ID        string
	Nom       string
	Email     string
	Penalites int // en centimes, à régler avant tout nouvel emprunt
}

// Emprunt associe un livre à un membre pendant une période.
type Emprunt struct {
	ID       string
	ISBN     string
	Membre   string
	Debut    time.Time
	Echeance time.Time
	Retour   time.Time // zéro tant que le livre n'est pas rendu
}

// EnCours indique si le livre n'a pas encore été rendu.
func (e Emprunt) EnCours() bool { return e.Retour.IsZero() }

// EnRetard indique si l'emprunt dépasse son échéance à l'instant t.
func (e Emprunt) EnRetard(t time.Time) bool { return e.EnCours() && t.After(e.Echeance) }

// PeutEmprunter applique les règles d'emprunt : un membre sans pénalité,
// sous son quota, demandant un livre disponible.
func PeutEmprunter(m Membre, empruntsEnCours int, livreDisponible bool) error {
	switch {
	case m.Penalites > 0:
		return ErrMembreSuspendu
	case empruntsEnCours >= EmpruntsMax:
		return ErrQuotaAtteint
	case !livreDisponible:
		return ErrLivreIndisponible
	}
	return nil
}

// NouvelEmprunt crée un emprunt qui commence à l'instant debut.
func NouvelEmprunt(id, isbn, membre string, debut time.Time) Emprunt {
	return Emprunt{ID: id, ISBN: isbn, Membre: membre, Debut: debut, Echeance: debut.Add(DureeEmprunt)}
}

// Rend clôt l'emprunt à l'instant t et renvoie la pénalité due, en centimes :
// 20 centimes par jour de retard commencé, 5 euros au plus.
func (e *Emprunt) Rend(t time.Time) (int, error) {
	if !e.EnCours() {
		return 0, ErrDejaRendu
	}
	e.Retour = t
	if !t.After(e.Echeance) {
		return 0, nil
	}
	jours := int((t.Sub(e.Echeance) + 24*time.Hour - 1) / (24 * time.Hour))
	return min(jours*PenaliteParJour, PenaliteMaximale), nil
}
