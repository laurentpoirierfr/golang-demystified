---
title: "gRPC et Connect : des API à contrat"
linkTitle: "10. gRPC et Connect"
weight: 100
---

Le chapitre « Une API web complète » part d'un contrat OpenAPI pour décrire
une API REST. [gRPC](https://grpc.io) part d'un autre contrat : un fichier
[Protocol Buffers](https://protobuf.dev) (`.proto`), qui décrit les messages et
les méthodes d'un service. Le code du serveur et des clients en est généré,
dans tous les langages. C'est le protocole de prédilection entre
microservices : Kubernetes, etcd, Consul ou CockroachDB l'utilisent.

Ce chapitre utilise [Connect](https://connectrpc.com), une implémentation de
gRPC pour Go écrite par l'équipe de [Buf](https://buf.build). Un serveur
Connect parle **trois protocoles** sur la même adresse :

| Protocole | Transport | Pour qui |
|-----------|-----------|----------|
| **gRPC** | HTTP/2 | tous les clients gRPC existants, dans tous les langages |
| **gRPC-Web** | HTTP/1.1 ou HTTP/2 | les navigateurs, qui ne peuvent pas parler gRPC directement |
| **Connect** | HTTP/1.1 ou HTTP/2, JSON ou Protobuf | tout le reste : `curl`, un navigateur, un script |

Le protocole Connect est du HTTP ordinaire : un `POST` de JSON suffit pour
appeler une méthode. On garde les avantages de gRPC (le contrat, le code
généré, les flux) sans perdre la simplicité du web.

> [!NOTE]
> Connect pour Go est passé en **version 2** en octobre 2026
> (`connectrpc.com/connect/v2`). Les méthodes reçoivent et renvoient
> directement les messages, le cœur ne dépend plus de `net/http`, et les tests
> peuvent appeler le service en mémoire. Ce chapitre utilise la v2 ; le
> protocole, lui, n'a pas changé, et les clients v1 restent compatibles.

Le code est dans [`examples/approfondissements/grpc`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/grpc) :

```
grpc/
├── proto/taches/v1/taches.proto   le contrat, source de vérité
├── buf.yaml, buf.gen.yaml         configuration de buf
├── pkg/rpc/taches/v1/             code généré (messages, serveur, client)
├── internal/service/              l'implémentation et les intercepteurs
├── cmd/serveur/                   le programme serveur
└── cmd/client/                    un client en ligne de commande
```

## Le contrat

Le fichier `.proto` décrit un service de tâches avec les trois formes d'appel
les plus courantes :

```protobuf
syntax = "proto3";

package taches.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/…/grpc/pkg/rpc/taches/v1;tachesv1";

service TacheService {
  // Appel unaire : une requête, une réponse.
  rpc Cree(CreeRequest) returns (CreeResponse) {}

  // Sans effet de bord : appelable en GET avec le protocole Connect.
  rpc Trouve(TrouveRequest) returns (TrouveResponse) {
    option idempotency_level = NO_SIDE_EFFECTS;
  }

  // Flux serveur : une requête, une suite de réponses.
  rpc Suit(SuitRequest) returns (stream SuitResponse) {}

  // Flux client : une suite de requêtes, une réponse.
  rpc Importe(stream ImporteRequest) returns (ImporteResponse) {}
}

message Tache {
  int64 id = 1;
  string titre = 2;
  bool faite = 3;
  google.protobuf.Timestamp creation = 4;
}

message CreeRequest {
  string titre = 1;
}
// …
```

Quelques règles de Protocol Buffers :

- les **numéros** des champs (`= 1`, `= 2`) identifient les champs dans
  l'encodage binaire. On ne les change jamais et on ne les réutilise pas : c'est
  ce qui permet de faire évoluer un message sans casser les anciens clients ;
- chaque méthode a ses **propres messages** de requête et de réponse, même
  presque vides (`SuitRequest {}`) : on pourra leur ajouter des champs plus
  tard ;
- le **paquet versionné** (`taches.v1`) permet de publier un jour une `v2`
  incompatible à côté de la première ;
- un quatrième type d'appel, le flux **bidirectionnel** (`stream` des deux
  côtés), existe aussi. Il exige HTTP/2 de bout en bout.

## Générer le code avec buf

[buf](https://buf.build/docs) remplace `protoc` : il vérifie le contrat
(`buf lint`), détecte les changements incompatibles (`buf breaking`) et génère
le code (`buf generate`).

```yaml
# buf.yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go:v1.36.12
    out: pkg/rpc
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go:v2.0.2
    out: pkg/rpc
    opt: paths=source_relative
```

Les greffons `remote` sont exécutés par le registre de buf : il n'y a rien à
installer, sinon buf lui-même, que le `Makefile` lance dans un conteneur
(avec Docker ou Podman) :

```bash
make buf    # buf lint, puis buf generate
```

Deux fichiers sont générés :

- `taches.pb.go`, par `protoc-gen-go` : les messages (`Tache`, `CreeRequest`…),
  avec leurs accesseurs `GetTitre()`, sûrs même sur un message `nil` ;
- `tachesv1connect/taches.connect.go`, par `protoc-gen-connect-go` : l'interface
  du serveur, le client, et les noms des procédures.

```go
type TacheServiceHandler interface {
	Cree(context.Context, *v1.CreeRequest) (*v1.CreeResponse, error)
	Trouve(context.Context, *v1.TrouveRequest) (*v1.TrouveResponse, error)
	Suit(context.Context, *v1.SuitRequest, TacheServiceSuitServerStream) error
	Importe(context.Context, TacheServiceImporteServerStream) (*v1.ImporteResponse, error)
}
```

Comme pour oapi-codegen et sqlc, le code généré est versionné, et l'intégration
continue vérifie qu'il est à jour (`make generate`, puis `git diff`).

## Le service

L'implémentation est une structure ordinaire, qui satisfait l'interface
générée. Elle ignore tout du transport et du protocole :

```go
type Service struct {
	// Les méthodes non implémentées renvoient CodeUnimplemented : le code
	// compile toujours quand le contrat gagne une méthode.
	tachesv1connect.UnimplementedTacheServiceHandler

	mu      sync.Mutex
	taches  []*tachesv1.Tache
	abonnes map[chan *tachesv1.Tache]struct{}
	…
}

func (s *Service) Cree(_ context.Context, req *tachesv1.CreeRequest) (*tachesv1.CreeResponse, error) {
	t, err := s.ajoute(req.GetTitre())
	if err != nil {
		return nil, err
	}
	return &tachesv1.CreeResponse{Tache: t}, nil
}
```

### Les erreurs

gRPC définit une quinzaine de **codes d'erreur** (`NotFound`,
`InvalidArgument`, `Unauthenticated`…), communs à tous les langages, et que le
protocole Connect traduit en statuts HTTP (404, 400, 401…).

```go
return nil, connect.Errorf(connect.CodeNotFound, "tâche %d introuvable", req.GetId())
```

> [!IMPORTANT]
> En Connect v2, **seule une `*connect.Error` créée par le service envoie son
> message au client**. Une autre erreur, par exemple celle de la base de
> données, arrive chez le client sous la forme d'un code `Unknown` sans
> message : un détail interne ne fuit jamais par accident. Pour garder l'erreur
> d'origine côté serveur (journaux, `errors.Is`), on l'attache avec
> `connect.NewError(connect.CodeInternal, "stockage indisponible").WithCause(err)`.

Côté client, `connect.CodeOf(err)` donne le code, et `errors.AsType`
(Go 1.26) donne accès au message :

```go
if e, ok := errors.AsType[*connect.Error](err); ok {
	fmt.Println(e.Code(), e.Message()) // not_found tâche 42 introuvable
}
```

### Un flux serveur : Suit

`Suit` envoie les tâches existantes, puis chaque nouvelle tâche, jusqu'à ce
que le client se déconnecte. C'est la concurrence du chapitre « Concurrence en
pratique » appliquée à un service réseau :

```go
func (s *Service) Suit(ctx context.Context, _ *tachesv1.SuitRequest, flux tachesv1connect.TacheServiceSuitServerStream) error {
	// La copie des tâches existantes et l'abonnement se font sous le même
	// verrou : aucune tâche ne peut être manquée ni envoyée deux fois.
	s.mu.Lock()
	existantes := append([]*tachesv1.Tache(nil), s.taches...)
	nouvelles := make(chan *tachesv1.Tache, tailleTampon)
	s.abonnes[nouvelles] = struct{}{}
	s.mu.Unlock()
	defer s.desabonne(nouvelles)

	for _, t := range existantes {
		if err := flux.Send(&tachesv1.SuitResponse{Tache: t}); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done(): // le client s'est déconnecté
			return ctx.Err()
		case t, ok := <-nouvelles:
			if !ok {
				return connect.NewError(connect.CodeResourceExhausted, "abonné trop lent, déconnecté")
			}
			if err := flux.Send(&tachesv1.SuitResponse{Tache: t}); err != nil {
				return err
			}
		}
	}
}
```

Le contexte de l'appel est annulé quand le client se déconnecte : c'est le
signal d'arrêt de la boucle. Et un abonné trop lent ne doit pas ralentir les
autres. La création diffuse donc sans jamais bloquer, et déconnecte l'abonné
dont le tampon est plein :

```go
for abonne := range s.abonnes {
	select {
	case abonne <- t:
	default:
		close(abonne)
		delete(s.abonnes, abonne)
	}
}
```

### Un flux client : Importe

`Importe` lit les messages jusqu'à `io.EOF`, qui signale que le client a fini
d'envoyer :

```go
func (s *Service) Importe(_ context.Context, flux tachesv1connect.TacheServiceImporteServerStream) (*tachesv1.ImporteResponse, error) {
	var titres []string
	for {
		req, err := flux.Receive()
		if errors.Is(err, io.EOF) { // le client a fini d'envoyer
			break
		}
		if err != nil {
			return nil, err
		}
		titre, err := nettoie(req.GetTitre())
		if err != nil {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "titre n° %d : le titre est obligatoire", len(titres)+1)
		}
		titres = append(titres, titre)
	}
	for _, titre := range titres {
		if _, err := s.ajoute(titre); err != nil {
			return nil, err
		}
	}
	return &tachesv1.ImporteResponse{Nombre: int64(len(titres))}, nil
}
```

> [!TIP]
> La première version de ce code créait chaque tâche dès sa réception. Un
> essai avec le vrai client l'a montré : un import interrompu par un titre
> invalide laissait derrière lui les tâches déjà créées. Valider **tout** le
> flux avant de créer quoi que ce soit rend l'import atomique, et un test le
> vérifie désormais.

## Les intercepteurs

Un intercepteur est l'équivalent d'un middleware HTTP, pour les appels. En
Connect v2, il **entoure l'appel entier**, flux compris, et intervient avant
la lecture du message :

```go
func Journal(log *slog.Logger) connect.ServerInterceptor {
	return func(suivant connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, flux connect.ServerStream) error {
			debut := time.Now()
			err := suivant(ctx, spec, flux)
			info, _ := connect.CallInfoForServerContext(ctx)
			…
			log.Log(ctx, niveau, "appel", "procedure", spec.Procedure,
				"protocole", info.Protocol, "code", codeTexte(err), "duree", time.Since(debut))
			return err
		}
	}
}
```

L'authentification se lit dans les en-têtes, sans même décoder le message :
un client non authentifié ne coûte presque rien au serveur.

```go
func Authentification(jeton string) connect.ServerInterceptor {
	attendu := []byte("Bearer " + jeton)
	return func(suivant connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, flux connect.ServerStream) error {
			info, _ := connect.CallInfoForServerContext(ctx)
			recu := []byte(info.RequestHeader().Get("Authorization"))
			// Comparaison en temps constant : la durée ne révèle rien du jeton.
			if subtle.ConstantTimeCompare(recu, attendu) != 1 {
				return connect.NewError(connect.CodeUnauthenticated, "jeton absent ou invalide")
			}
			return suivant(ctx, spec, flux)
		}
	}
}
```

Côté client, un intercepteur symétrique ajoute l'en-tête à chaque appel
(`service.Jeton`). Un troisième, `Recuperation`, transforme une panique en
erreur `Internal` sans divulguer son message.

> [!TIP]
> L'écosystème fournit des intercepteurs prêts à l'emploi, eux aussi passés en
> v2 : [`otelconnect`](https://github.com/connectrpc/otelconnect-go) pour les
> traces et métriques OpenTelemetry (voir le chapitre « Observabilité »),
> [`validate`](https://github.com/connectrpc/validate-go) pour valider les
> messages d'après des règles écrites dans le `.proto`
> ([protovalidate](https://protovalidate.com)), et
> [`authn`](https://github.com/connectrpc/authn-go) pour l'authentification.

## Le serveur

Le programme assemble le tout. Les intercepteurs sont passés à
`connect.NewServer`, et `connecthttp.Mount` installe une route par méthode sur
un `http.ServeMux` ordinaire, à côté d'autres routes :

```go
serveur := connect.NewServer(service.Recuperation(log), service.Journal(log))
service.Nouveau().Enregistre(serveur)
// La réflexion décrit les services aux outils (grpcurl, buf curl,
// Postman) : ils peuvent les appeler sans le fichier .proto.
grpcreflect.Register(serveur)

mux := http.NewServeMux()
connecthttp.Mount(mux, serveur) // une route par méthode
mux.HandleFunc("GET /sante", …)

// gRPC exige HTTP/2. Sans TLS, c'est « h2c » (HTTP/2 en clair), que
// net/http sait servir depuis Go 1.24, sans dépendance.
protocoles := new(http.Protocols)
protocoles.SetHTTP1(true)
protocoles.SetUnencryptedHTTP2(true)
srv := &http.Server{
	Handler:           mux,
	Protocols:         protocoles,
	ReadHeaderTimeout: 5 * time.Second,
	// Pas de WriteTimeout : il couperait les flux de longue durée (Suit).
	IdleTimeout: 2 * time.Minute,
}
```

> [!WARNING]
> Un `WriteTimeout` fixe la durée maximale d'une **réponse entière**. Pour un
> flux comme `Suit`, qui peut durer des heures, il couperait la connexion au
> bout du délai. On le supprime, et on protège plutôt chaque appel unaire par
> un délai côté client (`context.WithTimeout`), que gRPC transmet au serveur
> (en-tête `grpc-timeout`).

## Les clients

Le client généré est le même pour les trois protocoles : seule l'option du
transport change.

```go
var options []connecthttp.Option
switch protocole {
case "grpc":
	options = append(options, connecthttp.WithGRPC())
case "grpcweb":
	options = append(options, connecthttp.WithGRPCWeb())
}
transport := connecthttp.NewTransport(clientHTTP(), adresse, options...)
c := tachesv1connect.NewTacheServiceClient(connect.NewClient(transport))

rep, err := c.Cree(ctx, &tachesv1.CreeRequest{Titre: "Apprendre gRPC"})
```

Le client HTTP doit lui aussi savoir parler HTTP/2 en clair :

```go
func clientHTTP() *http.Client {
	protocoles := new(http.Protocols)
	protocoles.SetHTTP2(true)
	protocoles.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: &http.Transport{Protocols: protocoles}}
}
```

Un flux serveur se lit jusqu'à `io.EOF` ; un flux client s'écrit, puis se
ferme avec `CloseAndReceive`, qui renvoie la réponse :

```go
flux, err := c.Suit(ctx, &tachesv1.SuitRequest{})
…
defer flux.Close()
for {
	rep, err := flux.Receive()
	if errors.Is(err, io.EOF) {
		break
	}
	…
}
```

Le programme `cmd/client` en fait une petite ligne de commande. Une session
réelle, avec `suit` lancé dans un autre terminal :

```console
$ client cree "Apprendre Connect"
1	Apprendre Connect	2026-11-19 19:57
$ client -protocole grpc cree "Apprendre gRPC"
2	Apprendre gRPC	2026-11-19 19:57
$ client -protocole grpcweb trouve 2
2	Apprendre gRPC	2026-11-19 19:57
$ client trouve 42
erreur : not_found: tâche 42 introuvable
$ printf 'Lire\nÉcrire\n' | client -protocole grpc importe
2 tâches importées

$ client suit          # dans un autre terminal, au fil de l'eau
1	Apprendre Connect	2026-11-19 19:57
2	Apprendre gRPC	2026-11-19 19:57
3	Lire	2026-11-19 19:57
4	Écrire	2026-11-19 19:57
```

### Sans client généré : curl et grpcurl

Le protocole Connect, c'est du HTTP : `curl` suffit. Une méthode sans effet de
bord s'appelle même en `GET`, et sa réponse peut être mise en cache :

```console
$ curl -H 'Content-Type: application/json' -d '{"titre":"via curl"}' \
    http://localhost:8090/taches.v1.TacheService/Cree
{"tache":{"id":"2", "titre":"via curl", "creation":"2026-11-19T19:58:03.215640161Z"}}

$ curl 'http://localhost:8090/taches.v1.TacheService/Trouve?encoding=json&message=%7B%22id%22%3A1%7D'
{"tache":{"id":"1", "titre":"Apprendre Connect", "creation":"2026-11-19T19:58:03.202043533Z"}}
```

En JSON, un `int64` est encodé en **chaîne** (`"id":"2"`), car JavaScript ne
représente pas exactement les entiers au-delà de 2⁵³.

Grâce à la réflexion, [grpcurl](https://github.com/fullstorydev/grpcurl)
découvre les services sans le fichier `.proto` :

```console
$ grpcurl -plaintext localhost:8090 list
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
taches.v1.TacheService

$ grpcurl -plaintext -d '{"id": 1}' localhost:8090 taches.v1.TacheService/Trouve
{
  "tache": {
    "id": "1",
    "titre": "Apprendre Connect",
    "creation": "2026-11-19T19:58:03.202043533Z"
  }
}
```

Le journal du serveur montre le protocole de chaque appel :

```json
{"level":"INFO","msg":"appel","procedure":"/taches.v1.TacheService/Cree","protocole":"connect","code":"ok"}
{"level":"WARN","msg":"appel","procedure":"/taches.v1.TacheService/Trouve","protocole":"connect","code":"not_found"}
{"level":"INFO","msg":"appel","procedure":"/taches.v1.TacheService/Trouve","protocole":"grpc","code":"ok"}
```

## Les tests

### En mémoire, sans réseau

`connectinprocess`, nouveau en v2, relie un client au serveur dans le même
processus : ni port, ni `httptest.Server`. Les tests du service restent
rapides et simples :

```go
func client(t *testing.T, intercepteurs ...connect.ServerInterceptor) (tachesv1connect.TacheServiceClient, *Service) {
	s := Nouveau()
	serveur := connect.NewServer(intercepteurs...)
	s.Enregistre(serveur)
	return tachesv1connect.NewTacheServiceClient(connect.NewClient(connectinprocess.New(serveur))), s
}
```

Les flux se testent avec `testing/synctest` (Go 1.25) : `synctest.Wait`
attend que toutes les goroutines soient bloquées, ce qui remplace les
`time.Sleep` fragiles :

```go
synctest.Test(t, func(t *testing.T) {
	c, _ := client(t)
	…
	if titre := <-recu; titre != "Avant" { … }
	synctest.Wait() // l'abonnement est en place
	c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "Après"})
	if titre := <-recu; titre != "Après" { … }
})
```

> [!NOTE]
> Avec `connectinprocess`, le service ne démarre qu'au **premier `Receive`**
> du client. Le test de l'abonné trop lent s'est d'abord bloqué pour cette
> raison : il saturait le tampon avant que le service ne soit abonné. Il lit
> désormais une première tâche avant de saturer le tampon. `synctest` a
> signalé le blocage immédiatement (« deadlock: all goroutines in bubble are
> blocked »), au lieu d'un test qui aurait expiré au bout de dix minutes.

### De bout en bout, sur les trois protocoles

Le test du programme lance le vrai serveur sur un port libre, et y fait passer
le même client sur les trois protocoles :

```go
protocoles := map[string][]connecthttp.Option{
	"connect": nil,
	"grpc":    {connecthttp.WithGRPC()},
	"grpcweb": {connecthttp.WithGRPCWeb()},
}
for nom, options := range protocoles {
	t.Run(nom, func(t *testing.T) {
		c := tachesv1connect.NewTacheServiceClient(connect.NewClient(
			connecthttp.NewTransport(h2c(), adresse, options...)))
		…
	})
}
```

Un dernier test prouve la compatibilité avec l'implémentation officielle,
[grpc-go](https://github.com/grpc/grpc-go). `conn.Invoke` appelle une méthode
par son nom, avec les messages générés, sans code généré pour grpc-go :

```go
conn, err := grpc.NewClient(adresse, grpc.WithTransportCredentials(insecure.NewCredentials()))
…
ctx := metadata.AppendToOutgoingContext(t.Context(), "authorization", "Bearer secret")
var rep tachesv1.CreeResponse
err = conn.Invoke(ctx, tachesv1connect.TacheServiceCreeProcedure,
	&tachesv1.CreeRequest{Titre: "via grpc-go"}, &rep)
```

## REST ou gRPC ?

| Critère | REST + OpenAPI | gRPC / Connect |
|---------|----------------|----------------|
| Contrat | OpenAPI (YAML), souvent écrit après le code | `.proto`, la source du code |
| Encodage | JSON | Protobuf binaire (compact), ou JSON |
| Flux | non (sauf SSE, WebSocket) | flux serveur, client et bidirectionnels |
| Évolution | à la discrétion de chacun | règles strictes, vérifiées par `buf breaking` |
| Navigateur, `curl` | natif | protocole Connect ou gRPC-Web |
| Cas typique | API publique, ressources | communication entre services |

Les deux se combinent : une API publique en REST pour les clients externes,
et gRPC entre les services internes. Des outils comme
[Vanguard](https://github.com/connectrpc/vanguard-go) exposent même un service
Connect en REST, d'après des annotations du `.proto`.

## En résumé

- Le fichier `.proto` est le contrat ; **buf** le vérifie et génère le code.
- **Connect** sert gRPC, gRPC-Web et Connect sur la même adresse, avec
  `net/http`.
- En v2, les méthodes reçoivent et renvoient des **messages simples**, les
  intercepteurs entourent l'**appel entier**, et seules les `*connect.Error`
  envoient un message au client.
- Les **flux** s'appuient sur la concurrence de Go : contexte annulé à la
  déconnexion, canaux tamponnés, jamais de blocage des autres clients.
- `connectinprocess` et `synctest` rendent les tests rapides et déterministes ;
  un test avec grpc-go prouve la compatibilité.
