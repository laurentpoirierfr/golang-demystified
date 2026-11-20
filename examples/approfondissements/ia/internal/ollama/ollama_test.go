package ollama_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/ollama"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// fauxOllama rejoue des réponses NDJSON écrites à l'avance.
type fauxOllama struct {
	mu       sync.Mutex
	reponses []string
	requetes []map[string]any
}

func (f *fauxOllama) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var corps map[string]any
	json.NewDecoder(r.Body).Decode(&corps)
	f.requetes = append(f.requetes, corps)
	w.Header().Set("Content-Type", "application/x-ndjson")
	io.WriteString(w, f.reponses[0])
	f.reponses = f.reponses[1:]
}

func agent(t *testing.T, reponses ...string) (*ollama.Agent, *fauxOllama, *outils.Taches) {
	t.Helper()
	faux := &fauxOllama{reponses: reponses}
	srv := httptest.NewServer(faux)
	t.Cleanup(srv.Close)
	taches := &outils.Taches{}
	boite, _ := outils.Boite(taches)
	return ollama.Nouveau(srv.URL, "qwen3:1.7b", boite), faux, taches
}

func TestBoucleOutils(t *testing.T) {
	a, faux, taches := agent(t,
		`{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"ajouter_tache","arguments":{"titre":"Arroser les plantes"}}}]},"done":false}
{"message":{"role":"assistant","content":""},"done":true,"done_reason":"stop"}
`,
		`{"message":{"role":"assistant","content":"C'est "},"done":false}
{"message":{"role":"assistant","content":"ajouté."},"done":false}
{"message":{"role":"assistant","content":""},"done":true,"done_reason":"stop"}
`)
	var flux []string
	reponse, err := a.Demande(t.Context(), "Ajoute « Arroser les plantes »", func(s string) { flux = append(flux, s) })
	if err != nil {
		t.Fatal(err)
	}
	if reponse != "C'est ajouté." || strings.Join(flux, "|") != "C'est |ajouté." {
		t.Errorf("réponse %q, flux %q", reponse, flux)
	}
	if l := taches.Liste(true); len(l) != 1 || l[0].Titre != "Arroser les plantes" {
		t.Errorf("tâches : %+v", l)
	}

	// Les outils sont décrits au format d'Ollama ; le résultat revient en
	// message « tool », avec le nom de l'outil.
	req := faux.requetes[0]
	outil := req["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if req["model"] != "qwen3:1.7b" || req["stream"] != true || outil["name"] != "ajouter_tache" {
		t.Errorf("requête : %v", req)
	}
	messages := faux.requetes[1]["messages"].([]any)
	dernier := messages[len(messages)-1].(map[string]any)
	if dernier["role"] != "tool" || dernier["tool_name"] != "ajouter_tache" || dernier["content"] != "tâche 1 ajoutée : Arroser les plantes" {
		t.Errorf("résultat d'outil : %v", dernier)
	}
}

func TestErreurs(t *testing.T) {
	tests := []struct {
		nom, reponse, attendu string
	}{
		{"modèle absent", `{"error":"model \"qwen3:1.7b\" not found, try pulling it first"}` + "\n", "not found"},
		{"flux interrompu", `{"message":{"role":"assistant","content":"Je"},"done":false}` + "\n", io.ErrUnexpectedEOF.Error()},
		{"réponse illisible", "pas du JSON\n", "illisible"},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			a, _, _ := agent(t, tt.reponse)
			_, err := a.Demande(t.Context(), "?", nil)
			if err == nil || !strings.Contains(err.Error(), tt.attendu) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestStatutHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"server busy"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	boite, _ := outils.Boite(&outils.Taches{})
	_, err := ollama.Nouveau(srv.URL, "m", boite).Demande(t.Context(), "?", nil)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %v", err)
	}
}
