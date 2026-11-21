package main

import (
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web/webtest"
)

func TestAssemble(t *testing.T) {
	srv := assemble(web.Config{Adresse: "localhost:0"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	webtest.VerifieAPI(t, srv.Handler)
}

// Un graphe incomplet est refusé par Wire, avant toute compilation du
// programme : l'erreur nomme le type sans fournisseur.
func TestGrapheIncomplet(t *testing.T) {
	if testing.Short() {
		t.Skip("lance go tool wire")
	}
	sortie, err := exec.Command("go", "tool", "wire", "check", "./testdata/incomplet").CombinedOutput()
	t.Logf("%s", sortie)
	if err == nil || !strings.Contains(string(sortie), "no provider found for func() time.Time") {
		t.Errorf("erreur attendue, obtenu %v", err)
	}
}
