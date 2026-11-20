package json

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProduit(t *testing.T) {
	b, err := json.Marshal(Produit{Nom: "Gopher", Prix: 12.5, Interne: "X-42"})
	if err != nil || string(b) != `{"nom":"Gopher","prix":12.5}` {
		t.Errorf("%s, %v", b, err)
	}
	b, _ = json.Marshal(Produit{Nom: "Tasse", Prix: 8, Stock: 3, Tags: []string{"cuisine"}})
	if string(b) != `{"nom":"Tasse","prix":8,"stock":3,"tags":["cuisine"]}` {
		t.Errorf("%s", b)
	}
}

func TestLitConfig(t *testing.T) {
	c, err := LitConfig(strings.NewReader(`{"adresse": "localhost", "port": 8080}`))
	if err != nil || c != (Config{"localhost", 8080}) {
		t.Errorf("%+v, %v", c, err)
	}
	for _, texte := range []string{
		`{"adresse": "localhost", "prot": 8080}`, // faute de frappe
		`{"port": 8080}`,
		`{"adresse": "localhost", "port": 70000}`,
		`{"adresse": "localhost", "port": "8080"}`,
		`{"adresse":`,
	} {
		if _, err := LitConfig(strings.NewReader(texte)); err == nil {
			t.Errorf("%s : erreur attendue", texte)
		}
	}
}

func TestDuree(t *testing.T) {
	type tache struct {
		Nom   string `json:"nom"`
		Delai Duree  `json:"delai"`
	}
	b, err := json.Marshal(tache{"sauvegarde", Duree(90 * time.Minute)})
	if err != nil || string(b) != `{"nom":"sauvegarde","delai":"1h30m0s"}` {
		t.Errorf("%s, %v", b, err)
	}
	var lue tache
	if err := json.Unmarshal([]byte(`{"nom":"x","delai":"2m30s"}`), &lue); err != nil || time.Duration(lue.Delai) != 150*time.Second {
		t.Errorf("%v, %v", lue, err)
	}
	if err := json.Unmarshal([]byte(`{"delai": 5}`), &lue); err == nil {
		t.Error("un nombre doit être refusé")
	}
}
