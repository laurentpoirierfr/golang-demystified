// Command serveur démarre les salons de discussion en temps réel.
//
//	SALONS_ADRESSE    adresse d'écoute (localhost:8080)
//	SALONS_ORIGINES   origines autorisées pour les WebSockets, séparées par
//	                  des virgules (en plus de l'hôte du serveur)
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
	"strings"
	"syscall"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/salon"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil, 0); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run contient tout le programme. pouls règle l'intervalle des signaux de vie
// (0 : valeur par défaut) ; les tests le raccourcissent.
func run(ctx context.Context, getenv func(string) string, sortie io.Writer, pret chan<- string, pouls time.Duration) error {
	log := slog.New(slog.NewJSONHandler(sortie, nil))
	adresse := getenv("SALONS_ADRESSE")
	if adresse == "" {
		adresse = "localhost:8080"
	}
	var origines []string
	if o := getenv("SALONS_ORIGINES"); o != "" {
		origines = strings.Split(o, ",")
	}

	serveur := web.Nouveau(salon.Nouveau(), log, web.Options{Origines: origines, Pouls: pouls})
	srv := &http.Server{
		Handler:           serveur.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		// Un WriteTimeout couperait les flux SSE au bout du délai : le flux le
		// lève pour sa propre réponse (ResponseController.SetWriteDeadline).
		// Pour les WebSockets, net/http efface les délais quand la connexion
		// est reprise (Hijack) ; sa documentation ne le garantit pas, un test
		// (TestDelaiEcriture) le vérifie.
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  2 * time.Minute,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	// Shutdown attend la fin des requêtes en cours… y compris les connexions
	// longues, qui ne finissent jamais d'elles-mêmes. RegisterOnShutdown
	// prévient les handlers, qui se terminent alors proprement.
	srv.RegisterOnShutdown(serveur.Arrete)

	ecoute, err := net.Listen("tcp", adresse)
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
	case err := <-erreurs:
		return err
	case <-ctx.Done():
	}
	debut := time.Now()
	arret, annule := context.WithTimeout(context.Background(), 10*time.Second)
	defer annule()
	err = srv.Shutdown(arret)
	log.Info("serveur arrêté", "duree", time.Since(debut).Round(time.Millisecond).String())
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
