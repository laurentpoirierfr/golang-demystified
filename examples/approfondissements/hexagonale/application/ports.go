// Package application orchestre les cas d'usage de la bibliothèque. Il
// définit les ports : les interfaces dont il a besoin pour accéder au monde
// extérieur, implémentées par les adaptateurs.
package application

import (
	"context"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/domaine"
)

// Catalogue donne accès aux livres et aux membres.
type Catalogue interface {
	Livre(ctx context.Context, isbn string) (domaine.Livre, error)
	Membre(ctx context.Context, id string) (domaine.Membre, error)
	EnregistreMembre(ctx context.Context, m domaine.Membre) error
}

// Emprunts stocke les emprunts.
type Emprunts interface {
	Emprunt(ctx context.Context, id string) (domaine.Emprunt, error)
	EnCoursPourLivre(ctx context.Context, isbn string) (bool, error)
	EnCoursPourMembre(ctx context.Context, membre string) ([]domaine.Emprunt, error)
	EnCours(ctx context.Context) ([]domaine.Emprunt, error)
	Enregistre(ctx context.Context, e domaine.Emprunt) error
}

// Notifieur prévient les membres.
type Notifieur interface {
	Notifie(ctx context.Context, m domaine.Membre, message string) error
}

// Horloge donne l'heure : la remplacer rend le temps contrôlable en test.
type Horloge interface {
	Maintenant() time.Time
}

// Identifiants produit des identifiants uniques.
type Identifiants interface {
	Nouveau() string
}

// Transactions exécute une fonction de manière atomique : sur une base de
// données, dans une transaction.
type Transactions interface {
	Atomique(ctx context.Context, f func(ctx context.Context) error) error
}
