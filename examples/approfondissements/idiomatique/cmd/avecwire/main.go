// Command avecwire assemble le même service avec Wire, qui génère le code
// d'assemblage à la compilation (wire_gen.go). Pour le régénérer : go generate
// (la directive est dans wire_gen.go), ou go tool wire. Le dépôt de Wire est archivé
// depuis août 2025 : l'outil marche, mais n'évolue plus.
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

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
)

// horloge fournit l'heure au service. Wire choisit un fournisseur par son
// type de retour : func() time.Time.
func horloge() func() time.Time { return time.Now }

func main() {
	adresse := flag.String("adresse", "localhost:8080", "adresse d'écoute")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := assemble(web.Config{Adresse: *adresse}, log) // généré par Wire

	// Wire ne gère que la construction : le cycle de vie reste à écrire.
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
