//go:build wireinject

// Un graphe auquel il manque l'horloge, pour TestGrapheIncomplet.
package incomplet

import (
	"log/slog"
	"net/http"

	"github.com/google/wire"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/infra"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
)

func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	wire.Build(
		infra.NouveauDepotMemoire,
		wire.Bind(new(inscription.Depot), new(*infra.DepotMemoire)),
		infra.NouveauCourrierJournal,
		wire.Bind(new(inscription.Courrier), new(*infra.CourrierJournal)),
		inscription.Nouveau, // demande un func() time.Time, que personne ne fournit
		wire.Bind(new(web.Inscripteur), new(*inscription.Service)),
		web.Routes,
		web.NouveauServeur,
	)
	return nil
}
