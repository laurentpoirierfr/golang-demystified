package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// demarre lance le vrai programme et renvoie son URL et son journal.
func demarre(t *testing.T, env map[string]string) (string, *journalTest) {
	t.Helper()
	env["BOUTIQUE_ADRESSE"] = "127.0.0.1:0"
	ctx, arrete := context.WithCancel(context.Background())
	sortie := &journalTest{}
	pret := make(chan string, 1)
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, func(k string) string { return env[k] }, sortie, pret) }()
	t.Cleanup(func() {
		arrete()
		if err := <-fini; err != nil {
			t.Errorf("arrêt : %v", err)
		}
	})
	select {
	case adresse := <-pret:
		return "http://" + adresse, sortie
	case err := <-fini:
		t.Fatalf("démarrage : %v", err)
		return "", nil
	}
}

// parcours passe deux commandes, l'une servie, l'autre non, et suit leur
// statut jusqu'à la fin de la saga.
func parcours(t *testing.T, base string, journal *journalTest) {
	t.Helper()
	servie := passe(t, base, `{"client": "alice", "articles": [{"ref": "clavier", "quantite": 2}]}`)
	refusee := passe(t, base, `{"client": "bob", "articles": [{"ref": "souris", "quantite": 1}, {"ref": "ecran", "quantite": 3}]}`)

	if c := attendStatut(t, base, servie, "confirmee"); c["client"] != "alice" {
		t.Errorf("commande servie : %v", c)
	}
	if c := attendStatut(t, base, refusee, "annulee"); c["raison"] != "stock insuffisant : ecran" {
		t.Errorf("commande refusée : %v", c)
	}
	if stock := lit(t, base+"/stock/clavier"); stock["clavier"] != float64(3) {
		t.Errorf("stock du clavier : %v", stock)
	}
	if stock := lit(t, base+"/stock/souris"); stock["souris"] != float64(10) {
		t.Errorf("la souris ne devrait pas être réservée : %v", stock)
	}
	attend(t, "notifications", func() bool {
		return journal.contient("Votre commande "+servie+" est confirmée.") &&
			journal.contient("Votre commande "+refusee+" est annulée : l'article ecran manque.")
	})
}

func TestDemonstration(t *testing.T) {
	base, journal := demarre(t, map[string]string{})
	parcours(t, base, journal)
	if !journal.contient("mode démonstration") {
		t.Error("le mode démonstration devrait être signalé")
	}
}

// TestPostgresNATS fait passer la saga par l'outbox PostgreSQL et par NATS
// JetStream, dans de vrais conteneurs.
func TestPostgresNATS(t *testing.T) {
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := t.Context()

	pg, err := tcpostgres.Run(ctx, "docker.io/postgres:17-alpine",
		tcpostgres.WithDatabase("boutique"),
		tcpostgres.WithUsername("gopher"),
		tcpostgres.WithPassword("secret"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, pg)
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	nats, err := tcnats.Run(ctx, "docker.io/nats:2.12-alpine") // JetStream activé (-js)
	testcontainers.CleanupContainer(t, nats)
	if err != nil {
		t.Fatal(err)
	}
	url, err := nats.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}

	base, journal := demarre(t, map[string]string{"BOUTIQUE_BASE": dsn, "NATS_URL": url})
	parcours(t, base, journal)
}

func passe(t *testing.T, base, corps string) string {
	t.Helper()
	rep, err := http.Post(base+"/commandes", "application/json", strings.NewReader(corps))
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Body.Close()
	var r map[string]string
	if err := json.NewDecoder(rep.Body).Decode(&r); err != nil || rep.StatusCode != http.StatusAccepted || r["statut"] != "en_attente" {
		t.Fatalf("commande : statut %d, %v, %v", rep.StatusCode, r, err)
	}
	return r["id"]
}

func lit(t *testing.T, url string) map[string]any {
	t.Helper()
	rep, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Body.Close()
	var v map[string]any
	json.NewDecoder(rep.Body).Decode(&v)
	return v
}

func attendStatut(t *testing.T, base, id, statut string) map[string]any {
	t.Helper()
	var c map[string]any
	attend(t, "commande "+id+" "+statut, func() bool {
		c = lit(t, base+"/commandes/"+id)
		return c["statut"] == statut
	})
	return c
}

func attend(t *testing.T, description string, condition func() bool) {
	t.Helper()
	limite := time.Now().Add(20 * time.Second)
	for !condition() {
		if time.Now().After(limite) {
			t.Fatalf("délai dépassé : %s", description)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// journalTest est une sortie sûre pour plusieurs goroutines.
type journalTest struct {
	mu sync.Mutex
	b  strings.Builder
}

func (j *journalTest) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.b.Write(p)
}

func (j *journalTest) contient(s string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return strings.Contains(j.b.String(), s)
}

var _ io.Writer = (*journalTest)(nil)
