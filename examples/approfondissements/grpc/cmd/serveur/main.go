// Command serveur démarre le service des tâches. Une seule adresse sert les
// trois protocoles : Connect (HTTP/1.1 ou HTTP/2, JSON ou Protobuf), gRPC
// (HTTP/2) et gRPC-Web (navigateurs).
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

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpcreflect/v2"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run contient tout le programme ; pret, s'il n'est pas nil, reçoit l'adresse
// d'écoute une fois le serveur démarré.
func run(ctx context.Context, getenv func(string) string, sortie io.Writer, pret chan<- string) error {
	adresse := getenv("TACHES_ADRESSE")
	if adresse == "" {
		adresse = "localhost:8090"
	}
	log := slog.New(slog.NewJSONHandler(sortie, nil))

	// Les intercepteurs s'appliquent à toutes les méthodes, dans l'ordre :
	// le premier enveloppe tous les autres.
	intercepteurs := []connect.ServerInterceptor{service.Recuperation(log), service.Journal(log)}
	if jeton := getenv("TACHES_JETON"); jeton != "" {
		intercepteurs = append(intercepteurs, service.Authentification(jeton))
	}
	serveur := connect.NewServer(intercepteurs...)
	service.Nouveau().Enregistre(serveur)
	// La réflexion décrit les services aux outils (grpcurl, buf curl,
	// Postman) : ils peuvent les appeler sans le fichier .proto.
	grpcreflect.Register(serveur)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, serveur) // une route par méthode
	mux.HandleFunc("GET /sante", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// gRPC exige HTTP/2. Sans TLS, c'est « h2c » (HTTP/2 en clair), que
	// net/http sait servir depuis Go 1.24, sans dépendance.
	protocoles := new(http.Protocols)
	protocoles.SetHTTP1(true)
	protocoles.SetUnencryptedHTTP2(true)
	srv := &http.Server{
		Handler:           mux,
		Protocols:         protocoles,
		ReadHeaderTimeout: 5 * time.Second,
		// Pas de WriteTimeout : il couperait les flux de longue durée (Suit).
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

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
	log.Info("arrêt en cours")
	arret, annule := context.WithTimeout(context.Background(), 10*time.Second)
	defer annule()
	// Shutdown attend la fin des requêtes, mais pas celle des flux sans fin :
	// au bout du délai, ils sont coupés.
	if err := srv.Shutdown(arret); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}
