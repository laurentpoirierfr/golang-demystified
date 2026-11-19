// Package tache définit le domaine du service : la tâche, ses règles de
// validation et ses erreurs. Il ne dépend d'aucun autre package du service.
package tache

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Tache est une chose à faire.
type Tache struct {
	ID       int64     `json:"id"`
	Titre    string    `json:"titre"`
	Faite    bool      `json:"faite"`
	Creation time.Time `json:"creation"`
}

// ErrIntrouvable signale une tâche qui n'existe pas.
var ErrIntrouvable = errors.New("tâche introuvable")

// ErrValidation signale une donnée invalide.
type ErrValidation struct {
	Champ  string
	Raison string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("%s : %s", e.Champ, e.Raison)
}

// TitreMax est la longueur maximale d'un titre, en caractères.
const TitreMax = 200

// ValideTitre vérifie un titre et renvoie sa forme normalisée.
func ValideTitre(titre string) (string, error) {
	titre = strings.TrimSpace(titre)
	switch {
	case titre == "":
		return "", &ErrValidation{Champ: "titre", Raison: "obligatoire"}
	case utf8.RuneCountInString(titre) > TitreMax:
		return "", &ErrValidation{Champ: "titre", Raison: fmt.Sprintf("%d caractères au plus", TitreMax)}
	}
	return titre, nil
}
