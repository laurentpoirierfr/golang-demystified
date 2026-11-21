// Command avecfx assemble le même service avec Uber Fx : les dépendances
// sont résolues à l'exécution, par réflexion, et Fx gère aussi le cycle de
// vie (démarrage, arrêt sur signal, dans l'ordre inverse).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/infra"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
)

// module décrit le graphe : des constructeurs ordinaires, que Fx appelle
// dans le bon ordre, une seule fois chacun. fx.As indique l'interface
// fournie, comme wire.Bind.
var module = fx.Options(
	fx.Provide(
		fx.Annotate(infra.NouveauDepotMemoire, fx.As(new(inscription.Depot))),
		fx.Annotate(infra.NouveauCourrierJournal, fx.As(new(inscription.Courrier))),
		func() func() time.Time { return time.Now },
		fx.Annotate(inscription.Nouveau, fx.As(new(web.Inscripteur))),
		web.Routes,
		web.NouveauServeur,
	),
	fx.Invoke(demarre), // demande le serveur : tout le graphe est construit
)

// demarre accroche le serveur au cycle de vie de l'application.
func demarre(lc fx.Lifecycle, srv *http.Server, log *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr) // une erreur d'écoute arrête le démarrage
			if err != nil {
				return err
			}
			log.Info("écoute", "adresse", ln.Addr().String())
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("serveur", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("arrêt")
			return srv.Shutdown(ctx)
		},
	})
}

// application assemble l'application ; la configuration et le journal sont
// fournis de l'extérieur.
func application(cfg web.Config, log *slog.Logger, opts ...fx.Option) *fx.App {
	return fx.New(
		fx.Supply(cfg, log),
		// Les événements de Fx (fournisseurs, hooks) passent au niveau Debug :
		// utiles pour comprendre un démarrage, trop bavards au quotidien.
		fx.WithLogger(func() fxevent.Logger {
			l := &fxevent.SlogLogger{Logger: log}
			l.UseLogLevel(slog.LevelDebug)
			return l
		}),
		module,
		fx.Options(opts...),
	)
}

func main() {
	adresse := flag.String("adresse", "localhost:8080", "adresse d'écoute")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// Run démarre, attend SIGINT ou SIGTERM, puis arrête proprement.
	application(web.Config{Adresse: *adresse}, log).Run()
}
