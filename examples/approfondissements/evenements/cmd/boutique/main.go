// Command boutique démarre la boutique : l'API HTTP qui reçoit les commandes,
// et les gestionnaires d'événements de la saga.
//
// Sans configuration, tout tourne en mémoire. Avec BOUTIQUE_BASE (PostgreSQL)
// et NATS_URL, les événements passent par une outbox et par NATS JetStream.
//
//	BOUTIQUE_ADRESSE   adresse d'écoute (localhost:8080)
//	BOUTIQUE_BASE      postgres://… (facultatif)
//	NATS_URL           nats://localhost:4222 (facultatif)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/boutique"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/memoire"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/messagerie"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/postgres"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

// stockInitial approvisionne la boutique de démonstration.
var stockInitial = map[string]int{"clavier": 5, "souris": 10, "ecran": 2}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// journal est le notifieur de démonstration : il écrit la notification dans
// le journal, au lieu d'envoyer un courriel.
type journal struct{ log *slog.Logger }

func (j journal) Notifie(ctx context.Context, cle, client, message string) error {
	j.log.InfoContext(ctx, "notification", "client", client, "message", message, "cle", cle)
	return nil
}

// infrastructure regroupe ce qui dépend du mode : dépôt, publieur, abonnés,
// et les tâches de fond (le transitaire de l'outbox).
type infrastructure struct {
	depot    boutique.Depot
	publieur message.Publisher
	abonne   messagerie.Abonne
	fond     []func(context.Context) error
	ferme    func()
}

func run(ctx context.Context, getenv func(string) string, sortie io.Writer, pret chan<- string) error {
	log := slog.New(slog.NewJSONHandler(sortie, nil))
	wlog := watermill.NewSlogLoggerWithLevelMapping(log, map[slog.Level]slog.Level{
		slog.LevelInfo: slog.LevelDebug, // Watermill est bavard au niveau Info
	})

	var infra *infrastructure
	var err error
	if base, url := getenv("BOUTIQUE_BASE"), getenv("NATS_URL"); base != "" && url != "" {
		infra, err = production(ctx, base, url, wlog)
	} else {
		log.Warn("mode démonstration : événements et données en mémoire")
		infra, err = demonstration(wlog)
	}
	if err != nil {
		return err
	}
	defer infra.ferme()

	service := boutique.Nouveau(infra.depot, journal{log}, watermill.NewULID)
	routeur, err := messagerie.NouveauRouteur(infra.publieur, log)
	if err != nil {
		return err
	}
	if err := messagerie.AjouteGestionnaires(routeur, infra.abonne, wlog, service.Gestionnaires()...); err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           routes(service, infra.depot),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	adresse := getenv("BOUTIQUE_ADRESSE")
	if adresse == "" {
		adresse = "localhost:8080"
	}
	ecoute, err := net.Listen("tcp", adresse)
	if err != nil {
		return err
	}

	// Le routeur, le transitaire et le serveur HTTP tournent ensemble ; le
	// premier qui s'arrête arrête les autres.
	ctx, arrete := context.WithCancel(ctx)
	defer arrete()
	var wg sync.WaitGroup
	erreurs := make(chan error, 3+len(infra.fond))
	lance := func(f func() error) {
		wg.Go(func() {
			defer arrete()
			if err := f(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				erreurs <- err
			}
		})
	}
	lance(func() error { return routeur.Run(ctx) })
	for _, f := range infra.fond {
		lance(func() error { return f(ctx) })
	}
	<-routeur.Running()
	lance(func() error { return srv.Serve(ecoute) })
	log.Info("boutique démarrée", "adresse", ecoute.Addr().String())
	if pret != nil {
		pret <- ecoute.Addr().String()
	}

	<-ctx.Done()
	arret, annule := context.WithTimeout(context.Background(), 10*time.Second)
	defer annule()
	err = srv.Shutdown(arret)
	wg.Wait()
	close(erreurs)
	for e := range erreurs {
		err = errors.Join(err, e)
	}
	return err
}

// demonstration : un pub/sub en mémoire (GoChannel) et un dépôt en mémoire.
func demonstration(wlog watermill.LoggerAdapter) (*infrastructure, error) {
	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, wlog)
	bus, err := messagerie.NouveauBus(pubsub, wlog)
	if err != nil {
		return nil, err
	}
	return &infrastructure{
		depot:    memoire.Nouveau(bus, stockInitial),
		publieur: pubsub,
		abonne:   func(string) (message.Subscriber, error) { return pubsub, nil },
		ferme:    func() { pubsub.Close() },
	}, nil
}

// production : PostgreSQL (données, outbox, inbox) et NATS JetStream.
func production(ctx context.Context, base, url string, wlog watermill.LoggerAdapter) (*infrastructure, error) {
	depot, err := postgres.Ouvre(ctx, base, wlog)
	if err != nil {
		return nil, fmt.Errorf("base : %w", err)
	}
	for ref, n := range stockInitial {
		if err := depot.Approvisionne(ctx, ref, n); err != nil {
			depot.Close()
			return nil, err
		}
	}
	courtier, err := messagerie.PublieurNATS(url, wlog)
	if err != nil {
		depot.Close()
		return nil, fmt.Errorf("NATS : %w", err)
	}
	transitaire, err := depot.Transitaire(courtier)
	if err != nil {
		depot.Close()
		return nil, err
	}
	return &infrastructure{
		depot: depot,
		// Les gestionnaires publient via l'outbox (dans leur transaction) ;
		// ce publieur ne sert qu'à la file poison.
		publieur: courtier,
		abonne:   messagerie.AbonneNATS(url, wlog),
		fond:     []func(context.Context) error{transitaire.Run},
		ferme: func() {
			transitaire.Close()
			courtier.Close()
			depot.Close()
		},
	}, nil
}

func routes(service *boutique.Service, depot boutique.Depot) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /commandes", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Client   string              `json:"client"`
			Articles []evenement.Article `json:"articles"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}
		id, err := service.Passe(r.Context(), req.Client, req.Articles)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// 202 Accepted : la commande est enregistrée, son traitement suit.
		w.Header().Set("Location", "/commandes/"+id)
		ecritJSON(w, http.StatusAccepted, map[string]string{"id": id, "statut": string(boutique.EnAttente)})
	})
	mux.HandleFunc("GET /commandes/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, err := service.Commande(r.Context(), r.PathValue("id"))
		if errors.Is(err, boutique.ErrIntrouvable) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "erreur interne", http.StatusInternalServerError)
			return
		}
		ecritJSON(w, http.StatusOK, c)
	})
	mux.HandleFunc("GET /stock/{ref}", func(w http.ResponseWriter, r *http.Request) {
		n, err := depot.Stock(r.Context(), r.PathValue("ref"))
		if err != nil {
			http.Error(w, "erreur interne", http.StatusInternalServerError)
			return
		}
		ecritJSON(w, http.StatusOK, map[string]int{r.PathValue("ref"): n})
	})
	return mux
}

func ecritJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	_ = json.NewEncoder(w).Encode(v)
}
