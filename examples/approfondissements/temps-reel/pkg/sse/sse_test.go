package sse_test

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/pkg/sse"
)

func ExampleFlux() {
	rec := httptest.NewRecorder()
	flux, _ := sse.Ouvre(rec)
	flux.Envoie(sse.Evenement{ID: "1", Type: "message", Donnees: `{"texte":"bonjour"}`, Reessai: 3 * time.Second})
	flux.Commentaire("toujours là")
	flux.Envoie(sse.Evenement{ID: "2", Donnees: "deux\nlignes"})

	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Print(rec.Body.String())
	// Output:
	// text/event-stream
	// id: 1
	// event: message
	// retry: 3000
	// data: {"texte":"bonjour"}
	//
	// : toujours là
	//
	// id: 2
	// data: deux
	// data: lignes
}

func ExampleLit() {
	flux := "retry: 1500\n\n: commentaire ignoré\nid: 7\nevent: arrivee\ndata: alice\n\ndata: sans identifiant\ndata: ni type\n\n"
	for e, err := range sse.Lit(strings.NewReader(flux)) {
		if err != nil {
			panic(err)
		}
		fmt.Printf("id=%q type=%q données=%q\n", e.ID, e.Type, e.Donnees)
	}
	// Output:
	// id="7" type="arrivee" données="alice"
	// id="7" type="" données="sans identifiant\nni type"
}

func TestAllerRetour(t *testing.T) {
	rec := httptest.NewRecorder()
	flux, err := sse.Ouvre(rec)
	if err != nil {
		t.Fatal(err)
	}
	envoyes := []sse.Evenement{
		{ID: "1", Type: "message", Donnees: "simple"},
		{ID: "2", Donnees: "a\r\nb\nc"},
		{ID: "3", Donnees: ""},
	}
	for _, e := range envoyes {
		if err := flux.Envoie(e); err != nil {
			t.Fatal(err)
		}
	}
	var recus []sse.Evenement
	for e, err := range sse.Lit(rec.Body) {
		if err != nil {
			t.Fatal(err)
		}
		recus = append(recus, e)
	}
	attendus := []string{"simple", "a\nb\nc", ""}
	if len(recus) != len(attendus) {
		t.Fatalf("%d événements reçus", len(recus))
	}
	for i, e := range recus {
		if e.Donnees != attendus[i] || e.ID != envoyes[i].ID {
			t.Errorf("événement %d : %+v", i, e)
		}
	}
}

// Un identifiant venu de l'extérieur ne doit pas pouvoir fabriquer un faux
// événement dans le flux.
func TestInjection(t *testing.T) {
	flux, _ := sse.Ouvre(httptest.NewRecorder())
	for _, e := range []sse.Evenement{
		{ID: "1\n\ndata: faux"},
		{Type: "message\nevent: admin"},
	} {
		if err := flux.Envoie(e); !errors.Is(err, sse.ErrSautDeLigne) {
			t.Errorf("%+v : err = %v", e, err)
		}
	}
}
