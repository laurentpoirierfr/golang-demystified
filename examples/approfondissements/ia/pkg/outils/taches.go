package outils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// Tache est une tâche de la liste.
type Tache struct {
	ID    int    `json:"id"`
	Titre string `json:"titre"`
	Faite bool   `json:"faite"`
}

// Taches est une liste de tâches en mémoire, sûre pour plusieurs goroutines.
type Taches struct {
	mu     sync.Mutex
	taches []Tache
}

// ErrIntrouvable signale une tâche inconnue.
var ErrIntrouvable = errors.New("tâche introuvable")

// Ajoute crée une tâche.
func (t *Taches) Ajoute(titre string) (Tache, error) {
	titre = strings.TrimSpace(titre)
	if titre == "" || utf8.RuneCountInString(titre) > 200 {
		return Tache{}, errors.New("le titre doit compter de 1 à 200 caractères")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	tache := Tache{ID: len(t.taches) + 1, Titre: titre}
	t.taches = append(t.taches, tache)
	return tache, nil
}

// Liste renvoie les tâches, terminées comprises si toutes est vrai.
func (t *Taches) Liste(toutes bool) []Tache {
	t.mu.Lock()
	defer t.mu.Unlock()
	var l []Tache
	for _, tache := range t.taches {
		if toutes || !tache.Faite {
			l = append(l, tache)
		}
	}
	return l
}

// Termine marque une tâche comme faite.
func (t *Taches) Termine(id int) (Tache, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.taches {
		if t.taches[i].ID == id {
			t.taches[i].Faite = true
			return t.taches[i], nil
		}
	}
	return Tache{}, fmt.Errorf("%w : %d", ErrIntrouvable, id)
}

// Entrées des outils : leur schéma JSON en est déduit.
type (
	entreeAjoute struct {
		Titre string `json:"titre" jsonschema:"le titre de la tâche, court et précis"`
	}
	entreeListe struct {
		Toutes bool `json:"toutes,omitempty" jsonschema:"vrai pour inclure les tâches terminées"`
	}
	entreeTermine struct {
		ID int `json:"id" jsonschema:"l'identifiant de la tâche, donné par lister_taches"`
	}
)

// Boite renvoie les outils de gestion des tâches.
func Boite(t *Taches) ([]Outil, error) {
	ajoute, err := Nouveau("ajouter_tache", "Ajoute une tâche à la liste et renvoie son identifiant.",
		func(_ context.Context, in entreeAjoute) (string, error) {
			tache, err := t.Ajoute(in.Titre)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tâche %d ajoutée : %s", tache.ID, tache.Titre), nil
		})
	if err != nil {
		return nil, err
	}
	liste, err := Nouveau("lister_taches", "Liste les tâches à faire (et les tâches terminées si demandé).",
		func(_ context.Context, in entreeListe) (string, error) {
			taches := t.Liste(in.Toutes)
			if len(taches) == 0 {
				return "aucune tâche", nil
			}
			var b strings.Builder
			for _, tache := range taches {
				etat := "à faire"
				if tache.Faite {
					etat = "faite"
				}
				fmt.Fprintf(&b, "%d. %s (%s)\n", tache.ID, tache.Titre, etat)
			}
			return b.String(), nil
		})
	if err != nil {
		return nil, err
	}
	termine, err := Nouveau("terminer_tache", "Marque une tâche comme faite, d'après son identifiant.",
		func(_ context.Context, in entreeTermine) (string, error) {
			tache, err := t.Termine(in.ID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tâche %d terminée : %s", tache.ID, tache.Titre), nil
		})
	if err != nil {
		return nil, err
	}
	return []Outil{ajoute, liste, termine}, nil
}
