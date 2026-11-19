// Package catalogue contient la logique du catalogue de livres. Il s'appuie
// sur les requêtes générées par sqlc (package db) et gère les transactions.
package catalogue

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/catalogue/internal/db"
)

// Erreurs métier.
var (
	ErrIntrouvable     = errors.New("livre introuvable")
	ErrPlusDExemplaire = errors.New("plus aucun exemplaire disponible")
	ErrISBNDejaUtilise = errors.New("ISBN déjà utilisé")
)

// Catalogue regroupe le pool de connexions et les requêtes générées.
type Catalogue struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// Nouveau crée le catalogue.
func Nouveau(pool *pgxpool.Pool) *Catalogue {
	return &Catalogue{pool: pool, q: db.New(pool)}
}

// NouveauLivre décrit un livre à ajouter.
type NouveauLivre struct {
	ISBN        string
	Titre       string
	Auteur      string
	Annee       *int32 // facultative
	Exemplaires int32
}

// Ajoute enregistre un livre et, si besoin, son auteur, dans une seule
// transaction : en cas d'erreur, rien n'est enregistré.
func (c *Catalogue) Ajoute(ctx context.Context, l NouveauLivre) (db.Livre, error) {
	var livre db.Livre
	err := c.transaction(ctx, func(q *db.Queries) error {
		auteur, err := q.AssureAuteur(ctx, l.Auteur)
		if err != nil {
			return fmt.Errorf("auteur %q : %w", l.Auteur, err)
		}
		livre, err = q.CreeLivre(ctx, db.CreeLivreParams{
			Isbn: l.ISBN, Titre: l.Titre, AuteurID: auteur.ID,
			Annee: l.Annee, Exemplaires: l.Exemplaires,
		})
		if violeUnicite(err) {
			return ErrISBNDejaUtilise
		}
		return err
	})
	return livre, err
}

// Emprunte retire un exemplaire du stock.
func (c *Catalogue) Emprunte(ctx context.Context, isbn string) error {
	n, err := c.q.PrendExemplaire(ctx, isbn)
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	// Aucune ligne modifiée : livre inconnu, ou stock épuisé ?
	if _, err := c.q.Livre(ctx, isbn); errors.Is(err, pgx.ErrNoRows) {
		return ErrIntrouvable
	} else if err != nil {
		return err
	}
	return ErrPlusDExemplaire
}

// Recherche trouve les livres dont le titre contient texte, publiés à partir
// de anneeMin si elle est donnée.
func (c *Catalogue) Recherche(ctx context.Context, texte string, anneeMin *int32) ([]db.Livre, error) {
	return c.q.RechercheLivres(ctx, db.RechercheLivresParams{Texte: texte, AnneeMin: anneeMin, Limite: 20})
}

// Requetes donne accès aux requêtes générées, pour les lectures simples.
func (c *Catalogue) Requetes() *db.Queries { return c.q }

// transaction exécute f avec des requêtes liées à une transaction, validée si
// f réussit et annulée sinon.
func (c *Catalogue) transaction(ctx context.Context, f func(*db.Queries) error) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // sans effet après un Commit réussi

	if err := f(c.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// violeUnicite indique si err est une violation de contrainte d'unicité
// (code PostgreSQL 23505).
func violeUnicite(err error) bool {
	e, ok := errors.AsType[*pgconn.PgError](err)
	return ok && e.Code == "23505"
}
