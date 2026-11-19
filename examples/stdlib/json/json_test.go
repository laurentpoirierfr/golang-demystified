// Exemples du chapitre « JSON : encoding/json ».
package json

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Adresse struct {
	Ville string `json:"ville"`
	Pays  string `json:"pays,omitempty"`
}

type Utilisateur struct {
	ID         int       `json:"id"`
	Nom        string    `json:"nom"`
	Email      string    `json:"email,omitempty"`
	Admin      bool      `json:"admin,omitzero"`
	Creation   time.Time `json:"creation,omitzero"`
	Adresse    *Adresse  `json:"adresse,omitempty"`
	Tags       []string  `json:"tags"`
	MotDePasse string    `json:"-"`
}

func Example_marshal() {
	u := Utilisateur{
		ID:         1,
		Nom:        "Alice",
		Adresse:    &Adresse{Ville: "Lyon"},
		MotDePasse: "secret",
	}
	data, err := json.Marshal(u)
	fmt.Println(string(data), err)

	indente, _ := json.MarshalIndent(Adresse{Ville: "Nantes", Pays: "FR"}, "", "  ")
	fmt.Println(string(indente))
	// Output:
	// {"id":1,"nom":"Alice","adresse":{"ville":"Lyon"},"tags":null} <nil>
	// {
	//   "ville": "Nantes",
	//   "pays": "FR"
	// }
}

func Example_unmarshal() {
	entree := `{"id": 2, "nom": "Bob", "creation": "2026-08-12T10:00:00Z", "inconnu": true}`
	var u Utilisateur
	if err := json.Unmarshal([]byte(entree), &u); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(u.ID, u.Nom, u.Creation.Format(time.DateOnly))
	// Output: 2 Bob 2026-08-12
}

func Example_erreurs() {
	var u Utilisateur
	err := json.Unmarshal([]byte(`{"id": "trois"}`), &u)
	var typeErr *json.UnmarshalTypeError
	fmt.Println(errors.As(err, &typeErr), typeErr.Field)

	err = json.Unmarshal([]byte(`{"id": 1,`), &u)
	fmt.Println(err != nil)
	// Output:
	// true id
	// true
}

func Example_dynamique() {
	var v map[string]any
	json.Unmarshal([]byte(`{"n": 42, "ok": true, "liste": [1, "a"]}`), &v)
	fmt.Printf("%T %T %T\n", v["n"], v["ok"], v["liste"])
	// Output: float64 bool []interface {}
}

func Example_flux() {
	entree := strings.NewReader(`{"ville": "Lille"}
{"ville": "Brest"}
`)
	dec := json.NewDecoder(entree)
	for dec.More() {
		var a Adresse
		if err := dec.Decode(&a); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(a.Ville)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(Adresse{Ville: "Pau"})
	// Output:
	// Lille
	// Brest
	// {"ville":"Pau"}
}

type Statut int

const (
	Brouillon Statut = iota
	Publie
)

func (s Statut) MarshalText() ([]byte, error) {
	switch s {
	case Brouillon:
		return []byte("brouillon"), nil
	case Publie:
		return []byte("publié"), nil
	}
	return nil, fmt.Errorf("statut inconnu : %d", s)
}

func (s *Statut) UnmarshalText(b []byte) error {
	switch string(b) {
	case "brouillon":
		*s = Brouillon
	case "publié":
		*s = Publie
	default:
		return fmt.Errorf("statut inconnu : %q", b)
	}
	return nil
}

func Example_personnalise() {
	type Article struct {
		Titre  string `json:"titre"`
		Statut Statut `json:"statut"`
	}
	data, _ := json.Marshal(Article{"Go 1.27", Publie})
	fmt.Println(string(data))

	var a Article
	err := json.Unmarshal([]byte(`{"titre": "x", "statut": "archivé"}`), &a)
	fmt.Println(err)
	// Output:
	// {"titre":"Go 1.27","statut":"publié"}
	// statut inconnu : "archivé"
}

func Example_v2() {
	type Config struct {
		Hote  string   `json:"hote"`
		Ports []int    `json:"ports"`
		Tags  []string `json:"tags"`
	}
	data, _ := jsonv2.Marshal(Config{Hote: "localhost", Ports: []int{80, 443}},
		jsontext.WithIndent("  "))
	fmt.Println(string(data))

	var c Config
	err := jsonv2.Unmarshal([]byte(`{"hote": "a", "inconnu": 1}`), &c,
		jsonv2.RejectUnknownMembers(true))
	fmt.Println(err != nil)

	err = jsonv2.Unmarshal([]byte(`{"hote": "a", "hote": "b"}`), &c)
	fmt.Println(err != nil)
	// Output:
	// {
	//   "hote": "localhost",
	//   "ports": [
	//     80,
	//     443
	//   ],
	//   "tags": []
	// }
	// true
	// true
}
