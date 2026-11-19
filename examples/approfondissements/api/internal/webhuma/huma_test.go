package webhuma

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
)

func nouveauStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestScenario utilise humatest : un routeur de test, sans réseau, et des
// fonctions d'appel qui encodent le corps en JSON.
func TestScenario(t *testing.T) {
	_, api := humatest.New(t)
	Enregistre(api, nouveauStore(t))

	etapes := []struct {
		nom      string
		reponse  *httptest.ResponseRecorder
		statut   int
		contient string
	}{
		{"création", api.Post("/taches", map[string]any{"titre": "  Relire  "}), 201, `"titre":"Relire"`},
		{"détail", api.Get("/taches/1"), 200, `"faite":false`},
		{"modification", api.Patch("/taches/1", map[string]any{"faite": true}), 200, `"faite":true`},
		{"liste", api.Get("/taches"), 200, `"id":1`},
		{"titre blanc (domaine)", api.Post("/taches", map[string]any{"titre": " "}), 422, "titre : obligatoire"},
		{"titre trop long (schéma)", api.Post("/taches", map[string]any{"titre": strings.Repeat("x", 201)}), 422, "expected length <= 200"},
		{"champ inconnu (schéma)", api.Post("/taches", map[string]any{"titre": "a", "priorite": 1}), 422, "unexpected property"},
		{"faite absent (schéma)", api.Patch("/taches/1", map[string]any{}), 422, "expected required property faite"},
		{"identifiant nul (schéma)", api.Get("/taches/0"), 422, "expected number >= 1"},
		{"introuvable", api.Get("/taches/99"), 404, "tâche introuvable"},
		{"suppression", api.Delete("/taches/1"), 204, ""},
		{"santé", api.Get("/sante"), 200, `"statut":"ok"`},
	}
	for _, e := range etapes {
		if e.reponse.Code != e.statut {
			t.Errorf("%s : statut %d, attendu %d (%s)", e.nom, e.reponse.Code, e.statut, e.reponse.Body)
		}
		if !strings.Contains(e.reponse.Body.String(), e.contient) {
			t.Errorf("%s : %q absent de %s", e.nom, e.contient, e.reponse.Body)
		}
	}
}

// TestContratDeduit vérifie le contrat OpenAPI que Huma déduit des types Go.
func TestContratDeduit(t *testing.T) {
	_, api := humatest.New(t)
	Enregistre(api, nouveauStore(t))

	contrat := api.OpenAPI()
	if contrat.OpenAPI != "3.1.0" {
		t.Errorf("version OpenAPI = %s", contrat.OpenAPI)
	}
	post := contrat.Paths["/taches"].Post
	if post == nil || post.OperationID != "cree-tache" {
		t.Fatalf("opération POST /taches absente : %+v", post)
	}
	schema := contrat.Components.Schemas.Map()["CreeEntreeBody"]
	if schema == nil || *schema.Properties["titre"].MaxLength != 200 {
		t.Errorf("le schéma du corps devrait limiter le titre à 200 caractères : %+v", schema)
	}
}

// TestContratEtDocumentation vérifie ce que Huma sert en plus des opérations.
func TestContratEtDocumentation(t *testing.T) {
	h := Routes(nouveauStore(t))
	for chemin, attendu := range map[string]string{
		"/openapi.yaml": "operationId: liste-taches",
		"/openapi.json": `"openapi":"3.1.0"`,
		"/docs":         "<!doctype html>",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", chemin, nil))
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), attendu) {
			t.Errorf("%s : %d, %q absent", chemin, rec.Code, attendu)
		}
	}
}
