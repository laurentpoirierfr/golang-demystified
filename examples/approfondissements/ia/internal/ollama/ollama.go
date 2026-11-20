// Package ollama est un agent qui gère une liste de tâches avec un modèle
// local, servi par Ollama (https://ollama.com). Il n'utilise que net/http :
// l'API d'Ollama est un simple POST, dont la réponse arrive en flux d'objets
// JSON, un par ligne (NDJSON).
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// MaxTours limite les allers-retours avec les outils pour une même demande.
const MaxTours = 10

// Consigne est l'instruction système de l'assistant.
const Consigne = `Tu es l'assistant d'une liste de tâches. Utilise les outils pour consulter
et modifier la liste ; n'invente jamais une tâche ni un identifiant. Réponds en
français, en une ou deux phrases.`

// Types de l'API /api/chat.
type (
	message struct {
		Role      string       `json:"role"`
		Content   string       `json:"content"`
		ToolCalls []appelOutil `json:"tool_calls,omitempty"`
		ToolName  string       `json:"tool_name,omitempty"`
	}
	appelOutil struct {
		Function struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"function"`
	}
	outil struct {
		Type     string `json:"type"` // « function »
		Function struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"function"`
	}
	requete struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
		Tools    []outil   `json:"tools,omitempty"`
		Stream   bool      `json:"stream"`
	}
	morceau struct {
		Message    message `json:"message"`
		Done       bool    `json:"done"`
		DoneReason string  `json:"done_reason"`
		Error      string  `json:"error"`
	}
)

// Agent converse avec un modèle local.
type Agent struct {
	url        string
	modele     string
	client     *http.Client
	boite      []outils.Outil
	outils     []outil
	historique []message

	Observe func(outils.Appel) // facultatif : chaque appel d'outil
}

// Nouveau crée un agent. url est l'adresse d'Ollama (http://localhost:11434),
// modele un modèle capable d'appeler des outils (qwen3, llama3.2…).
func Nouveau(url, modele string, boite []outils.Outil) *Agent {
	a := &Agent{url: strings.TrimSuffix(url, "/"), modele: modele, client: http.DefaultClient, boite: boite,
		historique: []message{{Role: "system", Content: Consigne}}}
	for _, o := range boite {
		var t outil
		t.Type = "function"
		t.Function.Name, t.Function.Description, t.Function.Parameters = o.Nom, o.Description, o.Proprietes()
		a.outils = append(a.outils, t)
	}
	return a
}

// Demande envoie une question, exécute les outils appelés par le modèle, et
// renvoie la réponse finale. ecrit reçoit le texte au fil de sa génération.
func (a *Agent) Demande(ctx context.Context, question string, ecrit func(string)) (string, error) {
	a.historique = append(a.historique, message{Role: "user", Content: question})
	for range MaxTours {
		reponse, err := a.tour(ctx, ecrit)
		if err != nil {
			return "", err
		}
		a.historique = append(a.historique, reponse)
		if len(reponse.ToolCalls) == 0 {
			return reponse.Content, nil
		}
		for _, appel := range reponse.ToolCalls {
			r := outils.Appelle(ctx, a.boite, appel.Function.Name, appel.Function.Arguments)
			if a.Observe != nil {
				a.Observe(r)
			}
			contenu := r.Resultat
			if r.Err != nil {
				contenu = "erreur : " + r.Err.Error() // le modèle peut se corriger
			}
			a.historique = append(a.historique, message{Role: "tool", Content: contenu, ToolName: appel.Function.Name})
		}
	}
	return "", fmt.Errorf("plus de %d allers-retours avec les outils", MaxTours)
}

// tour envoie l'historique et lit la réponse, objet JSON par objet JSON.
func (a *Agent) tour(ctx context.Context, ecrit func(string)) (message, error) {
	corps, err := json.Marshal(requete{Model: a.modele, Messages: a.historique, Tools: a.outils, Stream: true})
	if err != nil {
		return message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url+"/api/chat", bytes.NewReader(corps))
	if err != nil {
		return message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	rep, err := a.client.Do(req)
	if err != nil {
		return message{}, err
	}
	defer rep.Body.Close()
	if rep.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(rep.Body, 4096))
		return message{}, fmt.Errorf("ollama : statut %d : %s", rep.StatusCode, strings.TrimSpace(string(b)))
	}

	var complet message
	complet.Role = "assistant"
	lignes := bufio.NewScanner(rep.Body)
	lignes.Buffer(make([]byte, 64*1024), 1<<20)
	for lignes.Scan() {
		var m morceau
		if err := json.Unmarshal(lignes.Bytes(), &m); err != nil {
			return message{}, fmt.Errorf("ollama : réponse illisible : %w", err)
		}
		if m.Error != "" {
			return message{}, fmt.Errorf("ollama : %s", m.Error)
		}
		if m.Message.Content != "" && ecrit != nil {
			ecrit(m.Message.Content)
		}
		complet.Content += m.Message.Content
		complet.ToolCalls = append(complet.ToolCalls, m.Message.ToolCalls...)
		if m.Done {
			return complet, nil
		}
	}
	if err := lignes.Err(); err != nil {
		return message{}, err
	}
	return message{}, io.ErrUnexpectedEOF // flux interrompu avant « done »
}
