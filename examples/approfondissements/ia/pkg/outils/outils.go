// Package outils définit les outils qu'un modèle de langage peut appeler :
// ajouter, lister et terminer des tâches. Ils sont écrits une seule fois, et
// servis à trois consommateurs : l'agent Claude, l'agent Ollama et le serveur
// MCP.
//
// Le schéma JSON de chaque outil est déduit du type Go de son entrée, et
// chaque entrée est validée contre ce schéma avant l'exécution : ce qu'envoie
// un modèle n'est jamais présumé correct.
package outils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// Outil est un outil appelable par un modèle.
type Outil struct {
	Nom         string
	Description string
	Schema      *jsonschema.Schema // schéma JSON de l'entrée

	resolu  *jsonschema.Resolved
	execute func(ctx context.Context, entree json.RawMessage) (string, error)
}

// Execute valide l'entrée, puis exécute l'outil. Le résultat est un texte,
// destiné au modèle.
func (o Outil) Execute(ctx context.Context, entree json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(entree)) == 0 {
		entree = json.RawMessage("{}")
	}
	var valeur any
	if err := json.Unmarshal(entree, &valeur); err != nil {
		return "", fmt.Errorf("entrée JSON invalide : %w", err)
	}
	if err := o.resolu.Validate(valeur); err != nil {
		return "", fmt.Errorf("entrée non conforme au schéma : %w", err)
	}
	return o.execute(ctx, entree)
}

// Nouveau crée un outil dont l'entrée a le type In. Le schéma est déduit des
// champs de In : leur nom JSON, leur type, et leur description (étiquette
// jsonschema). Un champ sans « omitempty » est obligatoire.
func Nouveau[In any](nom, description string, f func(context.Context, In) (string, error)) (Outil, error) {
	schema, err := jsonschema.For[In](nil)
	if err != nil {
		return Outil{}, fmt.Errorf("outil %s : %w", nom, err)
	}
	resolu, err := schema.Resolve(nil)
	if err != nil {
		return Outil{}, fmt.Errorf("outil %s : %w", nom, err)
	}
	return Outil{
		Nom: nom, Description: description, Schema: schema, resolu: resolu,
		execute: func(ctx context.Context, entree json.RawMessage) (string, error) {
			var in In
			dec := json.NewDecoder(bytes.NewReader(entree))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&in); err != nil {
				return "", fmt.Errorf("entrée invalide : %w", err)
			}
			return f(ctx, in)
		},
	}, nil
}

// Proprietes renvoie le schéma sous forme de carte, pour les API qui
// l'attendent ainsi (Ollama, Claude).
func (o Outil) Proprietes() map[string]any {
	b, _ := json.Marshal(o.Schema)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// Trouve renvoie l'outil nommé.
func Trouve(boite []Outil, nom string) (Outil, bool) {
	for _, o := range boite {
		if o.Nom == nom {
			return o, true
		}
	}
	return Outil{}, false
}

// Appel décrit un appel d'outil par un modèle, pour l'afficher ou le
// journaliser.
type Appel struct {
	Outil    string
	Entree   json.RawMessage
	Resultat string
	Err      error
}

// Appelle exécute l'outil nommé. Une erreur est renvoyée au modèle comme
// résultat, pour qu'il puisse se corriger : elle ne termine pas la
// conversation.
func Appelle(ctx context.Context, boite []Outil, nom string, entree json.RawMessage) Appel {
	a := Appel{Outil: nom, Entree: entree}
	o, ok := Trouve(boite, nom)
	if !ok {
		a.Err = fmt.Errorf("outil inconnu : %s", nom)
		return a
	}
	a.Resultat, a.Err = o.Execute(ctx, entree)
	return a
}
