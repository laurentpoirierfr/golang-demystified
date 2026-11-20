---
title: "Temps réel : WebSocket et Server-Sent Events"
linkTitle: "11. Temps réel"
weight: 110
---

HTTP est fait de questions et de réponses : le client demande, le serveur
répond, puis se tait. Une discussion, un tableau de bord, une notification ou
la progression d'une tâche demandent l'inverse : que le **serveur parle le
premier**, dès qu'il a du nouveau. Deux techniques le permettent dans un
navigateur :

| | Server-Sent Events (SSE) | WebSocket |
|-|--------------------------|-----------|
| Sens | serveur → client | dans les deux sens |
| Protocole | HTTP ordinaire, réponse sans fin | connexion « reprise » après une poignée de main HTTP |
| Format | texte (`text/event-stream`) | texte ou binaire, messages libres |
| Reconnexion | automatique, avec reprise (`Last-Event-ID`) | à programmer soi-même |
| Proxys, HTTP/2, compression | comme toute requête HTTP | traitement particulier |
| En Go | bibliothèque standard | bibliothèque tierce |
| Cas typique | flux d'actualité, notifications, progression | discussion, jeu, édition collaborative |

Les deux se combinent bien : **SSE pour recevoir**, et un simple `POST` ou un
WebSocket pour envoyer. C'est ce que fait l'exemple de ce chapitre, des
salons de discussion :

- un **WebSocket** pour discuter ;
- un **flux SSE** pour suivre un salon, avec reprise après coupure ;
- un `POST` pour publier sans connexion permanente (depuis `curl`, un script) ;
- une page HTML qui utilise le flux pour lire et le WebSocket pour écrire.

Le code est dans [`examples/approfondissements/temps-reel`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/temps-reel) :

```
temps-reel/
├── pkg/sse/              écrire et lire un flux SSE (bibliothèque standard)
├── internal/salon/       le hub : abonnés, historique, diffusion
├── internal/web/         WebSocket, SSE, POST, et la page HTML (embarquée)
└── cmd/serveur/          le programme, et son arrêt propre
```

## Le hub

Au centre, un **hub** diffuse chaque message aux abonnés de son salon. Il
ignore le transport : un abonné peut être un WebSocket, un flux SSE ou un
test. C'est le même motif que le flux `Suit` du chapitre « gRPC et Connect »,
avec un historique en plus :

```go
// Abonne inscrit un abonné au salon, et renvoie les messages publiés après
// depuis (0 : aucun), qu'il a manqués. L'historique et l'inscription sont
// pris sous le même verrou : aucun message ne peut être manqué ni reçu deux
// fois.
func (h *Hub) Abonne(salon string, depuis int64) (*Abonnement, []Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var manques []Message
	if depuis > 0 {
		for _, m := range h.historique {
			if m.Salon == salon && m.ID > depuis {
				manques = append(manques, m)
			}
		}
	}
	a := &Abonnement{salon: salon, messages: make(chan Message, TailleTampon)}
	h.abonnes[a] = struct{}{}
	return a, manques
}
```

La diffusion ne bloque jamais. Un abonné dont le tampon est plein est
déconnecté. Il se reconnectera, et l'historique lui rendra ce qu'il a manqué :

```go
for a := range h.abonnes {
	if a.salon != salon {
		continue
	}
	select {
	case a.messages <- m:
	default:
		// Un abonné trop lent ne doit ralentir ni les autres, ni
		// l'auteur : il est déconnecté, et se reconnectera.
		h.retire(a)
	}
}
```

Chaque message reçoit un identifiant croissant. C'est lui qui permettra la
reprise après une coupure.

## Server-Sent Events, avec la bibliothèque standard

### Le format

Un flux SSE est une réponse HTTP qui ne se termine pas. Chaque événement est
fait de lignes `champ: valeur`, et se termine par une ligne vide :

```
event: bienvenue
retry: 2000
data: golang

id: 1
event: message
data: {"id":1,"salon":"golang","auteur":"alice","texte":"Bonjour depuis websocat",…}

: pouls
```

| Champ | Rôle |
|-------|------|
| `id` | identifiant, renvoyé par le navigateur à la reconnexion |
| `event` | type d'événement, écouté par `addEventListener` |
| `data` | contenu ; plusieurs lignes `data:` sont jointes par des sauts de ligne |
| `retry` | délai de reconnexion conseillé, en millisecondes |
| `:` | commentaire, ignoré : sert de signal de vie |

### Écrire le flux

```go
func Ouvre(w http.ResponseWriter) (*Flux, error) {
	rc := http.NewResponseController(w)
	// Un flux dure longtemps : on lève le délai d'écriture du serveur
	// (http.Server.WriteTimeout) pour cette réponse seulement.
	if err := rc.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return nil, err
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // demande à nginx de ne pas mettre en tampon
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		return nil, fmt.Errorf("sse : la réponse ne peut pas être vidée : %w", err)
	}
	return &Flux{w: w, rc: rc}, nil
}
```

Deux outils de `net/http` font tout le travail :

- **`Flush`** pousse immédiatement ce qui a été écrit. Sans lui, `net/http`
  garde les octets en tampon, et le client ne voit rien arriver ;
- **`http.ResponseController`** (Go 1.20) donne accès à `Flush` et aux délais
  de **cette** réponse, même à travers des middlewares qui enveloppent le
  `ResponseWriter`.

> [!IMPORTANT]
> Un serveur bien configuré a un `WriteTimeout` (chapitre « Une API web
> complète »). Il couperait le flux au bout de quelques secondes. Le flux le
> lève pour lui seul, avec `SetWriteDeadline(time.Time{})`, et le serveur
> garde sa protection pour toutes les autres réponses. Un test le vérifie avec
> un `WriteTimeout` de 300 ms. Sans cette ligne, il échoue : « flux terminé ».

L'envoi d'un événement protège le format contre l'injection :

```go
func (f *Flux) Envoie(e Evenement) error {
	if strings.ContainsAny(e.ID, "\r\n") || strings.ContainsAny(e.Type, "\r\n") {
		return ErrSautDeLigne
	}
	…
	// Une donnée sur plusieurs lignes devient plusieurs lignes « data: » :
	// un saut de ligne ne peut pas terminer l'événement prématurément.
	donnees := strings.ReplaceAll(e.Donnees, "\r\n", "\n")
	for ligne := range strings.SplitSeq(donnees, "\n") {
		b.WriteString("data: " + ligne + "\n")
	}
	b.WriteString("\n")
	return f.ecrit(b.String())
}
```

Un identifiant `"1\n\ndata: faux"` fabriquerait sinon un événement complet,
au contenu choisi par l'attaquant.

### Le handler : reprise et signal de vie

```go
func (s *Serveur) flux(w http.ResponseWriter, r *http.Request) {
	…
	// À la reconnexion, EventSource renvoie le dernier identifiant reçu.
	depuis, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)

	flux, err := sse.Ouvre(w)
	…
	abonnement, manques := s.hub.Abonne(nom, depuis)
	defer s.hub.Desabonne(abonnement)

	// Le premier événement indique le délai de reconnexion conseillé.
	flux.Envoie(sse.Evenement{Type: "bienvenue", Donnees: nom, Reessai: 2 * time.Second})
	for _, m := range manques {
		envoie(m)
	}

	pouls := time.NewTicker(s.opts.Pouls)
	defer pouls.Stop()
	for {
		select {
		case m, ok := <-abonnement.Messages():
			if !ok {
				return // trop lent : le client se reconnectera, et rattrapera
			}
			if err := envoie(m); err != nil {
				return
			}
		case <-pouls.C:
			if err := flux.Commentaire("pouls"); err != nil {
				return // client parti
			}
		case <-r.Context().Done():
			return
		case <-s.arret.Done():
			return
		}
	}
}
```

Le **pouls** (un commentaire toutes les 30 secondes) a deux rôles. Il empêche
les proxys et les répartiteurs de charge de couper une connexion qu'ils
croient inactive. Et il révèle au serveur un client disparu sans prévenir :
l'écriture échoue, et la goroutine se termine au lieu de rester bloquée.

La reprise, observée avec `curl` après deux messages :

```console
$ curl -N -H 'Last-Event-ID: 1' localhost:8080/salons/golang/flux
event: bienvenue
retry: 2000
data: golang

id: 2
event: message
data: {"id":2,"salon":"golang","auteur":"bob","texte":"Et moi par curl",…}
```

Le message 1, déjà reçu, n'est pas renvoyé ; le message 2, manqué, l'est.

### Lire un flux en Go : un itérateur

Côté client Go (un outil, un autre service, un test), `pkg/sse` lit un flux
comme le fait `EventSource`. Un itérateur (Go 1.23) s'y prête bien :

```go
func Lit(r io.Reader) iter.Seq2[Evenement, error]

for e, err := range sse.Lit(reponse.Body) {
	if err != nil {
		return err
	}
	fmt.Println(e.ID, e.Type, e.Donnees)
}
```

## WebSocket avec coder/websocket

La bibliothèque standard n'implémente pas WebSocket (`golang.org/x/net/websocket`
existe, mais sa documentation renvoie vers deux paquets plus complets et mieux
maintenus, présentés ci-dessous). Deux
bibliothèques dominent :

| Bibliothèque | Points forts |
|--------------|--------------|
| [`coder/websocket`](https://github.com/coder/websocket) (anciennement `nhooyr.io/websocket`) | API avec `context`, `wsjson`, écriture concurrente sûre, maintenue par Coder |
| [`gorilla/websocket`](https://github.com/gorilla/websocket) | la plus ancienne et la plus répandue, API plus bas niveau |

Ce chapitre utilise `coder/websocket`.

### Ouvrir la connexion

```go
// Accept vérifie l'en-tête Origin : une page d'un autre site ne peut pas
// ouvrir de WebSocket avec les cookies de l'utilisateur.
c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.opts.Origines})
if err != nil {
	return // Accept a déjà répondu (403 pour une origine refusée)
}
defer func() { _ = c.CloseNow() }()
c.SetReadLimit(TailleMaxMessage) // 4 Kio ; au-delà : fermeture, code 1009
```

> [!CAUTION]
> Un navigateur envoie les **cookies** avec la requête d'ouverture d'un
> WebSocket, quelle que soit la page qui l'ouvre. Sans contrôle de l'origine,
> n'importe quel site pourrait ouvrir un WebSocket vers votre application au
> nom de l'utilisateur connecté : c'est le *Cross-Site WebSocket Hijacking*.
> `Accept` n'autorise par défaut que l'hôte du serveur ; `OriginPatterns`
> ajoute les origines légitimes. N'utilisez pas `InsecureSkipVerify` pour
> « faire marcher » un environnement de développement.

### Lire et écrire en même temps

Une connexion WebSocket est bidirectionnelle : le client peut envoyer un
message pendant que le serveur lui en écrit un. On lit donc dans une
goroutine, et on écrit dans la boucle principale :

```go
ctx, termine := context.WithCancel(r.Context())
defer termine()
defer context.AfterFunc(s.arret, termine)() // arrêt du serveur

abonnement, _ := s.hub.Abonne(nom, 0)
defer s.hub.Desabonne(abonnement)

go func() {
	defer termine()
	for {
		var e entrant
		if err := wsjson.Read(r.Context(), c, &e); err != nil {
			return // client parti, message trop gros, JSON invalide…
		}
		if _, err := s.hub.Publie(nom, pseudo, e.Texte); err != nil {
			// L'écriture est sûre depuis plusieurs goroutines.
			_ = s.ecrit(ctx, c, sortant{Type: "erreur", Erreur: err.Error()})
		}
	}
}()

pouls := time.NewTicker(s.opts.Pouls)
defer pouls.Stop()
for {
	select {
	case m, ok := <-abonnement.Messages():
		if !ok {
			c.Close(websocket.StatusTryAgainLater, "client trop lent")
			return
		}
		if err := s.ecrit(ctx, c, sortant{Type: "message", Message: &m}); err != nil {
			return
		}
	case <-pouls.C:
		// Le ping vérifie que le client répond toujours : une connexion
		// coupée sans fermeture (Wi-Fi perdu) serait sinon gardée à vie.
		ping, annule := context.WithTimeout(ctx, s.opts.Pouls/2)
		err := c.Ping(ping)
		annule()
		if err != nil {
			return
		}
	case <-ctx.Done():
		if s.arret.Err() != nil {
			c.Close(websocket.StatusGoingAway, "arrêt du serveur")
		}
		return
	}
}
```

Chaque écriture a son propre délai (`ecrit` utilise
`context.WithTimeout(ctx, 5*time.Second)`) : un client qui ne lit plus ne doit
pas immobiliser la goroutine.

> [!WARNING]
> Avec `coder/websocket`, **annuler le contexte passé à `Read` ferme
> brutalement la connexion**. La première version de ce handler lisait avec
> `ctx`, annulé à l'arrêt du serveur : la connexion était coupée avant que la
> boucle principale n'envoie la trame de fermeture « going away », et le
> client recevait un simple `EOF`. Le test d'arrêt l'a révélé. La lecture
> utilise donc `r.Context()`, et se termine avec la connexion.

## Arrêter un serveur aux connexions longues

`http.Server.Shutdown` attend la fin des requêtes en cours. Or un flux SSE ne
se termine jamais de lui-même, et un WebSocket, une fois la connexion reprise
(*hijack*), échappe complètement au serveur : `Shutdown` ne l'attend pas, mais
ne le ferme pas non plus.

`RegisterOnShutdown` enregistre une fonction appelée au début de `Shutdown`.
Elle annule un contexte que les handlers surveillent :

```go
serveur := web.Nouveau(salon.Nouveau(), log, web.Options{Origines: origines})
srv := &http.Server{Handler: serveur.Routes(), …}
// Shutdown attend la fin des requêtes en cours… y compris les connexions
// longues, qui ne finissent jamais d'elles-mêmes. RegisterOnShutdown
// prévient les handlers, qui se terminent alors proprement.
srv.RegisterOnShutdown(serveur.Arrete)
```

Le test lance le vrai programme, ouvre un flux SSE et un WebSocket, puis
arrête le serveur :

```go
debut := time.Now()
arrete()
if err := <-fini; err != nil {
	t.Fatalf("arrêt : %v", err)
}
if duree := time.Since(debut); duree > time.Second {
	t.Errorf("arrêt en %v : les connexions longues ont bloqué Shutdown", duree)
}
// Le client WebSocket reçoit une fermeture en règle : « going away ».
_, _, err = ws.Read(t.Context())
if websocket.CloseStatus(err) != websocket.StatusGoingAway { … }
```

L'arrêt prend quelques millisecondes. En retirant `RegisterOnShutdown`, le
même test échoue au bout de **dix secondes** (« context deadline exceeded ») :
`Shutdown` a attendu le flux SSE jusqu'à la fin de son délai.

> [!NOTE]
> Et le `WriteTimeout` pour les WebSockets ? La documentation de `Hijack`
> prévient que la connexion reprise « peut » garder les délais du serveur. En
> pratique, `net/http` les efface au moment de la reprise. Comme ce n'est pas
> garanti, le test `TestDelaiEcriture` vérifie qu'un WebSocket survit à un
> `WriteTimeout` de 300 ms.

## La page HTML

La page est embarquée dans le binaire (`//go:embed statique`). Elle lit avec
`EventSource` et écrit avec `WebSocket` :

```js
// Lecture : un flux SSE. EventSource se reconnecte seul, et renvoie
// Last-Event-ID pour rattraper les messages manqués.
const flux = new EventSource(`/salons/${encodeURIComponent(salon)}/flux`);
flux.addEventListener("message", (e) => affiche(JSON.parse(e.data)));

// Écriture : un WebSocket.
const ws = new WebSocket(`${proto}://${location.host}/salons/${encodeURIComponent(salon)}/ws?pseudo=${encodeURIComponent(pseudo)}`);
ws.send(JSON.stringify({ texte: champ.value }));
```

Les messages viennent d'autres utilisateurs : ils sont affichés avec
`textContent`, **jamais** `innerHTML`.

```js
function affiche(m) {
  const li = document.createElement("li");
  const auteur = document.createElement("strong");
  auteur.textContent = m.auteur + " : ";
  li.append(auteur, document.createTextNode(m.texte));
  liste.append(li);
}
```

La page a été vérifiée dans un vrai navigateur, Chromium sans interface piloté
par [Puppeteer](https://pptr.dev). Le script saisit un message dans le
formulaire, publie par l'API un message piégé, puis lit la page :

```console
état : connecté en tant que carole
message : carole : Bonjour depuis Chromium
message : mallory : <img src=x onerror=alert(1)>
balises <img> dans la page : 0
erreurs : aucune
```

Le message piégé s'affiche comme du texte : aucune balise créée, aucune
alerte déclenchée.

## Essayer

```bash
go run ./cmd/serveur
```

Puis, dans un navigateur, `http://localhost:8080/?salon=golang&pseudo=alice`
et `http://localhost:8080/?salon=golang&pseudo=bob` côte à côte. En ligne de
commande :

```bash
# suivre le salon (-N : sans tampon)
curl -N localhost:8080/salons/golang/flux

# publier
curl -d '{"auteur":"bob","texte":"Et moi par curl"}' localhost:8080/salons/golang/messages

# discuter par WebSocket, avec websocat
websocat "ws://localhost:8080/salons/golang/ws?pseudo=alice"
{"texte":"Bonjour depuis websocat"}
```

## Les tests

| Test | Vérifie |
|------|---------|
| `pkg/sse` : `ExampleFlux`, `ExampleLit` | format écrit et lu, sur des exemples exécutables |
| `TestInjection` | un saut de ligne dans `id` ou `event` est refusé |
| `TestDiffusionParSalon`, `TestReprise` | diffusion par salon, rattrapage d'après l'identifiant |
| `TestAbonneTropLent` | un abonné lent est déconnecté, sans bloquer les autres |
| `TestDiscussion` | deux WebSockets et un flux SSE reçoivent le même message |
| `TestRepriseSSE` | `Last-Event-ID` renvoie les messages manqués, et eux seuls |
| `TestOrigineRefusee` | une origine étrangère reçoit un 403 |
| `TestMessageTropGros` | au-delà de 4 Kio, fermeture avec le code 1009 |
| `TestPouls` | le flux SSE émet ses signaux de vie |
| `TestDelaiEcriture` | SSE et WebSocket survivent au `WriteTimeout` du serveur |
| `TestArretAvecConnexionsOuvertes` | arrêt rapide, client WebSocket prévenu |

## Passer à plusieurs instances

Le hub vit dans la mémoire d'un processus. Avec plusieurs instances derrière
un répartiteur de charge, alice et bob peuvent être connectés à deux
instances différentes. Chaque instance doit alors relayer les messages par un
système commun : NATS, Redis, ou PostgreSQL (`LISTEN`/`NOTIFY`). Le hub reste
le même ; seule la publication passe par le courtier, comme au chapitre
« Architecture événementielle ».

## En résumé

- **SSE** pour pousser vers le client : HTTP ordinaire, reconnexion et reprise
  automatiques, bibliothèque standard seule (`Flush`, `ResponseController`).
- **WebSocket** pour dialoguer : une goroutine de lecture, une boucle
  d'écriture, des délais partout, un ping régulier.
- **Protégez** : contrôle de l'origine, taille maximale des messages,
  échappement des données à l'affichage, pas d'injection dans le flux SSE.
- **Ne bloquez jamais** la diffusion : un abonné lent est déconnecté, et
  l'historique lui permet de rattraper.
- **Arrêtez proprement** : `RegisterOnShutdown`, car `Shutdown` ne sait pas
  terminer les connexions longues.
