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

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/config"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/telemetrie"
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
	// Journal JSON, enrichi des identifiants de trace (trace_id, span_id).
	log := slog.New(telemetrie.Journal{Handler: slog.NewJSONHandler(sortie, &slog.HandlerOptions{Level: cfg.NiveauJournal})})

	// Observabilité : traces envoyées en OTLP si un collecteur est configuré,
	// métriques exposées sur /metrics.
	options := telemetrie.Options{Service: "api-taches", Version: "1.0.0"}
	if getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		// L'exportateur lit lui-même les variables standard d'OpenTelemetry :
		// OTEL_EXPORTER_OTLP_ENDPOINT (adresse de base, complétée par /v1/traces),
		// OTEL_EXPORTER_OTLP_HEADERS (authentification), etc.
		exportateur, err := otlptracehttp.New(ctx)
		if err != nil {
			return fmt.Errorf("traces : %w", err)
		}
		options.Exportateur = exportateur
	}
	tel, err := telemetrie.Demarre(options)
	if err != nil {
		return err
	}
	defer func() {
		// Envoie les dernières traces avant de quitter.
		arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
		defer annule()
		if err := tel.Arrete(arret); err != nil {
			log.Warn("arrêt de la télémétrie", "erreur", err)
		}
	}()

	s, err := store.Ouvre(ctx, cfg.Base)
	if err != nil {
		return err
	}
	defer s.Close()
	s.Instrumente(tel.Traces)

	serveur := web.Nouveau(s, log)
	if err := serveur.Instrumente(tel.Metriques); err != nil {
		return err
	}
	api, err := serveur.Routes()
	if err != nil {
		return err
	}

	// /metrics, et un span et des métriques pour chaque requête de l'API.
	racine := tel.Handler("api-taches", api)

	srv := &http.Server{
		Handler:           racine,
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
