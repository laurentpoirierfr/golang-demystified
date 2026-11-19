package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

// nouveauServeur assemble l'API avec une vraie base SQLite en mémoire : le
// test couvre le service de bout en bout, sans réseau ni fichier.
func nouveauServeur(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()
	s, err := store.Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	var journal bytes.Buffer
	h, err := Nouveau(s, slog.New(slog.NewJSONHandler(&journal, nil))).Routes()
	if err != nil {
		t.Fatal(err)
	}
	return h, &journal
}

func appel(t *testing.T, h http.Handler, methode, chemin, corps string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(methode, chemin, strings.NewReader(corps))
	if corps != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestScenario(t *testing.T) {
	h, _ := nouveauServeur(t)

	etapes := []struct {
		nom, methode, chemin, corps string
		statut                      int
		contient                    string
	}{
		{"liste vide", "GET", "/taches", "", 200, "[]"},
		{"création", "POST", "/taches", `{"titre": "  Relire le livre  "}`, 201, `"titre":"Relire le livre"`},
		{"détail", "GET", "/taches/1", "", 200, `"faite":false`},
		{"modification", "PATCH", "/taches/1", `{"faite": true}`, 200, `"faite":true`},
		{"liste", "GET", "/taches", "", 200, `"id":1`},
		{"titre vide", "POST", "/taches", `{"titre": " "}`, 422, "titre : obligatoire"},
		{"titre trop long (contrat)", "POST", "/taches", `{"titre": "` + strings.Repeat("x", 201) + `"}`, 400, "maximum string length is 200"},
		{"champ inconnu (contrat)", "POST", "/taches", `{"titre": "a", "priorite": 1}`, 400, `property \"priorite\" is unsupported`},
		{"deux objets", "POST", "/taches", `{"titre": "a"}{"titre": "b"}`, 400, "un seul objet"},
		{"faite absent (contrat)", "PATCH", "/taches/1", `{}`, 400, `property \"faite\" is missing`},
		{"identifiant invalide", "GET", "/taches/abc", "", 400, "paramètre invalide"},
		{"identifiant nul (contrat)", "GET", "/taches/0", "", 400, "number must be at least 1"},
		{"introuvable", "GET", "/taches/99", "", 404, "tâche introuvable"},
		{"suppression", "DELETE", "/taches/1", "", 204, ""},
		{"déjà supprimée", "DELETE", "/taches/1", "", 404, "tâche introuvable"},
		{"méthode interdite", "PUT", "/taches/1", "", 405, ""},
		{"santé", "GET", "/sante", "", 200, `"statut":"ok"`},
		{"contrat", "GET", "/openapi.yaml", "", 200, "operationId: creeTache"},
		{"documentation", "GET", "/docs", "", 200, "swagger-ui-bundle.js"},
	}
	for _, e := range etapes {
		rec := appel(t, h, e.methode, e.chemin, e.corps)
		if rec.Code != e.statut {
			t.Errorf("%s : statut %d, attendu %d (%s)", e.nom, rec.Code, e.statut, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), e.contient) {
			t.Errorf("%s : %q absent de %s", e.nom, e.contient, rec.Body)
		}
	}
}

// TestContentType vérifie que le contrat impose le type du corps.
func TestContentType(t *testing.T) {
	h, _ := nouveauServeur(t)
	req := httptest.NewRequest("POST", "/taches", strings.NewReader(`{"titre": "a"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req) // sans Content-Type
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "Content-Type") {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}

func TestCreationRenvoieLaTache(t *testing.T) {
	h, _ := nouveauServeur(t)
	rec := appel(t, h, "POST", "/taches", `{"titre": "Tester"}`)

	var cree tache.Tache
	if err := json.NewDecoder(rec.Body).Decode(&cree); err != nil {
		t.Fatal(err)
	}
	if loc := rec.Header().Get("Location"); loc != "/taches/1" || cree.ID != 1 {
		t.Errorf("Location = %q, ID = %d", loc, cree.ID)
	}
}

func TestIdentifiantEtJournal(t *testing.T) {
	h, journal := nouveauServeur(t)

	req := httptest.NewRequest("GET", "/taches", nil)
	req.Header.Set("X-Request-ID", "test-42")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "test-42" {
		t.Errorf("X-Request-ID = %q", got)
	}
	var ligne map[string]any
	if err := json.Unmarshal(journal.Bytes(), &ligne); err != nil {
		t.Fatalf("journal illisible : %v\n%s", err, journal)
	}
	if ligne["requete_id"] != "test-42" || ligne["statut"] != float64(200) || ligne["chemin"] != "/taches" {
		t.Errorf("journal inattendu : %v", ligne)
	}

	// Sans en-tête, un identifiant est généré.
	if id := appel(t, h, "GET", "/taches", "").Header().Get("X-Request-ID"); len(id) != 36 {
		t.Errorf("identifiant généré inattendu : %q", id)
	}
}
