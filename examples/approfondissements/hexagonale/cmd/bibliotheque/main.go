// Command bibliotheque assemble l'application : c'est le seul endroit qui
// connaît à la fois les ports et les adaptateurs concrets.
package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/adaptateurs/memoire"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/adaptateurs/notification"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/adaptateurs/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/application"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/domaine"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/horloge"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/identifiant"
)

func main() {
	journal := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Adaptateurs secondaires : en mémoire pour la démonstration. Une base de
	// données implémenterait les mêmes ports.
	depot := memoire.Nouveau(
		[]domaine.Livre{{ISBN: "978-2-0000-0001-1", Titre: "Le langage Go"}},
		[]domaine.Membre{{ID: "alice", Nom: "Alice", Email: "alice@example.com"}},
	)

	app := application.NouveauService(application.Dependances{
		Catalogue:    depot,
		Emprunts:     depot,
		Transactions: depot,
		Notifieur:    notification.Journal{Log: journal},
		Horloge:      horloge.Systeme{},
		Identifiants: identifiant.UUID{},
		Journal:      journal,
	})

	// Adaptateur primaire : HTTP. Une interface en ligne de commande ou un
	// consommateur de messages piloterait les mêmes cas d'usage.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           web.Routes(app),
		ReadHeaderTimeout: 5 * time.Second,
	}
	journal.Info("bibliothèque démarrée", "adresse", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
