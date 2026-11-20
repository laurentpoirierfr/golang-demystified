package fuites_test

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/pkg/fuites"
)

func rapide(context.Context) string { return "rapide" }

func lente(ctx context.Context) string {
	select {
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
	}
	return "lente"
}

// fuitesDe demande au runtime les goroutines qui fuient : le profil
// goroutineleak lance un cycle de GC qui repère les goroutines bloquées sur
// un canal que plus aucune goroutine ne peut atteindre.
func fuitesDe(t *testing.T, fonction string) int {
	t.Helper()
	var b bytes.Buffer
	if err := pprof.Lookup("goroutineleak").WriteTo(&b, 1); err != nil {
		t.Fatal(err)
	}
	// Le profil liste des piles : une ligne « 3 @ 0x… » (trois goroutines
	// partagent cette pile), puis une ligne « # … » par fonction de la pile.
	n, pile := 0, 0
	for ligne := range strings.Lines(b.String()) {
		var k int
		if _, err := fmt.Sscanf(ligne, "%d @", &k); err == nil {
			pile = k
		} else if strings.HasPrefix(ligne, "#") && strings.Contains(ligne, fonction) {
			n += pile
			pile = 0 // une pile ne compte qu'une fois
		}
	}
	return n
}

func TestFuiteDetectee(t *testing.T) {
	// Une goroutine qui fuit ne disparaît jamais : on mesure l'écart, les
	// fuites des exécutions précédentes du test (-count) étant toujours là.
	avant := fuitesDe(t, "fuites.PremierFuyant")
	for range 10 {
		if r := fuites.PremierFuyant(t.Context(), rapide, lente, lente); r != "rapide" {
			t.Fatalf("r = %q", r)
		}
	}
	time.Sleep(50 * time.Millisecond) // les sources lentes ont répondu, et bloquent
	n := fuitesDe(t, "fuites.PremierFuyant") - avant
	t.Logf("goroutines qui fuient : %d", n)
	if n != 20 {
		t.Errorf("%d fuites détectées, attendu 20 (deux sources lentes × dix appels)", n)
	}
}

func TestSansFuite(t *testing.T) {
	for range 10 {
		if r := fuites.Premier(t.Context(), rapide, lente, lente); r != "rapide" {
			t.Fatalf("r = %q", r)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if n := fuitesDe(t, "fuites.Premier.func1"); n != 0 {
		t.Errorf("%d fuites détectées", n)
	}
}
