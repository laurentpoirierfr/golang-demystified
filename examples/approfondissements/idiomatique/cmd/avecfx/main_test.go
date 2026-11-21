package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/fx"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/web/webtest"
)

func silencieux() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestAssemble(t *testing.T) {
	var srv *http.Server
	app := application(web.Config{Adresse: "localhost:0"}, silencieux(), fx.Populate(&srv))
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	webtest.VerifieAPI(t, srv.Handler)
}

// Le cycle de vie : Start appelle OnStart (le serveur écoute), Stop appelle
// OnStop (le serveur s'arrête).
func TestCycleDeVie(t *testing.T) {
	var journal bytes.Buffer
	log := slog.New(slog.NewTextHandler(&journal, nil))
	app := application(web.Config{Adresse: "localhost:0"}, log)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	for _, attendu := range []string{"msg=écoute", "msg=arrêt"} {
		if !strings.Contains(journal.String(), attendu) {
			t.Errorf("journal sans %q :\n%s", attendu, journal.String())
		}
	}
}

// Une adresse invalide fait échouer le démarrage : OnStart renvoie
// l'erreur, et Fx arrête ce qui avait démarré.
func TestDemarrageImpossible(t *testing.T) {
	app := application(web.Config{Adresse: "adresse invalide"}, silencieux())
	if err := app.Start(context.Background()); err == nil {
		t.Error("le démarrage aurait dû échouer")
	}
}

// Un graphe incomplet n'est détecté qu'à l'exécution, quand Fx construit
// l'application. fx.ValidateApp le vérifie sans rien démarrer : c'est le test
// à écrire dans tout projet qui utilise Fx.
func TestGrapheIncomplet(t *testing.T) {
	err := fx.ValidateApp(fx.NopLogger, fx.Supply(web.Config{}), module) // le journal manque
	t.Logf("%v", err)
	if err == nil || !strings.Contains(err.Error(), "missing type: *slog.Logger") {
		t.Errorf("erreur attendue, obtenu %v", err)
	}
	if err := fx.ValidateApp(fx.NopLogger, fx.Supply(web.Config{}, silencieux()), module); err != nil {
		t.Errorf("graphe complet : %v", err)
	}
}
