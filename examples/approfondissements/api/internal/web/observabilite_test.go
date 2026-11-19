package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/telemetrie"
)

// serveurObserve assemble l'API comme cmd/taches, avec des traces conservées
// en mémoire au lieu d'être envoyées à un collecteur.
func serveurObserve(t *testing.T) (http.Handler, *tracetest.InMemoryExporter, *bytes.Buffer) {
	t.Helper()
	traces := tracetest.NewInMemoryExporter()
	tel, err := telemetrie.Demarre(telemetrie.Options{
		Service: "api-taches", Version: "test", Exportateur: traces, Synchrone: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tel.Arrete(t.Context()) })

	s, err := store.Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	s.Instrumente(tel.Traces)

	var journal bytes.Buffer
	log := slog.New(telemetrie.Journal{Handler: slog.NewJSONHandler(&journal, nil)})
	srv := Nouveau(s, log)
	if err := srv.Instrumente(tel.Metriques); err != nil {
		t.Fatal(err)
	}
	api, err := srv.Routes()
	if err != nil {
		t.Fatal(err)
	}

	racine := tel.Handler("api-taches", api)
	return racine, traces, &journal
}

func TestTracesJournauxEtMetriques(t *testing.T) {
	h, traces, journal := serveurObserve(t)

	req := httptest.NewRequest("POST", "/taches", strings.NewReader(`{"titre": "Observer"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("statut %d : %s", rec.Code, rec.Body)
	}

	// 1. Traces : un span pour la requête, et un span enfant pour le stockage.
	spans := traces.GetSpans()
	noms := map[string]tracetest.SpanStub{}
	for _, s := range spans {
		noms[s.Name] = s
	}
	serveur, ok := noms["POST /taches"]
	if !ok {
		t.Fatalf("span de la requête absent ; spans : %v", spanNoms(spans))
	}
	stockage, ok := noms["store.Cree"]
	if !ok {
		t.Fatalf("span du stockage absent ; spans : %v", spanNoms(spans))
	}
	if stockage.Parent.SpanID() != serveur.SpanContext.SpanID() {
		t.Error("le span du stockage devrait être l'enfant du span de la requête")
	}
	traceID := serveur.SpanContext.TraceID().String()

	// 2. Journaux : la ligne de la requête porte l'identifiant de la trace.
	var ligne map[string]any
	if err := json.Unmarshal(bytes.Split(bytes.TrimSpace(journal.Bytes()), []byte("\n"))[0], &ligne); err != nil {
		t.Fatalf("journal illisible : %v\n%s", err, journal)
	}
	if ligne["trace_id"] != traceID {
		t.Errorf("trace_id du journal = %v, attendu %s", ligne["trace_id"], traceID)
	}

	// 3. Métriques : la métrique métier et celles de otelhttp, au format Prometheus.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	metriques := rec.Body.String()
	for _, attendu := range []string{
		"taches_creees_total{",
		"http_server_request_duration_seconds_bucket{",
		`http_route="/taches"`,
		`service_name="api-taches"`,
	} {
		if !strings.Contains(metriques, attendu) {
			t.Errorf("%q absent de /metrics", attendu)
		}
	}
}

func TestErreurDansLaTrace(t *testing.T) {
	h, traces, _ := serveurObserve(t)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/taches/42", nil))

	for _, s := range traces.GetSpans() {
		if s.Name == "store.Trouve" && s.Status.Code.String() == "Error" {
			t.Error("une tâche introuvable ne doit pas marquer le span en erreur")
		}
	}
}

func spanNoms(spans tracetest.SpanStubs) []string {
	var noms []string
	for _, s := range spans {
		noms = append(noms, s.Name)
	}
	return noms
}
