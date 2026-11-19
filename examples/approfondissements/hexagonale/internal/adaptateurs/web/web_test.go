package web

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/domaine"
)

// faux remplace l'application : l'adaptateur se teste seul, sans métier.
type faux struct{ err error }

func (f faux) Emprunte(_ context.Context, membre, isbn string) (domaine.Emprunt, error) {
	return domaine.NouvelEmprunt("e1", isbn, membre, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)), f.err
}

func (f faux) Rend(context.Context, string) (int, error) { return 60, f.err }

func TestTraductionDesErreurs(t *testing.T) {
	cas := []struct {
		err     error
		statut  int
		contenu string
	}{
		{nil, 201, `"echeance":"2026-09-22T00:00:00Z"`},
		{domaine.ErrQuotaAtteint, 409, "3 emprunts au plus"},
		{domaine.ErrIntrouvable, 404, "introuvable"},
		{errors.New("disque plein"), 500, "erreur interne"},
	}
	for _, c := range cas {
		rec := httptest.NewRecorder()
		Routes(faux{c.err}).ServeHTTP(rec, httptest.NewRequest("POST", "/emprunts",
			strings.NewReader(`{"membre": "alice", "isbn": "go-1"}`)))
		if rec.Code != c.statut || !strings.Contains(rec.Body.String(), c.contenu) {
			t.Errorf("%v : %d %s", c.err, rec.Code, rec.Body)
		}
	}
}

func TestRetour(t *testing.T) {
	rec := httptest.NewRecorder()
	Routes(faux{}).ServeHTTP(rec, httptest.NewRequest("POST", "/emprunts/e1/retour", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"penalite_centimes":60`) {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}
