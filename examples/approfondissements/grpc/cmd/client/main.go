// Command client appelle le service des tâches.
//
//	client cree "Apprendre gRPC"
//	client trouve 1
//	client suit                    # affiche les tâches au fil de l'eau
//	client importe < titres.txt    # une tâche par ligne
//
// Options : -adresse http://localhost:8090, -protocole connect|grpc|grpcweb,
// -jeton secret.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/internal/service"
	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1/tachesv1connect"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, entree io.Reader, sortie io.Writer) error {
	drapeaux := flag.NewFlagSet("client", flag.ContinueOnError)
	adresse := drapeaux.String("adresse", "http://localhost:8090", "adresse du serveur")
	protocole := drapeaux.String("protocole", "connect", "connect, grpc ou grpcweb")
	jeton := drapeaux.String("jeton", "", "jeton d'authentification")
	if err := drapeaux.Parse(args); err != nil {
		return err
	}

	c, err := nouveauClient(*adresse, *protocole, *jeton)
	if err != nil {
		return err
	}

	switch commande := drapeaux.Arg(0); commande {
	case "cree":
		rep, err := c.Cree(ctx, &tachesv1.CreeRequest{Titre: drapeaux.Arg(1)})
		if err != nil {
			return err
		}
		affiche(sortie, rep.GetTache())
	case "trouve":
		id, err := strconv.ParseInt(drapeaux.Arg(1), 10, 64)
		if err != nil {
			return fmt.Errorf("identifiant invalide : %w", err)
		}
		rep, err := c.Trouve(ctx, &tachesv1.TrouveRequest{Id: id})
		if err != nil {
			return err
		}
		affiche(sortie, rep.GetTache())
	case "suit":
		flux, err := c.Suit(ctx, &tachesv1.SuitRequest{})
		if err != nil {
			return err
		}
		defer flux.Close()
		for {
			rep, err := flux.Receive()
			if errors.Is(err, io.EOF) || connect.CodeOf(err) == connect.CodeCanceled {
				return nil // fin du flux, ou Ctrl+C
			}
			if err != nil {
				return err
			}
			affiche(sortie, rep.GetTache())
		}
	case "importe":
		flux, err := c.Importe(ctx)
		if err != nil {
			return err
		}
		lignes := bufio.NewScanner(entree)
		for lignes.Scan() {
			if err := flux.Send(&tachesv1.ImporteRequest{Titre: lignes.Text()}); err != nil {
				break // l'erreur du serveur est renvoyée par CloseAndReceive
			}
		}
		rep, err := flux.CloseAndReceive()
		if err != nil {
			return err
		}
		if err := lignes.Err(); err != nil {
			return fmt.Errorf("lecture des titres : %w", err)
		}
		fmt.Fprintf(sortie, "%d tâches importées\n", rep.GetNombre())
	default:
		return fmt.Errorf("commande inconnue %q (cree, trouve, suit, importe)", commande)
	}
	return nil
}

// nouveauClient choisit le protocole. Le code généré, lui, ne change pas.
func nouveauClient(adresse, protocole, jeton string) (tachesv1connect.TacheServiceClient, error) {
	var options []connecthttp.Option
	switch protocole {
	case "connect":
	case "grpc":
		options = append(options, connecthttp.WithGRPC())
	case "grpcweb":
		options = append(options, connecthttp.WithGRPCWeb())
	default:
		return nil, fmt.Errorf("protocole inconnu %q", protocole)
	}
	var intercepteurs []connect.ClientInterceptor
	if jeton != "" {
		intercepteurs = append(intercepteurs, service.Jeton(jeton))
	}
	transport := connecthttp.NewTransport(clientHTTP(), adresse, options...)
	return tachesv1connect.NewTacheServiceClient(connect.NewClient(transport, intercepteurs...)), nil
}

// clientHTTP parle HTTP/2 en clair (h2c) aux adresses http://, comme l'exige
// gRPC, et HTTP/2 avec TLS aux adresses https://.
func clientHTTP() *http.Client {
	protocoles := new(http.Protocols)
	protocoles.SetHTTP2(true)
	protocoles.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: &http.Transport{Protocols: protocoles}}
}

func affiche(w io.Writer, t *tachesv1.Tache) {
	fmt.Fprintf(w, "%d\t%s\t%s\n", t.GetId(), t.GetTitre(), t.GetCreation().AsTime().Format("2006-01-02 15:04"))
}
