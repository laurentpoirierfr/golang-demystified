package intergiciel_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/pkg/intergiciel"
)

// trace note son passage, à l'aller et au retour.
func trace(nom string, journal *[]string) intergiciel.Intergiciel {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*journal = append(*journal, "→ "+nom)
			suivant.ServeHTTP(w, r)
			*journal = append(*journal, "← "+nom)
		})
	}
}

func TestOrdre(t *testing.T) {
	var journal []string
	h := intergiciel.Chaine(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { journal = append(journal, "handler") }),
		trace("A", &journal), trace("B", &journal),
	)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got := strings.Join(journal, " "); got != "→ A → B handler ← B ← A" {
		t.Errorf("ordre : %s", got)
	}
}

func TestJournalEtRecuperation(t *testing.T) {
	var sortie bytes.Buffer
	log := slog.New(slog.NewTextHandler(&sortie, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == "duree" {
				return slog.Attr{}
			}
			return a
		},
	}))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	mux.HandleFunc("GET /boum", func(http.ResponseWriter, *http.Request) { panic("oups") })
	// Recupere à l'extérieur, Journal à l'intérieur : la panique n'atteint
	// pas Journal, qui ne note donc pas la requête /boum.
	h := intergiciel.Chaine(mux, intergiciel.Recupere(log), intergiciel.Journal(log))

	for _, chemin := range []string{"/ok", "/boum"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", chemin, nil))
		t.Logf("%s → %d", chemin, rec.Code)
	}
	attendu := "level=INFO msg=requête methode=GET chemin=/ok statut=202\n" +
		"level=ERROR msg=panique valeur=oups chemin=/boum\n"
	if sortie.String() != attendu {
		t.Errorf("journal :\n%s\nattendu :\n%s", sortie.String(), attendu)
	}
}
