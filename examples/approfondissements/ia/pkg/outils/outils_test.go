package outils_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

func ExampleBoite() {
	boite, _ := outils.Boite(&outils.Taches{})
	for _, o := range boite {
		schema, _ := json.Marshal(o.Schema)
		fmt.Printf("%s : %s\n", o.Nom, schema)
	}
	// Output:
	// ajouter_tache : {"type":"object","properties":{"titre":{"type":"string","description":"le titre de la tâche, court et précis"}},"required":["titre"],"additionalProperties":false}
	// lister_taches : {"type":"object","properties":{"toutes":{"type":"boolean","description":"vrai pour inclure les tâches terminées"}},"additionalProperties":false}
	// terminer_tache : {"type":"object","properties":{"id":{"type":"integer","description":"l'identifiant de la tâche, donné par lister_taches"}},"required":["id"],"additionalProperties":false}
}

func TestOutils(t *testing.T) {
	boite, err := outils.Boite(&outils.Taches{})
	if err != nil {
		t.Fatal(err)
	}
	appelle := func(nom, entree string) (string, error) {
		o, ok := outils.Trouve(boite, nom)
		if !ok {
			t.Fatalf("outil %s absent", nom)
		}
		return o.Execute(t.Context(), json.RawMessage(entree))
	}

	if r, err := appelle("ajouter_tache", `{"titre": "Écrire le chapitre"}`); err != nil || r != "tâche 1 ajoutée : Écrire le chapitre" {
		t.Errorf("ajout : %q %v", r, err)
	}
	appelle("ajouter_tache", `{"titre": "Relire"}`)
	if r, err := appelle("terminer_tache", `{"id": 1}`); err != nil || !strings.Contains(r, "terminée") {
		t.Errorf("fin : %q %v", r, err)
	}
	if r, _ := appelle("lister_taches", ``); r != "2. Relire (à faire)\n" {
		t.Errorf("liste : %q", r)
	}
	if r, _ := appelle("lister_taches", `{"toutes": true}`); !strings.Contains(r, "1. Écrire le chapitre (faite)") {
		t.Errorf("liste complète : %q", r)
	}
}

// Ce qu'envoie un modèle peut être faux : chaque entrée est validée.
func TestEntreesInvalides(t *testing.T) {
	boite, _ := outils.Boite(&outils.Taches{})
	tests := []struct{ nom, entree, erreur string }{
		{"ajouter_tache", `{}`, "schéma"},                            // champ obligatoire absent
		{"ajouter_tache", `{"titre": 42}`, "schéma"},                 // mauvais type
		{"ajouter_tache", `{"titre": "x", "admin": true}`, "schéma"}, // champ inconnu
		{"ajouter_tache", `{"titre": "  "}`, "1 à 200"},              // règle métier
		{"terminer_tache", `{"id": "1"}`, "schéma"},
		{"terminer_tache", `{"id": 99}`, "introuvable"},
		{"lister_taches", `{"toutes": tr`, "JSON invalide"}, // entrée tronquée
	}
	for _, tt := range tests {
		o, _ := outils.Trouve(boite, tt.nom)
		if _, err := o.Execute(t.Context(), json.RawMessage(tt.entree)); err == nil || !strings.Contains(err.Error(), tt.erreur) {
			t.Errorf("%s %s : err = %v, attendu %q", tt.nom, tt.entree, err, tt.erreur)
		}
	}
}
