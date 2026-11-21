// Package webtest vérifie qu'un handler sert l'API d'inscription : les trois
// assemblages (à la main, Wire, Fx) passent le même contrôle.
package webtest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// VerifieAPI inscrit un membre, puis le même une seconde fois.
func VerifieAPI(t *testing.T, h http.Handler) {
	t.Helper()
	for _, cas := range []struct {
		corps  string
		statut int
	}{
		{`{"courriel": "Ada@Exemple.fr", "nom": "Ada"}`, http.StatusCreated},
		{`{"courriel": "ada@exemple.fr", "nom": "Ada"}`, http.StatusConflict},
		{`{"courriel": "pas-une-adresse", "nom": "Bob"}`, http.StatusBadRequest},
		{`{"courriel": `, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/membres", strings.NewReader(cas.corps)))
		if w.Code != cas.statut {
			t.Errorf("%s : %d %s, attendu %d", cas.corps, w.Code, strings.TrimSpace(w.Body.String()), cas.statut)
		}
	}
}
