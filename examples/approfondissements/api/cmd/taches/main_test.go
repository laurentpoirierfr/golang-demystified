package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer protège le tampon du journal, écrit par plusieurs goroutines.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// TestDemarrageEtArret démarre le vrai programme sur un port libre, l'appelle,
// puis simule un SIGTERM en annulant son contexte.
func TestDemarrageEtArret(t *testing.T) {
	env := map[string]string{
		"TACHES_ADRESSE": "127.0.0.1:0", // port choisi par le système
		"TACHES_BASE":    ":memory:",
	}
	ctx, arrete := context.WithCancel(t.Context())
	var journal syncBuffer
	pret := make(chan string, 1)
	fin := make(chan error, 1)
	go func() { fin <- run(ctx, func(k string) string { return env[k] }, &journal, pret) }()

	var adresse string
	select {
	case adresse = <-pret:
	case err := <-fin:
		t.Fatalf("démarrage : %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post("http://"+adresse+"/taches", "application/json",
		strings.NewReader(`{"titre": "Arrêter proprement"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("statut %d", resp.StatusCode)
	}

	// Les métriques comptent la tâche créée.
	metriques, err := client.Get("http://" + adresse + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	corps, _ := io.ReadAll(metriques.Body)
	metriques.Body.Close()
	if !strings.Contains(string(corps), "taches_creees_total{") {
		t.Errorf("métrique taches_creees_total absente de /metrics")
	}

	arrete()
	if err := <-fin; err != nil {
		t.Fatalf("arrêt : %v", err)
	}
	for _, attendu := range []string{"serveur démarré", `"statut":201`, "serveur arrêté"} {
		if !strings.Contains(journal.String(), attendu) {
			t.Errorf("%q absent du journal :\n%s", attendu, journal.String())
		}
	}
}

func TestConfigurationInvalide(t *testing.T) {
	env := map[string]string{"TACHES_JOURNAL": "bavard"}
	err := run(t.Context(), func(k string) string { return env[k] }, &bytes.Buffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), "TACHES_JOURNAL") {
		t.Errorf("erreur = %v", err)
	}
}
