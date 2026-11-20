package valide_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/coulisses/pkg/valide"
)

type Adresse struct {
	Ville string `valide:"requis"`
}

type Tache struct {
	Titre      string   `valide:"requis,max=20"`
	Priorite   int      `valide:"min=1,max=5"`
	Etiquettes []string `valide:"max=3"`
	Adresse    Adresse
	note       string // non exporté : ignoré
}

func ExampleStructure() {
	err := valide.Structure(Tache{Titre: "", Priorite: 9, Etiquettes: []string{"a", "b", "c", "d"}})
	fmt.Println(err)
	// note, non exporté, est ignoré : la réflexion ne lit pas les champs privés.
	fmt.Println(valide.Structure(&Tache{Titre: "Écrire", Priorite: 2, Adresse: Adresse{Ville: "Lyon"}, note: "privée"}))
	// Output:
	// validation : Adresse.Ville est obligatoire ; Etiquettes doit valoir au plus 3 éléments ; Priorite doit valoir au plus 5 ; Titre est obligatoire
	// <nil>
}

func TestErreurs(t *testing.T) {
	err := valide.Structure(Tache{Titre: strings.Repeat("é", 21), Priorite: 0, Adresse: Adresse{Ville: "Nice"}})
	var e *valide.Erreur
	if !errors.As(err, &e) {
		t.Fatalf("err = %v", err)
	}
	if e.Champs["Titre"] != "doit valoir au plus 20 caractères" || e.Champs["Priorite"] != "doit valoir au moins 1" {
		t.Errorf("%#v", e.Champs)
	}
	if err := valide.Structure(42); err == nil || !strings.Contains(err.Error(), "structure attendue") {
		t.Errorf("err = %v", err)
	}
	if err := valide.Structure((*Tache)(nil)); err == nil {
		t.Error("un pointeur nil doit être refusé")
	}
	type Mauvaise struct {
		Prix string `valide:"min=abc"`
	}
	if err := valide.Structure(Mauvaise{}); err == nil || !strings.Contains(err.Error(), "règle invalide") {
		t.Errorf("err = %v", err)
	}
}

// Ce que coûte la réflexion : la même vérification, écrite à la main.
func verifieALaMain(t Tache) bool {
	n := 0
	for range t.Titre {
		n++
	}
	return t.Titre != "" && n <= 20 && t.Priorite >= 1 && t.Priorite <= 5 && len(t.Etiquettes) <= 3 && t.Adresse.Ville != ""
}

var (
	tache = Tache{Titre: "Écrire", Priorite: 2, Etiquettes: []string{"go"}, Adresse: Adresse{Ville: "Lyon"}}
	puits bool
)

func BenchmarkReflexion(b *testing.B) {
	for b.Loop() {
		puits = valide.Structure(tache) == nil
	}
}

func BenchmarkALaMain(b *testing.B) {
	for b.Loop() {
		puits = verifieALaMain(tache)
	}
}

// Les itérateurs de reflect (Go 1.26) décrivent un type sans instance.
func ExampleTypeFor() {
	for champ := range reflect.TypeFor[Tache]().Fields() {
		fmt.Printf("%-10s %-8s %q\n", champ.Name, champ.Type, champ.Tag.Get("valide"))
	}
	// Output:
	// Titre      string   "requis,max=20"
	// Priorite   int      "min=1,max=5"
	// Etiquettes []string "max=3"
	// Adresse    valide_test.Adresse ""
	// note       string   ""
}
