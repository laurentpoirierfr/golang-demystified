package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func requete(h http.Handler, methode, chemin, corps string, entetes ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(methode, chemin, strings.NewReader(corps))
	for i := 0; i+1 < len(entetes); i += 2 {
		r.Header.Set(entetes[i], entetes[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestBonjour(t *testing.T) {
	w := requete(Routes(), "GET", "/bonjour/Ada", "")
	if w.Code != 200 || w.Body.String() != "Bonjour, Ada !\n" {
		t.Errorf("%d %q", w.Code, w.Body)
	}
}

func TestAdditionne(t *testing.T) {
	h := Routes()
	w := requete(h, "POST", "/additions", `{"a": 40, "b": 2}`)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"somme":42}` || w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("%d %q %v", w.Code, w.Body, w.Header())
	}
	if w := requete(h, "POST", "/additions", `{"a": `); w.Code != http.StatusBadRequest {
		t.Errorf("JSON invalide : %d", w.Code)
	}
	if w := requete(h, "POST", "/additions", `{"a": 1, "b": "`+strings.Repeat("x", 2000)+`"}`); w.Code != http.StatusBadRequest {
		t.Errorf("corps trop grand : %d", w.Code)
	}
	if w := requete(h, "GET", "/additions", ""); w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") == "" {
		t.Errorf("GET : %d, Allow %q", w.Code, w.Header().Get("Allow"))
	}
}

func TestExigeCle(t *testing.T) {
	h := ExigeCle("s3cret", Routes())
	if w := requete(h, "GET", "/bonjour/Ada", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("sans clé : %d", w.Code)
	}
	if w := requete(h, "GET", "/bonjour/Ada", "", "X-Cle", "s3cre"); w.Code != http.StatusUnauthorized {
		t.Errorf("mauvaise clé : %d", w.Code)
	}
	if w := requete(h, "GET", "/bonjour/Ada", "", "X-Cle", "s3cret"); w.Code != 200 {
		t.Errorf("bonne clé : %d", w.Code)
	}
}
