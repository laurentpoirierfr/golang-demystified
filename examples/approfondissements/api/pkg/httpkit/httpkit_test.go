package httpkit

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChaineOrdre(t *testing.T) {
	var ordre []string
	marque := func(nom string) Middleware {
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ordre = append(ordre, nom)
				h.ServeHTTP(w, r)
			})
		}
	}
	h := Chaine(http.NotFoundHandler(), marque("a"), marque("b"), marque("c"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got := strings.Join(ordre, ""); got != "abc" {
		t.Errorf("ordre = %q, attendu abc", got)
	}
}

func TestRecuperation(t *testing.T) {
	var journal bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&journal, nil))
	h := Chaine(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boum") }),
		Identifiant(), Recuperation(log))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "erreur interne") {
		t.Errorf("réponse = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(journal.String(), `"valeur":"boum"`) || !strings.Contains(journal.String(), "requete_id") {
		t.Errorf("panique mal journalisée : %s", journal.String())
	}
}

func TestLitJSON(t *testing.T) {
	cas := []struct {
		corps string
		ok    bool
	}{
		{`{"nom": "a"}`, true},
		{`{"nom": "a", "autre": 1}`, false},
		{`{"nom": "a"}{"nom": "b"}`, false},
		{`{"nom":`, false},
		{`{"nom": "` + strings.Repeat("x", TailleMax) + `"}`, false},
	}
	for _, c := range cas {
		var dest struct{ Nom string }
		r := httptest.NewRequest("POST", "/", strings.NewReader(c.corps))
		err := LitJSON(httptest.NewRecorder(), r, &dest)
		if (err == nil) != c.ok {
			t.Errorf("%.30q : erreur %v", c.corps, err)
		}
		if err != nil {
			if _, ok := errors.AsType[*ErrRequete](err); !ok {
				t.Errorf("erreur de type %T, attendu *ErrRequete", err)
			}
		}
	}
}
