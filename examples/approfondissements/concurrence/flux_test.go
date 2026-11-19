package concurrence

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
)

// Requete est une ligne de journal d'accès, analysée.
type Requete struct {
	Chemin string
	Statut int
}

func analyse(ligne string) Requete {
	var r Requete
	fmt.Sscanf(ligne, "%s %d", &r.Chemin, &r.Statut)
	return r
}

// Example_goStreams enchaîne une analyse, un filtre et un regroupement par
// lots de 2, avec la bibliothèque go-streams.
func Example_goStreams() {
	entree := make(chan any)
	sortie := make(chan any)

	source := extension.NewChanSource(entree)
	analyseur := flow.NewMap(analyse, 1) // parallélisme 1 : l'ordre est conservé
	erreurs := flow.NewFilter(func(r Requete) bool { return r.Statut >= 500 }, 1)
	lots := flow.NewBatch[Requete](2, time.Hour) // lots de 2, ou toutes les heures

	go source.Via(analyseur).Via(erreurs).Via(lots).To(extension.NewChanSink(sortie))

	go func() {
		defer close(entree)
		for _, l := range []string{
			"/accueil 200", "/api/taches 500", "/api/taches 201",
			"/api/taches 503", "/connexion 500", "/accueil 200",
		} {
			entree <- l
		}
	}()

	for lot := range sortie {
		var chemins []string
		for _, r := range lot.([]Requete) { // la sortie est un chan any : assertion de type
			chemins = append(chemins, r.Chemin)
		}
		fmt.Println(strings.Join(chemins, ", "))
	}
	// Output:
	// /api/taches, /api/taches
	// /connexion
}

// TestFenetreTemporelle compte les requêtes par fenêtre d'une seconde. Avec
// synctest, le temps est virtuel : le test est instantané et déterministe.
func TestFenetreTemporelle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entree := make(chan any)
		sortie := make(chan any)
		fenetre := flow.NewTumblingWindow[Requete](time.Second)
		go extension.NewChanSource(entree).Via(fenetre).To(extension.NewChanSink(sortie))

		go func() {
			defer close(entree)
			for _, pause := range []time.Duration{0, 100, 200, 900, 50, 1000} {
				time.Sleep(pause * time.Millisecond)
				entree <- Requete{Chemin: "/", Statut: 200}
			}
		}()

		var comptes []int
		for f := range sortie {
			comptes = append(comptes, len(f.([]Requete)))
		}
		// Instants d'arrivée : 0, 100, 300, 1200, 1250, 2250 ms.
		if got := fmt.Sprint(comptes); got != "[3 2 1]" {
			t.Errorf("requêtes par seconde = %s, attendu [3 2 1]", got)
		}
	})
}
