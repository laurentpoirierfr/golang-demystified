// Command taches-cli est le client en ligne de commande de l'API des tâches.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	racine := cli.NouvelleRacine(cli.Options{Sortie: os.Stdout, Erreurs: os.Stderr})
	if err := racine.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}
