// Package store stocke les tâches avec database/sql, dans SQLite ou
// PostgreSQL selon la chaîne de connexion.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pilote PostgreSQL, enregistré sous le nom « pgx »
	_ "modernc.org/sqlite"             // pilote SQLite en Go pur, enregistré sous le nom « sqlite »

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

//go:embed migrations
var migrations embed.FS

// dialecte décrit ce qui diffère d'une base à l'autre.
type dialecte struct {
	pilote     string // nom du pilote database/sql
	migrations string // répertoire des migrations
	dollars    bool   // paramètres $1, $2… au lieu de ?
}

var (
	sqlite   = dialecte{pilote: "sqlite", migrations: "migrations/sqlite"}
	postgres = dialecte{pilote: "pgx", migrations: "migrations/postgres", dollars: true}
)

// Store implémente le stockage des tâches.
type Store struct {
	db         *sql.DB
	d          dialecte
	maintenant func() time.Time // remplaçable dans les tests
}

// Ouvre ouvre la base, vérifie la connexion et applique les migrations.
// dsn vaut par exemple "file:taches.db", ":memory:" (SQLite) ou
// "postgres://utilisateur:motdepasse@hote:5432/base" (PostgreSQL).
func Ouvre(ctx context.Context, dsn string) (*Store, error) {
	d := sqlite
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		d = postgres
	}
	db, err := sql.Open(d.pilote, dsn)
	if err != nil {
		return nil, err
	}
	if d == sqlite {
		// SQLite n'accepte qu'un écrivain à la fois, et une base « :memory: »
		// n'existe que dans sa connexion : une seule connexion suffit.
		db.SetMaxOpenConns(1)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connexion à la base %s : %w", d.pilote, err)
	}
	s := &Store{db: db, d: d, maintenant: time.Now}
	if err := s.migre(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations : %w", err)
	}
	return s, nil
}

// Close ferme la base.
func (s *Store) Close() error { return s.db.Close() }

// Ping vérifie que la base répond, pour le contrôle de santé.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// q adapte une requête écrite avec des « ? » au dialecte de la base :
// PostgreSQL attend $1, $2…
func (s *Store) q(requete string) string {
	if !s.d.dollars {
		return requete
	}
	var b strings.Builder
	n := 0
	for _, r := range requete {
		if r == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// migre applique, dans l'ordre et une seule fois, les fichiers de migrations.
func (s *Store) migre(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS migrations (nom TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	fichiers, err := fs.Glob(migrations, s.d.migrations+"/*.sql") // triés par nom
	if err != nil {
		return err
	}
	for _, f := range fichiers {
		var n int
		if err := s.db.QueryRowContext(ctx,
			s.q(`SELECT COUNT(*) FROM migrations WHERE nom = ?`), f).Scan(&n); err != nil {
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
			_, err := tx.ExecContext(ctx, s.q(`INSERT INTO migrations (nom) VALUES (?)`), f)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// transaction exécute f dans une transaction, validée si f réussit.
func (s *Store) transaction(ctx context.Context, f func(*sql.Tx) error) error {
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
func (s *Store) Liste(ctx context.Context) ([]tache.Tache, error) {
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
func (s *Store) Trouve(ctx context.Context, id int64) (tache.Tache, error) {
	ligne := s.db.QueryRowContext(ctx,
		s.q(`SELECT id, titre, faite, creation FROM taches WHERE id = ?`), id)
	t, err := scanne(ligne)
	if errors.Is(err, sql.ErrNoRows) {
		return tache.Tache{}, tache.ErrIntrouvable
	}
	return t, err
}

// Cree enregistre une nouvelle tâche.
func (s *Store) Cree(ctx context.Context, titre string) (tache.Tache, error) {
	t := tache.Tache{Titre: titre, Creation: s.maintenant().UTC().Truncate(time.Millisecond)}
	err := s.db.QueryRowContext(ctx,
		s.q(`INSERT INTO taches (titre, creation) VALUES (?, ?) RETURNING id`),
		t.Titre, t.Creation.UnixMilli()).Scan(&t.ID)
	return t, err
}

// MarqueFaite change l'état d'une tâche et renvoie la tâche à jour.
func (s *Store) MarqueFaite(ctx context.Context, id int64, faite bool) (tache.Tache, error) {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE taches SET faite = ? WHERE id = ?`), faite, id)
	if err := verifieUneLigne(res, err); err != nil {
		return tache.Tache{}, err
	}
	return s.Trouve(ctx, id)
}

// Supprime supprime une tâche.
func (s *Store) Supprime(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM taches WHERE id = ?`), id)
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
