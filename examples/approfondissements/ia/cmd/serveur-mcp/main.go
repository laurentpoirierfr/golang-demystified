// Command serveur-mcp expose la liste de tâches par MCP.
//
// Par défaut, il dialogue sur l'entrée et la sortie standard (stdio) : c'est
// ainsi que Claude Code, Claude Desktop ou un éditeur lancent un serveur MCP
// local. Avec -http, il sert le transport « Streamable HTTP » sur le réseau.
//
// Le journal part sur la sortie d'erreur : en mode stdio, la sortie standard
// appartient au protocole, et une ligne de journal la corromprait.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/mcpserveur"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

func main() {
	adresse := flag.String("http", "", "adresse d'écoute HTTP (localhost:8090) ; stdio si vide")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *adresse); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, adresse string) error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	boite, err := outils.Boite(&outils.Taches{})
	if err != nil {
		return err
	}
	serveur := mcpserveur.Nouveau(boite, "1.0.0")

	if adresse == "" {
		log.Info("serveur MCP sur stdio")
		return serveur.Run(ctx, &mcp.StdioTransport{})
	}

	srv := &http.Server{
		Addr: adresse,
		// Le même serveur pour toutes les sessions : elles partagent la liste.
		Handler:           mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return serveur }, nil),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
		defer annule()
		_ = srv.Shutdown(arret)
	}()
	log.Info("serveur MCP sur HTTP", "adresse", "http://"+adresse)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
