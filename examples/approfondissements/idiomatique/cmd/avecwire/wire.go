//go:build wireinject

// Ce fichier décrit le graphe pour Wire ; il n'est jamais compilé dans le
// programme (contrainte wireinject). « go tool wire » le lit et écrit
// wire_gen.go : le même code que l'on écrirait à la main.

package main

import (
	"log/slog"
	"net/http"

	"github.com/google/wire"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/infra"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
)

// ensembleInfra regroupe les fournisseurs, et indique quelle structure
// concrète satisfait quelle interface : Wire ne le devine pas.
var ensembleInfra = wire.NewSet(
	infra.NouveauDepotMemoire,
	wire.Bind(new(inscription.Depot), new(*infra.DepotMemoire)),
	infra.NouveauCourrierJournal,
	wire.Bind(new(inscription.Courrier), new(*infra.CourrierJournal)),
)

func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	wire.Build(
		ensembleInfra,
		horloge,
		inscription.Nouveau,
		wire.Bind(new(web.Inscripteur), new(*inscription.Service)),
		web.Routes,
		web.NouveauServeur,
	)
	return nil // remplacé par le code généré
}
