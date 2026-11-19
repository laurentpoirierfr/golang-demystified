package db

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pilote PostgreSQL (pgx v5), schéma « pgx5:// »
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Les migrations sont embarquées dans l'exécutable : le programme migre sa
// propre base au démarrage, sans fichier à déployer à côté.
//
//go:embed migrations/*.sql
var migrations embed.FS

// NouveauMigrateur prépare golang-migrate pour la base dsn, de la forme
// postgres://utilisateur:motdepasse@hote:5432/base.
func NouveauMigrateur(dsn string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	// Le pilote pgx v5 de golang-migrate s'enregistre sous le schéma « pgx5 ».
	url := "pgx5://" + strings.TrimPrefix(strings.TrimPrefix(dsn, "postgres://"), "postgresql://")
	return migrate.NewWithSourceInstance("iofs", source, url)
}

// Migre applique les migrations qui ne l'ont pas encore été, et renvoie la
// version du schéma obtenue.
func Migre(dsn string) (uint, error) {
	m, err := NouveauMigrateur(dsn)
	if err != nil {
		return 0, err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, fmt.Errorf("migrations : %w", err)
	}
	version, sale, err := m.Version()
	if err != nil {
		return 0, err
	}
	if sale {
		// Une migration a échoué à mi-chemin : une intervention manuelle est
		// nécessaire (corriger, puis « migrate force »).
		return version, fmt.Errorf("schéma dans un état incohérent à la version %d", version)
	}
	return version, nil
}
