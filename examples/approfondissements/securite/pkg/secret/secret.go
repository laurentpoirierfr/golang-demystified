// Package secret protège les valeurs sensibles (mots de passe, clés d'API,
// jetons) contre les fuites accidentelles : dans les journaux, les messages
// d'erreur, les réponses JSON ou une trace de débogage.
package secret

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const masque = "[masqué]"

// Secret contient une valeur sensible. Toutes ses représentations textuelles
// sont masquées : seule la méthode Valeur, appelée explicitement, la révèle.
type Secret struct {
	valeur string
}

// Nouveau enveloppe une valeur sensible.
func Nouveau(valeur string) Secret { return Secret{valeur: valeur} }

// Valeur révèle le secret. Son nom, explicite, se repère à la relecture.
func (s Secret) Valeur() string { return s.valeur }

// Vide indique si le secret est absent.
func (s Secret) Vide() bool { return s.valeur == "" }

// Egal compare en temps constant : la durée de la comparaison ne révèle pas
// combien de caractères sont corrects.
func (s Secret) Egal(autre string) bool {
	return subtle.ConstantTimeCompare([]byte(s.valeur), []byte(autre)) == 1
}

// String masque le secret pour fmt (%v, %s, %q) et les erreurs.
func (s Secret) String() string { return masque }

// GoString masque le secret pour %#v.
func (s Secret) GoString() string { return masque }

// LogValue masque le secret pour log/slog.
func (s Secret) LogValue() slog.Value { return slog.StringValue(masque) }

// MarshalText masque le secret pour encoding/json, encoding/xml, etc.
func (s Secret) MarshalText() ([]byte, error) { return []byte(masque), nil }

// Lit lit le secret nom depuis l'environnement. Si la variable nom_FILE est
// définie, le secret est lu dans ce fichier : c'est la convention des secrets
// Docker et Kubernetes, montés comme fichiers, qui ne figurent alors ni dans
// l'environnement du processus ni dans `docker inspect`.
func Lit(getenv func(string) string, nom string) (Secret, error) {
	if fichier := getenv(nom + "_FILE"); fichier != "" {
		contenu, err := os.ReadFile(fichier)
		if err != nil {
			// L'erreur cite le fichier, jamais son contenu.
			return Secret{}, fmt.Errorf("secret %s : %w", nom, err)
		}
		return Nouveau(strings.TrimRight(string(contenu), "\r\n")), nil
	}
	return Nouveau(getenv(nom)), nil
}
