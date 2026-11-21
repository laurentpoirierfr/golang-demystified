// Command manuel assemble le service d'inscription à la main : c'est
// l'approche la plus répandue en Go. Tout le câblage tient dans assemble,
// du code ordinaire que le compilateur vérifie.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/infra"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
)

// assemble construit le graphe de dépendances, dans l'ordre.
func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	depot := infra.NouveauDepotMemoire()
	courrier := infra.NouveauCourrierJournal(log)
	service := inscription.Nouveau(depot, courrier, time.Now)
	return web.NouveauServeur(cfg, web.Routes(service))
}

func main() {
	adresse := flag.String("adresse", "localhost:8080", "adresse d'écoute")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := assemble(web.Config{Adresse: *adresse}, log)

	// Le cycle de vie, à la main aussi : démarrer, attendre un signal, arrêter.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		log.Info("écoute", "adresse", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("serveur", "err", err)
			stop()
		}
	}()
	<-ctx.Done()
	arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
	defer annule()
	if err := srv.Shutdown(arret); err != nil {
		log.Error("arrêt", "err", err)
	}
	log.Info("arrêté")
}
