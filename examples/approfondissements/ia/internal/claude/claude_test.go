package claude_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/claude"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// fauxClaude imite l'API Messages : il rejoue, réponse après réponse, des
// flux d'événements SSE écrits à l'avance, et garde les requêtes reçues.
type fauxClaude struct {
	mu       sync.Mutex
	reponses []string
	requetes []map[string]any
	entetes  []http.Header
}

func (f *fauxClaude) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var corps map[string]any
	b, _ := io.ReadAll(r.Body)
	json.Unmarshal(b, &corps)
	f.requetes = append(f.requetes, corps)
	f.entetes = append(f.entetes, r.Header.Clone())
	if len(f.reponses) == 0 {
		http.Error(w, `{"type":"error","error":{"type":"invalid_request_error","message":"plus de réponse"}}`, 400)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	io.WriteString(w, f.reponses[0])
	f.reponses = f.reponses[1:]
}

// Flux SSE de l'API Messages.
func evenement(nom, donnees string) string { return "event: " + nom + "\ndata: " + donnees + "\n\n" }

func debut() string {
	return evenement("message_start", `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":1}}}`)
}

func fin(raison string) string {
	return evenement("message_delta", `{"type":"message_delta","delta":{"stop_reason":"`+raison+`"},"usage":{"output_tokens":20}}`) +
		evenement("message_stop", `{"type":"message_stop"}`)
}

func appelOutil(index int, id, nom string, morceaux ...string) string {
	s := evenement("content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"%s","name":"%s","input":{}}}`, index, id, nom))
	for _, m := range morceaux { // l'entrée arrive en morceaux de JSON
		j, _ := json.Marshal(m)
		s += evenement("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}`, index, j))
	}
	return s + evenement("content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, index))
}

func texte(index int, morceaux ...string) string {
	s := evenement("content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, index))
	for _, m := range morceaux {
		j, _ := json.Marshal(m)
		s += evenement("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, index, j))
	}
	return s + evenement("content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, index))
}

func agent(t *testing.T, reponses ...string) (*claude.Agent, *fauxClaude, *outils.Taches) {
	t.Helper()
	faux := &fauxClaude{reponses: reponses}
	srv := httptest.NewServer(faux)
	t.Cleanup(srv.Close)
	taches := &outils.Taches{}
	boite, err := outils.Boite(taches)
	if err != nil {
		t.Fatal(err)
	}
	client := anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("cle-de-test"), option.WithMaxRetries(0))
	return claude.Nouveau(client, boite), faux, taches
}

func TestBoucleOutils(t *testing.T) {
	a, faux, taches := agent(t,
		// 1. Claude appelle deux outils dans la même réponse.
		debut()+appelOutil(0, "toolu_1", "ajouter_tache", `{"titre": "Ach`, `eter du pain"}`)+
			appelOutil(1, "toolu_2", "terminer_tache", `{"id": 7}`)+fin("tool_use"),
		// 2. Puis il répond, en flux.
		debut()+texte(0, "C'est noté : ", "« Acheter du pain » est ajoutée.")+fin("end_turn"),
	)
	var appels []string
	a.Observe = func(ap outils.Appel) { appels = append(appels, ap.Outil) }
	var flux []string

	reponse, err := a.Demande(t.Context(), "Ajoute « Acheter du pain »", func(s string) { flux = append(flux, s) })
	if err != nil {
		t.Fatal(err)
	}
	if reponse != "C'est noté : « Acheter du pain » est ajoutée." || len(flux) != 2 {
		t.Errorf("réponse %q, flux %q", reponse, flux)
	}
	if l := taches.Liste(true); len(l) != 1 || l[0].Titre != "Acheter du pain" {
		t.Errorf("tâches : %+v", l)
	}
	if fmt.Sprint(appels) != "[ajouter_tache terminer_tache]" {
		t.Errorf("appels : %v", appels)
	}

	// La première requête : modèle, effort, repli, outils en flux.
	req := faux.requetes[0]
	if req["model"] != "claude-opus-5-5" || req["stream"] != true || req["fallbacks"] != "default" {
		t.Errorf("requête : model=%v stream=%v fallbacks=%v", req["model"], req["stream"], req["fallbacks"])
	}
	if eff := req["output_config"].(map[string]any)["effort"]; eff != "low" {
		t.Errorf("effort : %v", eff)
	}
	outil := req["tools"].([]any)[0].(map[string]any)
	if outil["name"] != "ajouter_tache" || outil["eager_input_streaming"] != true {
		t.Errorf("outil : %v", outil)
	}
	if !strings.Contains(faux.entetes[0].Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Errorf("en-tête beta : %q", faux.entetes[0].Get("Anthropic-Beta"))
	}

	// La deuxième requête renvoie les deux résultats dans UN message, l'erreur
	// de l'outil marquée is_error : Claude peut se corriger.
	messages := faux.requetes[1]["messages"].([]any)
	dernier := messages[len(messages)-1].(map[string]any)
	resultats := dernier["content"].([]any)
	if dernier["role"] != "user" || len(resultats) != 2 {
		t.Fatalf("résultats : %v", dernier)
	}
	r1, r2 := resultats[0].(map[string]any), resultats[1].(map[string]any)
	if r1["tool_use_id"] != "toolu_1" || r1["is_error"] == true || !strings.Contains(fmt.Sprint(r1["content"]), "tâche 1 ajoutée") {
		t.Errorf("premier résultat : %v", r1)
	}
	if r2["tool_use_id"] != "toolu_2" || r2["is_error"] != true || !strings.Contains(fmt.Sprint(r2["content"]), "introuvable") {
		t.Errorf("second résultat : %v", r2)
	}
}

func TestEntreeInvalide(t *testing.T) {
	a, faux, taches := agent(t,
		debut()+appelOutil(0, "toolu_1", "ajouter_tache", `{"titre": 42}`)+fin("tool_use"),
		debut()+texte(0, "Désolé.")+fin("end_turn"),
	)
	if _, err := a.Demande(t.Context(), "?", nil); err != nil {
		t.Fatal(err)
	}
	if len(taches.Liste(true)) != 0 {
		t.Error("une entrée invalide ne doit rien créer")
	}
	messages := faux.requetes[1]["messages"].([]any)
	r := messages[len(messages)-1].(map[string]any)["content"].([]any)[0].(map[string]any)
	if r["is_error"] != true || !strings.Contains(fmt.Sprint(r["content"]), "schéma") {
		t.Errorf("résultat : %v", r)
	}
}

func TestRefus(t *testing.T) {
	a, _, _ := agent(t, debut()+fin("refusal"))
	if _, err := a.Demande(t.Context(), "?", nil); !errors.Is(err, claude.ErrRefus) {
		t.Errorf("err = %v", err)
	}
}

func TestBoucleInfinie(t *testing.T) {
	var reponses []string
	for range claude.MaxTours + 1 {
		reponses = append(reponses, debut()+appelOutil(0, "toolu_x", "lister_taches", `{}`)+fin("tool_use"))
	}
	a, faux, _ := agent(t, reponses...)
	if _, err := a.Demande(t.Context(), "?", nil); err == nil || !strings.Contains(err.Error(), "allers-retours") {
		t.Errorf("err = %v", err)
	}
	if len(faux.requetes) != claude.MaxTours {
		t.Errorf("%d requêtes, attendu %d", len(faux.requetes), claude.MaxTours)
	}
}

func TestErreurAPI(t *testing.T) {
	a, _, _ := agent(t) // aucune réponse : le faux serveur répond 400
	_, err := a.Demande(t.Context(), "?", nil)
	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 {
		t.Errorf("err = %v", err)
	}
}
