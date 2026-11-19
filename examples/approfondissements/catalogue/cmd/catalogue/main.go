// Command catalogue migre la base, puis affiche les statistiques du catalogue.
//
//	CATALOGUE_BASE=postgres://gopher:secret@localhost:5432/catalogue go run ./cmd/catalogue
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/catalogue/internal/catalogue"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/catalogue/internal/db"
)

func main() {
	if err := run(context.Background(), os.Getenv("CATALOGUE_BASE")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dsn string) error {
	if dsn == "" {
		return fmt.Errorf("CATALOGUE_BASE n'est pas définie")
	}

	// 1. Le programme met son schéma à jour lui-même, au démarrage.
	version, err := db.Migre(dsn)
	if err != nil {
		return err
	}
	slog.Info("schéma à jour", "version", version)

	// 2. Un pool de connexions pgx, partagé par tout le programme.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	stats, err := catalogue.Nouveau(pool).Requetes().StatistiquesParAuteur(ctx)
	if err != nil {
		return err
	}
	for _, s := range stats {
		fmt.Printf("%-25s %3d livre(s) %4d exemplaire(s)\n", s.Nom, s.Livres, s.Exemplaires)
	}
	return nil
}
