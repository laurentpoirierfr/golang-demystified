// Package claude est un agent qui gère une liste de tâches avec Claude, par le
// SDK Go officiel d'Anthropic : réponses en flux, appels d'outils, et une
// boucle qui exécute les outils demandés jusqu'à la réponse finale.
package claude

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go/v2"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// Modele est le modèle utilisé par défaut.
const Modele = "claude-opus-5-5"

// Consigne est l'instruction système de l'assistant.
const Consigne = `Tu es l'assistant d'une liste de tâches. Utilise les outils pour consulter
et modifier la liste ; n'invente jamais une tâche ni un identifiant. Réponds en
français, en une ou deux phrases.`

// MaxTours limite les allers-retours avec les outils pour une même demande.
const MaxTours = 10

// Agent converse avec Claude, et garde l'historique de la conversation.
type Agent struct {
	client     anthropic.Client
	boite      []outils.Outil
	outils     []anthropic.BetaToolUnionParam
	historique []anthropic.BetaMessageParam

	Modele  string
	Observe func(outils.Appel) // facultatif : chaque appel d'outil
}

// Nouveau crée un agent. Le client lit sa clé dans ANTHROPIC_API_KEY (ou le
// profil de la commande ant).
func Nouveau(client anthropic.Client, boite []outils.Outil) *Agent {
	a := &Agent{client: client, boite: boite, Modele: Modele}
	for _, o := range boite {
		p := o.Proprietes()
		a.outils = append(a.outils, anthropic.BetaToolUnionParam{OfTool: &anthropic.BetaToolParam{
			Name:        o.Nom,
			Description: anthropic.String(o.Description),
			InputSchema: anthropic.BetaToolInputSchemaParam{
				Properties: p["properties"],
				Required:   requis(p),
			},
			// L'entrée de l'outil arrive en flux, au fil de sa génération.
			// L'API ne la valide plus : outils.Execute s'en charge.
			EagerInputStreaming: anthropic.Bool(true),
		}})
	}
	return a
}

func requis(schema map[string]any) []string {
	var r []string
	if l, ok := schema["required"].([]any); ok {
		for _, v := range l {
			r = append(r, fmt.Sprint(v))
		}
	}
	return r
}

// ErrRefus signale une demande déclinée par le modèle.
var ErrRefus = errors.New("demande déclinée par le modèle")

// Demande envoie une question, exécute les outils que Claude appelle, et
// renvoie la réponse finale. ecrit reçoit le texte au fil de sa génération.
func (a *Agent) Demande(ctx context.Context, question string, ecrit func(string)) (string, error) {
	a.historique = append(a.historique, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(question)))

	for range MaxTours {
		reponse, err := a.tour(ctx, ecrit)
		if err != nil {
			return "", err
		}
		// La réponse rejoint l'historique avant l'exécution des outils.
		a.historique = append(a.historique, reponse.ToParam())

		switch reponse.StopReason {
		case anthropic.BetaStopReasonToolUse:
			// Tous les résultats partent dans un seul message.
			var resultats []anthropic.BetaContentBlockParamUnion
			for _, bloc := range reponse.Content {
				appel, ok := bloc.AsAny().(anthropic.BetaToolUseBlock)
				if !ok {
					continue
				}
				r := outils.Appelle(ctx, a.boite, appel.Name, []byte(appel.JSON.Input.Raw()))
				if a.Observe != nil {
					a.Observe(r)
				}
				texte, erreur := r.Resultat, false
				if r.Err != nil {
					texte, erreur = r.Err.Error(), true // le modèle peut se corriger
				}
				resultats = append(resultats, anthropic.NewBetaToolResultBlock(appel.ID, texte, erreur))
			}
			a.historique = append(a.historique, anthropic.NewBetaUserMessage(resultats...))
		case anthropic.BetaStopReasonRefusal:
			return "", ErrRefus
		case anthropic.BetaStopReasonMaxTokens:
			return "", errors.New("réponse tronquée : limite de jetons atteinte")
		default:
			return texte(reponse), nil
		}
	}
	return "", fmt.Errorf("plus de %d allers-retours avec les outils", MaxTours)
}

// tour envoie l'historique et reçoit la réponse en flux.
func (a *Agent) tour(ctx context.Context, ecrit func(string)) (anthropic.BetaMessage, error) {
	flux := a.client.Beta.Messages.NewStreaming(ctx, anthropic.BetaMessageNewParams{
		Model:     a.Modele,
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: Consigne}},
		Messages:  a.historique,
		Tools:     a.outils,
		// Un assistant de liste de tâches n'a pas besoin de réfléchir
		// longtemps : un effort faible répond plus vite, pour moins cher.
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow},
		// Si une demande est déclinée par les classifieurs de sécurité,
		// l'API la confie à un modèle de repli, dans le même appel.
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	defer flux.Close()

	var message anthropic.BetaMessage
	for flux.Next() {
		evenement := flux.Current()
		if err := message.Accumulate(evenement); err != nil {
			return message, err
		}
		if delta, ok := evenement.AsAny().(anthropic.BetaRawContentBlockDeltaEvent); ok {
			if t, ok := delta.Delta.AsAny().(anthropic.BetaTextDelta); ok && ecrit != nil {
				ecrit(t.Text)
			}
		}
	}
	return message, flux.Err()
}

func texte(m anthropic.BetaMessage) string {
	var s string
	for _, bloc := range m.Content {
		if t, ok := bloc.AsAny().(anthropic.BetaTextBlock); ok {
			s += t.Text
		}
	}
	return s
}
