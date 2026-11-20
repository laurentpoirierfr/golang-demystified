---
title: "Architecture événementielle : Watermill et NATS"
linkTitle: "12. Architecture événementielle"
weight: 120
---

Le chapitre « Concurrence en pratique » relie des goroutines par des canaux,
et sa section sur le traitement de flux enchaîne des étapes dans un même
programme. Une **architecture événementielle** applique la même idée entre
**services** : chacun publie ce qui lui arrive (« une commande a été
passée »), et les autres y réagissent, sans se connaître. Entre eux, un
**courtier de messages** joue le rôle des canaux. Il conserve les messages et
les livre, même si le destinataire est momentanément arrêté.

On y gagne du découplage (ajouter un service qui réagit aux commandes ne
modifie pas celui qui les reçoit) et de la résistance aux pannes. On y perd
la simplicité de l'appel direct : le traitement devient **asynchrone**, un
message peut arriver **deux fois**, et l'état du système est
**cohérent à terme**, pas immédiatement. Ce chapitre montre comment Go et
[Watermill](https://watermill.io) maîtrisent ces trois difficultés.

Le code est dans [`examples/approfondissements/evenements`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/evenements) :

```
evenements/
├── pkg/evenement/          le contrat : les événements échangés
├── internal/boutique/      la logique métier (la saga), sans infrastructure
├── internal/messagerie/    Watermill : sujets, routeur, middlewares, NATS
├── internal/memoire/       dépôt en mémoire (tests, démonstration)
├── internal/postgres/      dépôt PostgreSQL : outbox et inbox
└── cmd/boutique/           le programme : API HTTP et gestionnaires
```

## Les outils

[Watermill](https://github.com/ThreeDotsLabs/watermill) est une bibliothèque
Go pour les applications pilotées par les messages. Il offre une interface
unique, `message.Publisher` et `message.Subscriber`, au-dessus d'une vingtaine
de transports. Changer de courtier ne change pas le code métier.

| Transport | Module Watermill | Points forts |
|-----------|------------------|--------------|
| En mémoire (GoChannel) | intégré | tests, démonstrations, un seul processus |
| [NATS JetStream](https://nats.io) | `watermill-nats` | léger, écrit en Go, persistance et rejeu |
| [Apache Kafka](https://kafka.apache.org) | `watermill-kafka` | très gros volumes, rétention longue |
| [RabbitMQ](https://www.rabbitmq.com) | `watermill-amqp` | routage riche, files classiques |
| PostgreSQL, MySQL | `watermill-sql` | pas de courtier à exploiter ; base de l'outbox |
| Google Pub/Sub, AWS SNS/SQS | `watermill-googlecloud`, `watermill-aws` | services gérés |

Ce chapitre utilise **NATS JetStream**. NATS est lui-même écrit en Go, tient
dans un binaire de quelques mégaoctets, et sa couche JetStream conserve les
messages, suit la position de chaque consommateur et relivre ce qui n'a pas
été acquitté. C'est l'un des grands projets Go présentés au début du livre.

## Le domaine : une saga de commande

Une boutique reçoit des commandes. Réserver le stock, confirmer ou annuler,
prévenir le client : chaque étape réagit à un événement et en publie un
autre. C'est une **saga** chorégraphiée, sans chef d'orchestre :

```
POST /commandes ──▶ CommandePassee ──▶ réservation du stock
                                         ├─▶ StockReserve ─────▶ confirmation + notification
                                         └─▶ StockInsuffisant ─▶ annulation + notification
```

L'annulation est une **compensation** : il n'existe pas de transaction qui
couvre plusieurs services. Si une étape échoue, une autre étape défait ou
neutralise ce qui a été fait.

### Le contrat

Les événements sont des structures Go, encodées en JSON :

```go
// CommandePassee : un client vient de passer une commande. Elle attend la
// réservation du stock.
type CommandePassee struct {
	CommandeID string    `json:"commande_id"`
	Client     string    `json:"client"`
	Articles   []Article `json:"articles"`
	Le         time.Time `json:"le"`
}

// StockReserve : le stock de tous les articles de la commande est réservé.
type StockReserve struct {
	CommandeID string `json:"commande_id"`
}

// StockInsuffisant : un article manque, rien n'a été réservé.
type StockInsuffisant struct {
	CommandeID string `json:"commande_id"`
	Ref        string `json:"ref"` // l'article manquant
}
```

Un événement décrit un fait **passé** (« commande passée », pas « réserver le
stock ») : son émetteur ne sait pas qui le lira. C'est un contrat public. On
peut lui ajouter des champs, jamais en retirer ni en changer le sens, car des
consommateurs écrits pour l'ancienne version liront encore les nouveaux
messages.

### Les gestionnaires

Le composant `cqrs` de Watermill associe un gestionnaire typé à chaque
événement. Les génériques font le décodage :

```go
func (s *Service) Gestionnaires() []cqrs.EventHandler {
	return []cqrs.EventHandler{
		cqrs.NewEventHandler("stock.reserve", s.reserve),
		cqrs.NewEventHandler("commandes.confirme", s.confirme),
		cqrs.NewEventHandler("commandes.annule", s.annule),
		cqrs.NewEventHandler("notifications.confirmation", s.notifieConfirmation),
		cqrs.NewEventHandler("notifications.annulation", s.notifieAnnulation),
	}
}

func (s *Service) reserve(ctx context.Context, e *evenement.CommandePassee) error {
	return s.depot.Execute(ctx, cle(ctx, "stock.reserve"), func(ctx context.Context, u Unite) error {
		manquant, err := u.Reserve(ctx, e.Articles)
		if err != nil {
			return err
		}
		if manquant != "" {
			return u.Publie(ctx, &evenement.StockInsuffisant{CommandeID: e.CommandeID, Ref: manquant})
		}
		return u.Publie(ctx, &evenement.StockReserve{CommandeID: e.CommandeID})
	})
}
```

Le type de l'argument (`*evenement.CommandePassee`) détermine l'événement
écouté. Le nom du gestionnaire (`"stock.reserve"`) identifie son abonnement :
chaque gestionnaire reçoit **tous** les événements de son type, à son rythme.
Une erreur renvoyée provoque une nouvelle livraison du message ; `nil`
l'acquitte.

Le package `boutique` ignore tout du transport et du stockage. Il ne connaît
que deux interfaces, `Depot` et `Notifieur`, comme le cœur de l'architecture
hexagonale (chapitre « Architecture hexagonale »).

## Le routeur et ses middlewares

Le **routeur** de Watermill reçoit les messages et les passe aux
gestionnaires, à travers des middlewares, comme `net/http` :

```go
routeur, err := message.NewRouter(message.RouterConfig{CloseTimeout: 10 * time.Second}, wlog)
filePoison, err := middleware.PoisonQueue(poison, SujetPoison)
routeur.AddMiddleware(
	filePoison,
	middleware.Retry{
		MaxRetries:      3,
		InitialInterval: 50 * time.Millisecond,
		Multiplier:      2,
		Logger:          wlog,
	}.Middleware,
	middleware.Recoverer,
)
```

L'ordre compte : le premier enveloppe les suivants.

- **`Recoverer`** : une panique dans un gestionnaire devient une erreur ;
- **`Retry`** : une erreur provoque jusqu'à trois nouvelles tentatives, à 50,
  100 puis 200 ms. Une panne passagère (base surchargée, service de courriel
  indisponible) se résout souvent d'elle-même ;
- **`PoisonQueue`** : si toutes les tentatives échouent, le message part dans
  une **file poison** (`boutique-poison`), avec la raison de l'échec en
  métadonnée. Le traitement continue avec les messages suivants. Sans elle,
  un seul message défectueux serait relivré indéfiniment et bloquerait la
  file.

## Les deux pièges : double écriture et doublons

### La double écriture

Le service des commandes doit faire deux choses : enregistrer la commande en
base, et publier `CommandePassee`. Dans quel ordre ?

- enregistrer, puis publier : si le programme s'arrête entre les deux, ou si
  le courtier est indisponible, la commande existe mais personne ne le saura.
  Elle restera « en attente » pour toujours ;
- publier, puis enregistrer : si l'enregistrement échoue, le stock sera
  réservé pour une commande qui n'existe pas.

Il n'y a pas de bon ordre : ce sont deux systèmes, sans transaction commune.

### L'outbox

La solution est de n'écrire **qu'à un seul endroit** : la base. L'événement
est écrit dans une table, l'**outbox**, dans la **même transaction** que la
commande. Un **transitaire** (*forwarder*) lit ensuite cette table et envoie
les événements au courtier :

```
            une transaction
     ┌──────────────────────────┐
     │ INSERT INTO commandes    │      transitaire        NATS JetStream
     │ INSERT INTO outbox  ─────┼───▶  (lit l'outbox) ───▶ boutique-CommandePassee
     └──────────────────────────┘
```

Si la transaction échoue, aucun événement n'existe. Si elle réussit,
l'événement partira, quitte à attendre que le courtier revienne. Watermill
fournit les deux pièces : `watermill-sql` publie dans une table, au sein d'une
transaction, et `forwarder` enveloppe chaque message avec son sujet de
destination puis le transmet.

```go
func (d *Depot) Execute(ctx context.Context, cle string, f func(context.Context, boutique.Unite) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	…
	// Les événements sont publiés dans la table de l'outbox, au sein de la
	// transaction, enveloppés avec leur sujet de destination.
	dansTx, err := wsql.NewPublisher(wsql.TxFromStdSQL(tx), wsql.PublisherConfig{
		SchemaAdapter: wsql.DefaultPostgreSQLSchema{},
	}, d.log)
	…
	bus, err := messagerie.NouveauBus(forwarder.NewPublisher(dansTx, forwarder.PublisherConfig{ForwarderTopic: SujetOutbox}), d.log)
	…
	if err := f(ctx, &unite{tx: tx, bus: bus}); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}
```

Le transitaire tourne en tâche de fond :

```go
func (d *Depot) Transitaire(courtier message.Publisher) (*forwarder.Forwarder, error) {
	abonne, err := d.AbonneOutbox() // lit la table de l'outbox
	…
	return forwarder.NewForwarder(abonne, courtier, d.log, forwarder.Config{ForwarderTopic: SujetOutbox})
}
```

Le test d'intégration le vérifie sur un vrai PostgreSQL : une transaction qui
échoue après avoir publié ne laisse **ni commande, ni événement**, et une
transaction réussie laisse les deux.

### Les doublons et l'inbox

Un courtier fiable livre chaque message **au moins une fois**. Si le
consommateur traite un message, mais s'arrête avant de l'acquitter, le
message sera relivré. Livrer « exactement une fois » est impossible à travers
un réseau. On obtient le même **effet** avec des gestionnaires
**idempotents** : traiter deux fois le même message a le même effet que le
traiter une fois.

Chaque message a un identifiant unique, conservé lors des relivraisons. La
clé `gestionnaire/identifiant` est enregistrée dans une table, l'**inbox**,
dans la même transaction que les effets du traitement :

```go
// cle identifie le traitement d'un message par un gestionnaire : le même
// message, livré deux fois au même gestionnaire, a la même clé.
func cle(ctx context.Context, gestionnaire string) string {
	return gestionnaire + "/" + cqrs.OriginalMessageFromCtx(ctx).UUID
}
```

```go
if cle != "" {
	// Si deux livraisons du même message arrivent en même temps, la
	// seconde attend la fin de la première sur la clé primaire, puis ne
	// fait rien.
	res, err := tx.ExecContext(ctx, `INSERT INTO messages_traites (cle) VALUES ($1) ON CONFLICT DO NOTHING`, cle)
	…
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return errors.Join(err, tx.Rollback()) // déjà traité
	}
}
```

L'outbox garantit qu'un événement part **au moins une fois** ; l'inbox,
qu'il n'est traité qu'**une fois** par chaque gestionnaire. Ensemble, ils
donnent l'effet « exactement une fois ».

> [!NOTE]
> Une notification (courriel, SMS) est un effet de bord **externe** : elle ne
> peut pas entrer dans la transaction. On transmet alors la clé au fournisseur
> (`Notifie(ctx, cle, …)`). Les services d'envoi sérieux acceptent une clé
> d'idempotence et ignorent les doublons. À défaut, un doublon reste
> possible, mais rare.

### Tout ou rien, même dans le stock

La réservation de plusieurs articles doit, elle aussi, être atomique. Si le
deuxième article manque, la réservation du premier doit être annulée, **mais
pas** la transaction, qui doit encore publier `StockInsuffisant`. Un point de
sauvegarde le permet :

```go
func (u *unite) Reserve(ctx context.Context, articles []evenement.Article) (string, error) {
	if _, err := u.tx.ExecContext(ctx, `SAVEPOINT reservation`); err != nil {
		return "", err
	}
	for _, a := range articles {
		// La condition « quantite >= $2 » rend la vérification et la
		// décrémentation atomiques : pas de survente entre deux commandes
		// concurrentes.
		res, err := u.tx.ExecContext(ctx,
			`UPDATE stock SET quantite = quantite - $2 WHERE ref = $1 AND quantite >= $2`, a.Ref, a.Quantite)
		…
		} else if n == 0 {
			_, err := u.tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT reservation`)
			return a.Ref, err
		}
	}
	_, err := u.tx.ExecContext(ctx, `RELEASE SAVEPOINT reservation`)
	return "", err
}
```

## NATS JetStream

Avec NATS, chaque gestionnaire obtient un **consommateur durable** :
JetStream retient où il en est, et plusieurs instances du service se
partagent ses messages (groupe de consommateurs) :

```go
func AbonneNATS(url string, log watermill.LoggerAdapter) Abonne {
	return func(gestionnaire string) (message.Subscriber, error) {
		// « stock.reserve » → « stock-reserve » : le point est interdit dans
		// les noms de consommateurs.
		nom := strings.ReplaceAll(gestionnaire, ".", "-")
		return wnats.NewSubscriber(wnats.SubscriberConfig{
			URL:              url,
			QueueGroupPrefix: nom,
			SubscribersCount: 1,
			AckWaitTimeout:   30 * time.Second, // sans acquittement, le message est relivré
			CloseTimeout:     10 * time.Second,
			Unmarshaler:      &wnats.NATSMarshaler{},
			JetStream: wnats.JetStreamConfig{
				AutoProvision: true,
				DurablePrefix: nom,
			},
		}, log)
	}
}
```

> [!WARNING]
> Watermill utilise le **sujet** comme **nom de flux** JetStream, et NATS
> interdit le point dans les noms de flux et de consommateurs. Le sujet
> `boutique.CommandePassee`, naturel en NATS, échoue donc : ce chapitre
> utilise `boutique-CommandePassee`, et remplace les points des noms de
> gestionnaires.

## Les tests

### En mémoire

Le transport **GoChannel** de Watermill fonctionne dans le processus, sans
courtier. Les tests de la saga utilisent le même routeur, les mêmes
middlewares et les mêmes gestionnaires qu'en production, avec un dépôt en
mémoire :

```go
pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, wlog)
bus, _ := messagerie.NouveauBus(pubsub, wlog)
depot := memoire.Nouveau(bus, map[string]int{"clavier": 5, "souris": 10})
service := boutique.Nouveau(depot, notifieur, nouvelID)

routeur, _ := messagerie.NouveauRouteur(pubsub, log)
abonne := func(string) (message.Subscriber, error) { return pubsub, nil }
messagerie.AjouteGestionnaires(routeur, abonne, wlog, service.Gestionnaires()...)
go routeur.Run(ctx)
<-routeur.Running()
```

Le traitement est asynchrone : un test passe une commande, puis **attend**
que la saga aboutisse, avec un délai maximal :

```go
id, _ := b.service.Passe(t.Context(), "alice", []evenement.Article{{Ref: "clavier", Quantite: 2}})
// Juste après, la commande est en attente : la suite est asynchrone.
attend(t, "confirmation", func() bool { return b.statut(t, id) == boutique.Confirmee })
```

Chaque propriété de la saga a son test :

| Test | Vérifie |
|------|---------|
| `TestCommandeConfirmee` | stock réservé, commande confirmée, client prévenu |
| `TestCompensation` | article manquant : rien n'est réservé, commande annulée |
| `TestIdempotence` | chaque message livré **deux fois** : stock réservé une fois |
| `TestNouvellesTentatives` | deux pannes du courriel, puis succès à la troisième tentative |
| `TestFilePoison` | panne permanente : message dans la file poison, saga poursuivie |

Le test d'idempotence enveloppe le publieur pour **dupliquer chaque
message**, avec le même identifiant, comme un courtier après une coupure
réseau :

```go
type doublon struct{ message.Publisher }

func (d doublon) Publish(sujet string, msgs ...*message.Message) error {
	for _, m := range msgs {
		if err := d.Publisher.Publish(sujet, m, m.Copy()); err != nil {
			return err
		}
	}
	return nil
}
```

### Avec PostgreSQL et NATS

Les tests d'intégration démarrent un vrai PostgreSQL et un vrai NATS avec
[Testcontainers](https://golang.testcontainers.org), comme au chapitre
« sqlc et golang-migrate ». Ils vérifient l'outbox, l'inbox et la
réservation tout ou rien, puis font passer la saga complète par le vrai
programme :

```go
pg, err := tcpostgres.Run(ctx, "docker.io/postgres:17-alpine", …)
nats, err := tcnats.Run(ctx, "docker.io/nats:2.12-alpine") // JetStream activé (-js)
…
base, journal := demarre(t, map[string]string{"BOUTIQUE_BASE": dsn, "NATS_URL": url})
parcours(t, base, journal)
```

```bash
make test-integration
```

## Essayer, et casser

Sans configuration, la boutique tourne entièrement en mémoire :

```bash
go run ./cmd/boutique
```

Avec PostgreSQL et NATS :

```bash
docker run -d --name pg -p 5432:5432 -e POSTGRES_USER=gopher \
  -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=boutique postgres:17-alpine
docker run -d --name nats -p 4222:4222 nats:2.12-alpine -js -sd /data

BOUTIQUE_BASE='postgres://gopher:secret@localhost:5432/boutique?sslmode=disable' \
NATS_URL=nats://localhost:4222 go run ./cmd/boutique
```

Une commande est acceptée immédiatement, avec le statut `202 Accepted` :
elle est enregistrée, et son traitement suit.

```console
$ curl -i -d '{"client":"alice","articles":[{"ref":"clavier","quantite":2}]}' localhost:8080/commandes
HTTP/1.1 202 Accepted
Location: /commandes/01M7VYBGGR8PZWHSN2WT6RCT2Q
{"id":"01M7VYBGGR8PZWHSN2WT6RCT2Q","statut":"en_attente"}

$ curl localhost:8080/commandes/01M7VYBGGR8PZWHSN2WT6RCT2Q      # un instant plus tard
{"id":"01M7VYBGGR8PZWHSN2WT6RCT2Q","client":"alice","articles":[…],"statut":"confirmee"}

$ curl -d '{"client":"bob","articles":[{"ref":"souris","quantite":1},{"ref":"ecran","quantite":3}]}' localhost:8080/commandes
$ curl localhost:8080/commandes/01M7VYBJ144N6VP2A5PFHZKEC5
{…,"statut":"annulee","raison":"stock insuffisant : ecran"}
$ curl localhost:8080/stock/souris
{"souris":10}
```

L'intérêt de l'outbox se mesure dans la panne. Lors de la mise au point de
ce chapitre, on a **arrêté NATS**, passé une commande, puis **tué le
programme**. La commande restait « en attente » en base, son événement dans
l'outbox. On a ensuite redémarré NATS et le programme : la commande a été
confirmée et le client prévenu.

```console
$ docker stop nats
$ curl -d '{"client":"david","articles":[{"ref":"ecran","quantite":1}]}' localhost:8080/commandes
{"id":"01M7VYC5FG30YEQSE5487E0AK8","statut":"en_attente"}       ← acceptée malgré la panne
                                         … arrêt brutal du programme …
$ docker start nats
$ go run ./cmd/boutique
{"level":"INFO","msg":"notification","client":"david","message":"Votre commande 01M7VYC5FG30YEQSE5487E0AK8 est confirmée.",…}
```

Le premier essai n'arrêtait que NATS, sans tuer le programme. Il réussissait
aussi, mais il ne prouvait rien : le client NATS garde en mémoire les messages
envoyés pendant une coupure. Seul l'arrêt du programme prouve que
l'événement survit **dans la base**. Ce même essai a révélé un défaut : au
démarrage, le stock initial **écrasait** le stock en base, et chaque
redémarrage remettait l'inventaire à neuf. Il ne crée plus que les articles
absents (`ON CONFLICT DO NOTHING`).

L'outil `nats` (image `natsio/nats-box`) montre les flux et les consommateurs
créés par Watermill :

```console
$ nats stream ls -n
boutique-CommandePassee
boutique-StockInsuffisant
boutique-StockReserve
$ nats consumer ls boutique-StockReserve -n
commandes-confirme
notifications-confirmation
```

## En résumé

- Un **événement** décrit un fait passé ; c'est un contrat public, qu'on
  étend sans jamais le casser.
- **Watermill** sépare le code métier du transport : GoChannel pour les
  tests, NATS, Kafka ou RabbitMQ en production, sans changer les
  gestionnaires.
- Le **routeur** et ses middlewares (`Retry`, `PoisonQueue`, `Recoverer`)
  absorbent les pannes passagères et isolent les messages défectueux.
- L'**outbox** supprime la double écriture : les données et les événements
  partent ensemble, ou pas du tout.
- L'**inbox** rend les gestionnaires idempotents : un message livré deux fois
  n'est traité qu'une fois.
- Une **saga** remplace la transaction distribuée : chaque étape a sa
  compensation.
- **Testez la panne**, et assurez-vous qu'elle prouve ce que vous croyez :
  arrêter le courtier ne suffit pas, il faut aussi arrêter le programme.
