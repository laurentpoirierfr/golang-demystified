// Command taches démarre l'API des tâches.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/config"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run contient tout le programme. Ses dépendances (contexte, environnement,
// sortie) sont des paramètres : il se teste comme une fonction ordinaire.
// pret, s'il n'est pas nil, reçoit l'adresse d'écoute une fois le serveur
// démarré.
func run(ctx context.Context, getenv func(string) string, sortie io.Writer, pret chan<- string) error {
	cfg, err := config.Charge(getenv)
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(sortie, &slog.HandlerOptions{Level: cfg.NiveauJournal}))

	s, err := store.Ouvre(ctx, cfg.Base)
	if err != nil {
		return err
	}
	defer s.Close()

	handler, err := web.Nouveau(s, log).Routes()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	ecoute, err := net.Listen("tcp", cfg.Adresse)
	if err != nil {
		return err
	}
	log.Info("serveur démarré", "adresse", ecoute.Addr().String())
	if pret != nil {
		pret <- ecoute.Addr().String()
	}

	erreurs := make(chan error, 1)
	go func() { erreurs <- srv.Serve(ecoute) }()

	select {
	case err := <-erreurs: // le serveur s'est arrêté seul : c'est une erreur
		return err
	case <-ctx.Done(): // signal reçu : arrêt propre
	}

	log.Info("arrêt en cours", "delai", cfg.DelaiArret)
	arret, annule := context.WithTimeout(context.Background(), cfg.DelaiArret)
	defer annule()
	if err := srv.Shutdown(arret); err != nil { // attend les requêtes en cours
		return fmt.Errorf("arrêt : %w", err)
	}
	if err := <-erreurs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("serveur arrêté")
	return nil
}
