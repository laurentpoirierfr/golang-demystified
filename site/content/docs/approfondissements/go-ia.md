---
title: "Go et l'IA : modèles de langage, outils et MCP"
linkTitle: "16. Go et l'IA"
weight: 160
---

Les modèles de langage s'appellent par une API HTTP : une application Go n'a
besoin ni de Python, ni de GPU. Go apporte ce qu'il fait bien ailleurs : la
concurrence pour les réponses en flux, un typage qui décrit les outils, des
binaires simples à déployer, et une bibliothèque standard qui suffit à parler
à un modèle local.

Ce chapitre construit un **assistant de liste de tâches** : on lui écrit
« Ajoute deux tâches », il appelle les bons outils, et répond. Le même code
d'outils sert trois consommateurs :

```
                 ┌──▶ agent Claude   (SDK Go d'Anthropic, API en ligne)
pkg/outils  ─────┼──▶ agent Ollama   (net/http, modèle local)
(une fois)       └──▶ serveur MCP    (Claude Code, Claude Desktop, éditeurs…)
```

Le code est dans [`examples/approfondissements/ia`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/ia) :

```
ia/
├── pkg/outils/            les outils : schéma déduit des types, validation
├── internal/claude/       l'agent Claude : flux, outils, boucle
├── internal/ollama/       l'agent Ollama : la même boucle, en net/http
├── internal/mcpserveur/   les outils exposés par MCP
├── cmd/assistant/         l'assistant en ligne de commande
└── cmd/serveur-mcp/       le serveur MCP (stdio ou HTTP)
```

## Les outils : la seule partie métier

Un modèle de langage ne fait que produire du texte. Pour **agir**, il demande
l'appel d'un **outil** : un nom, et une entrée JSON conforme au schéma que
l'application a décrit. L'application exécute l'outil et renvoie le résultat ;
le modèle poursuit avec ce résultat.

Le schéma JSON de chaque outil est **déduit du type Go** de son entrée, avec
[`jsonschema-go`](https://github.com/google/jsonschema-go), la bibliothèque
qu'utilise aussi le SDK MCP :

```go
type entreeAjoute struct {
	Titre string `json:"titre" jsonschema:"le titre de la tâche, court et précis"`
}

ajoute, err := Nouveau("ajouter_tache", "Ajoute une tâche à la liste et renvoie son identifiant.",
	func(_ context.Context, in entreeAjoute) (string, error) {
		tache, err := t.Ajoute(in.Titre)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("tâche %d ajoutée : %s", tache.ID, tache.Titre), nil
	})
```

Le schéma produit, vérifié par un exemple testé :

```json
{"type":"object","properties":{"titre":{"type":"string","description":"le titre de la tâche, court et précis"}},"required":["titre"],"additionalProperties":false}
```

Un champ sans `omitempty` est obligatoire, et les champs inconnus sont
interdits. La fonction générique `Nouveau` relie le tout :

```go
func Nouveau[In any](nom, description string, f func(context.Context, In) (string, error)) (Outil, error) {
	schema, err := jsonschema.For[In](nil)
	…
	resolu, err := schema.Resolve(nil)
	…
}

// Execute valide l'entrée, puis exécute l'outil.
func (o Outil) Execute(ctx context.Context, entree json.RawMessage) (string, error) {
	…
	if err := o.resolu.Validate(valeur); err != nil {
		return "", fmt.Errorf("entrée non conforme au schéma : %w", err)
	}
	return o.execute(ctx, entree)
}
```

> [!IMPORTANT]
> **Ce qu'envoie un modèle n'est jamais présumé correct.** Un modèle peut
> oublier un champ, se tromper de type, inventer un identifiant, ou recevoir
> une entrée tronquée. Chaque entrée est validée contre le schéma, puis
> décodée en refusant les champs inconnus, puis soumise aux règles métier. Un
> test essaie chacun de ces cas : champ absent, mauvais type, champ en trop,
> titre vide, identifiant inconnu, JSON tronqué.

Une erreur d'outil n'arrête pas la conversation : elle est **renvoyée au
modèle comme résultat**, marqué en erreur, et le modèle peut se corriger.

## Appeler Claude avec le SDK Go

Anthropic publie un SDK Go officiel,
[`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) :

```bash
go get github.com/anthropics/anthropic-sdk-go/v2
```

Le client lit sa clé dans la variable `ANTHROPIC_API_KEY`, ou dans le profil
créé par la commande `ant auth login`. Une question simple :

```go
client := anthropic.NewClient()
reponse, err := client.Messages.New(ctx, anthropic.MessageNewParams{
	Model:     "claude-opus-5-5",
	MaxTokens: 16000,
	Messages: []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("Qu'est-ce qu'une goroutine ?")),
	},
})
```

### Les réponses en flux

Une réponse longue s'écrit mot à mot : l'utilisateur la voit arriver au lieu
d'attendre. `NewStreaming` renvoie un flux d'événements, et `Accumulate`
reconstitue le message complet au passage :

```go
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
```

Trois réglages méritent un mot :

- **l'effort** (`low` à `max`) règle la profondeur de réflexion du modèle, et
  donc la durée et le coût. Une conversation simple n'a pas besoin de plus que
  `low` ; un problème difficile mérite `high` ;
- **le repli** (`fallbacks`) : si les classifieurs de sécurité déclinent une
  demande, l'API la confie à un autre modèle dans le même appel, au lieu de
  renvoyer un simple refus ;
- **le flux est le choix par défaut** dès qu'une réponse peut être longue : un
  appel sans flux risque d'expirer avant la fin.

### La boucle des outils

La réponse de Claude s'arrête pour l'une de quatre raisons, et la boucle les
traite toutes :

```go
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
```

Quelques règles du protocole, que la boucle respecte :

- une réponse peut demander **plusieurs outils** à la fois ; tous les résultats
  repartent dans **un seul** message, sans quoi le modèle apprend à ne plus
  les appeler ensemble ;
- chaque résultat cite l'identifiant de l'appel (`tool_use_id`) ;
- la boucle est **bornée** : un modèle qui tournerait en rond ne doit pas
  consommer de jetons indéfiniment.

Les outils sont déclarés avec `EagerInputStreaming` : leur entrée arrive en
flux, au fil de sa génération, au lieu d'arriver d'un bloc à la fin. L'API ne
la valide alors plus elle-même, et c'est la validation de `pkg/outils` qui
s'en charge.

> [!TIP]
> Le SDK propose aussi un exécuteur d'outils (`client.Beta.Messages.NewToolRunner`)
> qui conduit cette boucle à votre place. La boucle écrite ici montre le
> protocole, et permet de la partager avec l'agent Ollama.

### Tester sans dépenser un centime

Un faux serveur, dans `httptest`, rejoue le protocole de l'API : des
événements SSE (`message_start`, `content_block_delta`, `input_json_delta`…)
écrits à l'avance. Le client du SDK y est dirigé par
`option.WithBaseURL` :

```go
client := anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("cle-de-test"), option.WithMaxRetries(0))
```

Les tests vérifient toute la boucle, à travers le vrai SDK : deux appels
d'outils dans une même réponse, une entrée qui arrive en morceaux
(`{"titre": "Ach` puis `eter du pain"}`), une entrée invalide renvoyée comme
erreur, un refus, une boucle sans fin interrompue, une erreur de l'API. Ils
inspectent aussi les requêtes envoyées : modèle, effort, repli, en-tête
`anthropic-beta`, et les deux résultats regroupés dans un seul message.

## Un modèle local avec Ollama

[Ollama](https://ollama.com) fait tourner des modèles ouverts (Qwen, Llama,
Mistral, Gemma…) sur la machine, sans réseau ni coût à l'appel. Il est écrit
en Go. Son API est un simple `POST` dont la réponse arrive en **NDJSON** : un
objet JSON par ligne. La bibliothèque standard suffit :

```go
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
…
return message{}, io.ErrUnexpectedEOF // flux interrompu avant « done »
```

La boucle des outils a la même forme qu'avec Claude ; seul le format change.
Les outils sont décrits au format `function`, et chaque résultat revient dans
un message de rôle `tool`, avec le nom de l'outil :

```go
a.historique = append(a.historique, message{Role: "tool", Content: contenu, ToolName: appel.Function.Name})
```

### Essayer

```bash
docker run -d --name ollama -p 11434:11434 ollama/ollama
docker exec ollama ollama pull qwen3:1.7b      # 1,4 Go, sait appeler des outils

go run ./cmd/assistant -fournisseur ollama -modele qwen3:1.7b
```

Une session réelle, sur un processeur ordinaire, sans carte graphique
(réponses du modèle abrégées). Les lignes `↳` montrent les appels d'outils :

```console
> Ajoute deux tâches : « Acheter du pain » et « Réviser les génériques ».
  ↳ ajouter_tache {"titre":"Acheter du pain"} → tâche 1 ajoutée : Acheter du pain
  ↳ ajouter_tache {"titre":"Réviser les génériques"} → tâche 2 ajoutée : Réviser les génériques
Les tâches ont été ajoutées avec succès : « Acheter du pain » et « Réviser les génériques ».
> Quelles sont mes tâches ?
  ↳ lister_taches {"toutes":true} → 1. Acheter du pain (à faire)
2. Réviser les génériques (à faire)
Vos tâches actuelles sont : 1. Acheter du pain (à faire), 2. Réviser les génériques (à faire)
> J'ai acheté le pain.
  ↳ terminer_tache {"id":1} → tâche 1 terminée : Acheter du pain
  ↳ terminer_tache {"id":1} → tâche 1 terminée : Acheter du pain
La tâche "Acheter du pain" est maintenant terminée.
> Que me reste-t-il à faire ?
Votre tâche restante est : 2. Réviser les génériques (à faire)
```

Ce petit modèle de 1,7 milliard de paramètres fait le travail, mais la session
montre aussi ses limites :

- il termine la tâche 1 **deux fois**. C'est sans conséquence, parce que
  l'opération est idempotente : une raison de plus de concevoir des outils qui
  le sont ;
- à la dernière question, il répond **de mémoire, sans appeler
  `lister_taches`**. La réponse est juste ici, mais rien ne garantit qu'elle le
  serait si la liste avait changé entre-temps.

Un modèle plus grand suit mieux les consignes. Le choix se fait selon le cas :

| | API en ligne (Claude) | Modèle local (Ollama) |
|-|-----------------------|-----------------------|
| Qualité, suivi des consignes | élevés | variables selon la taille du modèle |
| Données | envoyées au fournisseur | restent sur la machine |
| Coût | à l'usage (jetons) | matériel et électricité |
| Latence | réseau, mais matériel puissant | locale, limitée par le matériel |
| Mise en route | une clé d'API | télécharger un modèle de quelques Go |
| Cas typique | assistant, agent, tâches complexes | données sensibles, hors ligne, prototypage |

## Un serveur MCP en Go

Les deux agents précédents sont des **clients** : ils appellent un modèle. Le
[Model Context Protocol](https://modelcontextprotocol.io) (MCP) prend le
problème dans l'autre sens. Une application **expose** ses outils et ses
données, et n'importe quel assistant compatible (Claude Code, Claude Desktop,
les éditeurs de code…) peut s'en servir. Un serveur MCP écrit une fois sert
tous ces clients.

Le projet MCP publie un [SDK Go officiel](https://github.com/modelcontextprotocol/go-sdk).
Le serveur des tâches y enregistre chaque outil de la boîte, avec son schéma :

```go
func Nouveau(boite []outils.Outil, version string) *mcp.Server {
	serveur := mcp.NewServer(&mcp.Implementation{Name: "taches", Version: version}, nil)
	for _, o := range boite {
		serveur.AddTool(&mcp.Tool{Name: o.Nom, Description: o.Description, InputSchema: o.Schema},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				resultat, err := o.Execute(ctx, req.Params.Arguments)
				if err != nil {
					// Une erreur d'outil est un résultat, marqué comme tel : le
					// modèle la lit et peut se corriger. Une erreur Go
					// signalerait une panne du protocole.
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
					}, nil
				}
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: resultat}}}, nil
			})
	}
	return serveur
}
```

> [!TIP]
> Pour un outil isolé, `mcp.AddTool` (la fonction générique) déduit lui-même
> le schéma du type de l'entrée, et décode les arguments. Ici, les outils ont
> déjà leur schéma et leur validation : `Server.AddTool` les branche tels quels.

### Deux transports

Le programme parle **stdio** par défaut : l'assistant lance le serveur comme
sous-processus, et dialogue par son entrée et sa sortie standard. Avec
`-http`, il sert le transport **Streamable HTTP**, pour un serveur partagé :

```go
if adresse == "" {
	return serveur.Run(ctx, &mcp.StdioTransport{})
}
srv := &http.Server{
	Addr:              adresse,
	Handler:           mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return serveur }, nil),
	ReadHeaderTimeout: 5 * time.Second,
}
```

> [!WARNING]
> En mode stdio, la sortie standard **appartient au protocole**. Une seule
> ligne écrite par `fmt.Println` ou par un journal la corrompt, et le client
> se déconnecte. Le journal du serveur part donc sur la sortie d'erreur.

### Brancher le serveur à Claude Code

```bash
go build -o serveur-mcp ./cmd/serveur-mcp
claude mcp add taches -- /chemin/vers/serveur-mcp

# ou, pour un serveur lancé avec -http localhost:8090 :
claude mcp add --transport http taches http://localhost:8090
```

Claude Code liste alors les outils `ajouter_tache`, `lister_taches` et
`terminer_tache`, et les propose dans la conversation.

### Tester

Le SDK fournit une paire de transports reliés en mémoire : un client et le
serveur dialoguent sans réseau ni processus.

```go
coteServeur, coteClient := mcp.NewInMemoryTransports()
ss, err := serveur.Connect(t.Context(), coteServeur, nil)
…
client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
cs, err := client.Connect(t.Context(), coteClient, nil)
…
res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
	Name: "ajouter_tache", Arguments: map[string]any{"titre": "Préparer la démo MCP"},
})
```

Deux tests de bout en bout complètent l'ensemble. Le premier compile le vrai
programme et le lance comme le ferait Claude Code (`mcp.CommandTransport`).
Le second le sert en HTTP, et vérifie que deux sessions successives partagent
la même liste.

## Sécurité : un modèle n'est pas un utilisateur de confiance

Un assistant qui appelle des outils agit au nom de l'utilisateur, d'après le
texte qu'il lit, et ce texte peut avoir été écrit par quelqu'un d'autre : une
page web, un courriel, le titre d'une tâche. C'est l'**injection de
consignes** (*prompt injection*). Quelques règles, dans l'esprit du chapitre
« Sécurité » :

- **valider chaque entrée d'outil**, comme celle d'une API publique ;
- **le moindre privilège** : n'exposer que les outils nécessaires, et des
  outils étroits (« terminer la tâche n »), plutôt qu'un outil générique
  (« exécuter une requête SQL ») ;
- **confirmer les actions irréversibles** (suppression, paiement, envoi)
  auprès de l'utilisateur, en dehors du modèle ;
- **borner** la boucle, la taille des réponses et le coût ;
- **ne jamais placer de secret** dans la consigne ou dans les résultats
  d'outils : le modèle pourrait le répéter.

## En résumé

- Un modèle de langage s'appelle par HTTP : le SDK Go d'Anthropic pour
  Claude, `net/http` et NDJSON pour Ollama.
- **Le flux** est la norme pour les réponses longues ; `Accumulate`
  reconstitue le message.
- **Les outils** se décrivent par des types Go, dont le schéma JSON est
  déduit ; chaque entrée est validée, et une erreur est renvoyée au modèle
  pour qu'il se corrige.
- **La boucle** exécute les outils demandés, regroupe les résultats, gère
  le refus et la troncature, et reste bornée.
- **MCP** expose les mêmes outils à tous les assistants compatibles, sur
  stdio ou HTTP.
- **Les faux serveurs** (`httptest`, transports en mémoire) testent tout cela
  sans réseau ni coût ; un essai avec un vrai modèle montre ce que les tests
  ne voient pas.

> [!NOTE]
> L'agent Ollama a été essayé avec un vrai modèle local (la session
> ci-dessus). L'agent Claude est testé contre un faux serveur qui rejoue le
> protocole de l'API ; pour l'essayer avec le vrai service, définissez
> `ANTHROPIC_API_KEY` (ou lancez `ant auth login`), puis
> `go run ./cmd/assistant`.
