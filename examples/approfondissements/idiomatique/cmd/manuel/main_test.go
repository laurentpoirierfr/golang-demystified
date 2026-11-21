package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web/webtest"
)

func TestAssemble(t *testing.T) {
	srv := assemble(web.Config{Adresse: "localhost:0"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	webtest.VerifieAPI(t, srv.Handler)
}
