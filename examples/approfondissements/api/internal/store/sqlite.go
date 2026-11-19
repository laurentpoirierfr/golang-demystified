// Package store stocke les tâches dans une base SQLite, avec database/sql.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	_ "modernc.org/sqlite" // pilote SQLite en Go pur, enregistré sous le nom « sqlite »

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

//go:embed migrations/*.sql
var migrations embed.FS

// SQLite implémente le stockage des tâches.
type SQLite struct {
	db         *sql.DB
	maintenant func() time.Time // remplaçable dans les tests
}

// Ouvre ouvre la base, vérifie la connexion et applique les migrations.
// dsn vaut par exemple "file:taches.db" ou ":memory:".
func Ouvre(ctx context.Context, dsn string) (*SQLite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite n'accepte qu'un écrivain à la fois, et une base « :memory: »
	// n'existe que dans sa connexion : une seule connexion suffit.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connexion à %s : %w", dsn, err)
	}
	s := &SQLite{db: db, maintenant: time.Now}
	if err := s.migre(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations : %w", err)
	}
	return s, nil
}

// Close ferme la base.
func (s *SQLite) Close() error { return s.db.Close() }

// Ping vérifie que la base répond, pour le contrôle de santé.
func (s *SQLite) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// migre applique, dans l'ordre et une seule fois, les fichiers de migrations.
func (s *SQLite) migre(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS migrations (nom TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	fichiers, err := fs.Glob(migrations, "migrations/*.sql") // triés par nom
	if err != nil {
		return err
	}
	for _, f := range fichiers {
		var n int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM migrations WHERE nom = ?`, f).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue // déjà appliquée
		}
		requete, err := migrations.ReadFile(f)
		if err != nil {
			return err
		}
		if err := s.transaction(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(requete)); err != nil {
				return fmt.Errorf("%s : %w", f, err)
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO migrations (nom) VALUES (?)`, f)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// transaction exécute f dans une transaction, validée si f réussit.
func (s *SQLite) transaction(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Liste renvoie toutes les tâches, par ordre de création.
func (s *SQLite) Liste(ctx context.Context) ([]tache.Tache, error) {
	lignes, err := s.db.QueryContext(ctx,
		`SELECT id, titre, faite, creation FROM taches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer lignes.Close()

	taches := []tache.Tache{} // une liste vide s'encode en [], pas en null
	for lignes.Next() {
		t, err := scanne(lignes)
		if err != nil {
			return nil, err
		}
		taches = append(taches, t)
	}
	return taches, lignes.Err() // erreur éventuelle pendant le parcours
}

// Trouve renvoie une tâche, ou tache.ErrIntrouvable.
func (s *SQLite) Trouve(ctx context.Context, id int64) (tache.Tache, error) {
	ligne := s.db.QueryRowContext(ctx,
		`SELECT id, titre, faite, creation FROM taches WHERE id = ?`, id)
	t, err := scanne(ligne)
	if errors.Is(err, sql.ErrNoRows) {
		return tache.Tache{}, tache.ErrIntrouvable
	}
	return t, err
}

// Cree enregistre une nouvelle tâche.
func (s *SQLite) Cree(ctx context.Context, titre string) (tache.Tache, error) {
	t := tache.Tache{Titre: titre, Creation: s.maintenant().UTC().Truncate(time.Millisecond)}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO taches (titre, creation) VALUES (?, ?) RETURNING id`,
		t.Titre, t.Creation.UnixMilli()).Scan(&t.ID)
	return t, err
}

// MarqueFaite change l'état d'une tâche et renvoie la tâche à jour.
func (s *SQLite) MarqueFaite(ctx context.Context, id int64, faite bool) (tache.Tache, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE taches SET faite = ? WHERE id = ?`, faite, id)
	if err := verifieUneLigne(res, err); err != nil {
		return tache.Tache{}, err
	}
	return s.Trouve(ctx, id)
}

// Supprime supprime une tâche.
func (s *SQLite) Supprime(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM taches WHERE id = ?`, id)
	return verifieUneLigne(res, err)
}

func verifieUneLigne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return tache.ErrIntrouvable
	}
	return nil
}

// scanneur est satisfait par *sql.Row et *sql.Rows.
type scanneur interface{ Scan(dest ...any) error }

func scanne(s scanneur) (tache.Tache, error) {
	var t tache.Tache
	var creation int64
	if err := s.Scan(&t.ID, &t.Titre, &t.Faite, &creation); err != nil {
		return tache.Tache{}, err
	}
	t.Creation = time.UnixMilli(creation).UTC()
	return t, nil
}
