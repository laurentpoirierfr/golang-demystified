package client_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/pkg/client"
)

func ExampleNouveau() {
	simple, _ := client.Nouveau("https://api.exemple.org")
	fmt.Println(simple.Delai(), simple.Essais())

	regle, _ := client.Nouveau("https://api.exemple.org",
		client.AvecDelai(2*time.Second),
		client.AvecEssais(3),
	)
	fmt.Println(regle.Delai(), regle.Essais())

	_, err := client.Nouveau("https://api.exemple.org", client.AvecDelai(0), client.AvecEssais(42))
	fmt.Println(err)
	// Output:
	// 10s 1
	// 2s 3
	// délai invalide : 0s
	// nombre d'essais invalide : 42 (de 1 à 10)
}

func TestAdresseInvalide(t *testing.T) {
	for _, u := range []string{"", "api.exemple.org", "://x"} {
		if _, err := client.Nouveau(u); err == nil || !strings.Contains(err.Error(), "adresse invalide") {
			t.Errorf("%q : err = %v", u, err)
		}
	}
}

// Les options s'appliquent dans l'ordre : la dernière l'emporte.
func TestOrdre(t *testing.T) {
	c, err := client.Nouveau("https://x.org", client.AvecAgent("a"), client.AvecAgent("b"))
	if err != nil || c.Agent() != "b" {
		t.Errorf("agent %q, err %v", c.Agent(), err)
	}
}
