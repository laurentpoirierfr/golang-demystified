// Package inscription inscrit des membres. Il illustre l'injection de
// dépendances à la manière de Go : le service déclare les petites interfaces
// dont il a besoin, et reçoit leurs implémentations à la construction. Pas de
// conteneur, pas de variable globale : le programme principal assemble le
// tout, et les tests y glissent leurs doublures.
package inscription

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Membre est un membre inscrit.
type Membre struct {
	Courriel string
	Nom      string
	Le       time.Time
}

// Depot stocke les membres. L'interface est déclarée ici, par le code qui
// l'utilise, et ne contient que ce dont il a besoin : deux méthodes, et non
// les vingt d'un dépôt complet.
type Depot interface {
	Existe(ctx context.Context, courriel string) (bool, error)
	Enregistre(ctx context.Context, m Membre) error
}

// Courrier envoie un message.
type Courrier interface {
	Envoie(ctx context.Context, a, sujet, corps string) error
}

// ErrDejaInscrit signale un courriel déjà utilisé.
var ErrDejaInscrit = errors.New("déjà inscrit")

// Service inscrit des membres. Ses dépendances sont des champs privés,
// fixées une fois pour toutes par Nouveau.
type Service struct {
	depot      Depot
	courrier   Courrier
	maintenant func() time.Time
}

// Nouveau assemble le service. maintenant est l'horloge : time.Now en
// production, une horloge fixe dans les tests.
func Nouveau(depot Depot, courrier Courrier, maintenant func() time.Time) *Service {
	return &Service{depot: depot, courrier: courrier, maintenant: maintenant}
}

// Inscrit enregistre un membre, puis lui souhaite la bienvenue.
func (s *Service) Inscrit(ctx context.Context, courriel, nom string) (Membre, error) {
	adresse, err := mail.ParseAddress(courriel)
	if err != nil {
		return Membre{}, fmt.Errorf("courriel invalide : %w", err)
	}
	courriel = strings.ToLower(adresse.Address)
	nom = strings.TrimSpace(nom)
	if nom == "" {
		return Membre{}, errors.New("nom obligatoire")
	}

	existe, err := s.depot.Existe(ctx, courriel)
	if err != nil {
		return Membre{}, fmt.Errorf("inscription de %s : %w", courriel, err)
	}
	if existe {
		return Membre{}, fmt.Errorf("%s : %w", courriel, ErrDejaInscrit)
	}
	m := Membre{Courriel: courriel, Nom: nom, Le: s.maintenant()}
	if err := s.depot.Enregistre(ctx, m); err != nil {
		return Membre{}, fmt.Errorf("inscription de %s : %w", courriel, err)
	}
	// L'inscription est faite : l'échec du courrier ne l'annule pas.
	if err := s.courrier.Envoie(ctx, courriel, "Bienvenue", "Bonjour "+nom+","); err != nil {
		return m, fmt.Errorf("membre inscrit, mais courriel non envoyé : %w", err)
	}
	return m, nil
}
