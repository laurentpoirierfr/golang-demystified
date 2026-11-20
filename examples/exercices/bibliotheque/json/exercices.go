// Package json contient les solutions des exercices du chapitre « JSON :
// encoding/json ».
package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Exercice 1 ★ : des tags.

// Produit s'encode avec des noms en minuscules ; le stock est omis s'il est
// nul, et le code interne n'est jamais encodé.
type Produit struct {
	Nom     string   `json:"nom"`
	Prix    float64  `json:"prix"`
	Stock   int      `json:"stock,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Interne string   `json:"-"`
}

// Exercice 2 ★★ : décoder strictement.

// Config est lue depuis un fichier JSON.
type Config struct {
	Adresse string `json:"adresse"`
	Port    int    `json:"port"`
}

// LitConfig refuse les champs inconnus (souvent une faute de frappe), et
// exige l'adresse et un port valide.
func LitConfig(r io.Reader) (Config, error) {
	var c Config
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("configuration : %w", err)
	}
	if c.Adresse == "" {
		return Config{}, errors.New("configuration : adresse manquante")
	}
	if c.Port < 1 || c.Port > 65535 {
		return Config{}, fmt.Errorf("configuration : port %d invalide", c.Port)
	}
	return c, nil
}

// Exercice 3 ★★★ : un encodage personnalisé.

// Duree s'encode en texte lisible, « 1h30m0s », plutôt qu'en nanosecondes.
type Duree time.Duration

// MarshalJSON satisfait json.Marshaler.
func (d Duree) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON satisfait json.Unmarshaler. Le récepteur est un pointeur :
// la méthode modifie la valeur.
func (d *Duree) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("durée : texte attendu : %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duree(v)
	return nil
}
