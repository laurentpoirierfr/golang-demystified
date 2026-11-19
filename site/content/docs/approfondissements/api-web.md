---
title: "Une API web complète"
weight: 30
---

Ce chapitre construit un service web complet, prêt pour la production, avec la
seule bibliothèque standard et un pilote de base de données : une API JSON de
gestion de tâches, stockées dans [SQLite](https://sqlite.org). Il rassemble ce que les chapitres
précédents ont présenté séparément : routage HTTP, JSON, `database/sql`,
`log/slog`, `context`, `embed` et les tests.

Le code complet est dans `examples/approfondissements/api`, et chaque
fonctionnalité est couverte par un test.

## L'API

| Méthode et chemin        | Rôle                          | Réponse            |
|--------------------------|-------------------------------|--------------------|
| `GET /taches`            | Liste les tâches              | 200                |
| `POST /taches`           | Crée une tâche `{"titre"}`    | 201 + `Location`   |
| `GET /taches/{id}`       | Détail d'une tâche            | 200 ou 404         |
| `PATCH /taches/{id}`     | Modifie `{"faite"}`           | 200 ou 404         |
| `DELETE /taches/{id}`    | Supprime une tâche            | 204 ou 404         |
| `GET /sante`             | Contrôle de santé             | 200 ou 503         |
| `GET /openapi.yaml`      | Le contrat OpenAPI de l'API   | 200                |
| `GET /docs`              | La documentation interactive  | 200                |

Les erreurs sont toujours du JSON, `{"erreur": "…"}`, avec le statut adapté :
400 pour une requête mal formée ou non conforme au contrat, 404 pour une tâche
absente, 422 pour une donnée refusée par les règles métier, 500 pour une
erreur interne.

L'interface de l'API est décrite par un **contrat OpenAPI**, à partir duquel
le code du routage et des types est généré.

## L'organisation du projet

```
api/
├── api/
│   ├── openapi.yaml         le contrat OpenAPI 3, source de vérité
│   └── oapi-codegen.yaml    la configuration du générateur
├── cmd/
│   ├── taches/              le serveur : assemblage, démarrage, arrêt propre
│   ├── taches-huma/         la même API, écrite avec Huma
│   └── taches-cli/          le client en ligne de commande (chapitre suivant)
├── internal/
│   ├── cli/                 les commandes de taches-cli (cobra, viper)
│   ├── config/              lecture de la configuration
│   ├── tache/               le domaine : type, validation, erreurs
│   ├── store/               stockage SQLite ou PostgreSQL (database/sql)
│   │   └── migrations/      schéma SQL par base, embarqué dans l'exécutable
│   ├── web/                 handlers HTTP de l'API
│   │   └── openapi.gen.go   types et routage générés depuis le contrat
│   └── webhuma/             la variante « code d'abord » avec Huma
└── pkg/
    ├── client/              client Go de l'API, généré depuis le contrat
    └── httpkit/             middlewares, JSON, documentation : réutilisables
```

Le répertoire `api/` à la racine, pour les contrats d'interface (OpenAPI,
Protocol Buffers, schémas JSON), est une autre convention courante.

Le projet suit la disposition la plus répandue dans l'écosystème Go :

| Répertoire  | Contenu                                                            |
|-------------|--------------------------------------------------------------------|
| `cmd/`      | un sous-répertoire par exécutable, avec un `main.go` minimal       |
| `internal/` | le code propre au service ; le compilateur interdit aux autres modules de l'importer |
| `pkg/`      | le code générique, sans lien avec le métier, que d'autres projets peuvent importer |

Chaque package a une responsabilité unique, et les dépendances vont dans un
seul sens : `cmd` assemble tout, `web` et `store` dépendent de `tache`,
`tache` ne dépend de rien, et `pkg/httpkit` ignore tout du reste du projet.

> [!NOTE]
> `cmd/` et `internal/` sont des conventions reconnues par l'outil `go`
> (`internal/` est même imposé par le compilateur). `pkg/` est une convention
> de la communauté, popularisée par le dépôt *golang-standards/project-layout*
> ; l'équipe Go ne l'impose pas. Elle a le mérite de signaler clairement ce
> qui est réutilisable. N'y placez que du code réellement générique : en cas
> de doute, `internal/` est le bon choix, car il est toujours possible d'en
> sortir un package plus tard.

## Le domaine

Le package `tache` contient le type métier, ses règles et ses erreurs. Il ne
sait rien du HTTP ni du SQL :

```go
type Tache struct {
	ID       int64     `json:"id"`
	Titre    string    `json:"titre"`
	Faite    bool      `json:"faite"`
	Creation time.Time `json:"creation"`
}

var ErrIntrouvable = errors.New("tâche introuvable")

type ErrValidation struct {
	Champ  string
	Raison string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("%s : %s", e.Champ, e.Raison)
}

// ValideTitre vérifie un titre et renvoie sa forme normalisée.
func ValideTitre(titre string) (string, error) {
	titre = strings.TrimSpace(titre)
	switch {
	case titre == "":
		return "", &ErrValidation{Champ: "titre", Raison: "obligatoire"}
	case utf8.RuneCountInString(titre) > TitreMax:
		return "", &ErrValidation{Champ: "titre", Raison: fmt.Sprintf("%d caractères au plus", TitreMax)}
	}
	return titre, nil
}
```

## Le stockage : database/sql

### Ouvrir la base

`database/sql` définit une API commune, et un **pilote** l'implémente pour
chaque base. Le pilote s'enregistre par un import « à blanc ». Le service
accepte deux bases, choisies par la chaîne de connexion :

- **SQLite**, avec [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), écrit entièrement en Go : il ne
  demande pas de compilateur C, et l'exécutable reste autonome. Idéal pour le
  développement, les tests et les petits déploiements ;
- **[PostgreSQL](https://www.postgresql.org)**, avec `github.com/jackc/pgx/v5/stdlib`, pour la production.

Ce qui diffère d'une base à l'autre tient dans une petite structure :

```go
import (
	_ "github.com/jackc/pgx/v5/stdlib" // pilote PostgreSQL, enregistré sous le nom « pgx »
	_ "modernc.org/sqlite"             // pilote SQLite, enregistré sous le nom « sqlite »
)

type dialecte struct {
	pilote     string // nom du pilote database/sql
	migrations string // répertoire des migrations
	dollars    bool   // paramètres $1, $2… au lieu de ?
}

func Ouvre(ctx context.Context, dsn string) (*Store, error) {
	d := sqlite
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		d = postgres
	}
	db, err := sql.Open(d.pilote, dsn)
	if err != nil {
		return nil, err
	}
	if d == sqlite {
		db.SetMaxOpenConns(1) // SQLite n'accepte qu'un écrivain à la fois
	}
	if err := db.PingContext(ctx); err != nil { // sql.Open ne se connecte pas
		db.Close()
		return nil, fmt.Errorf("connexion à la base %s : %w", d.pilote, err)
	}
	s := &Store{db: db, d: d, maintenant: time.Now}
	if err := s.migre(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations : %w", err)
	}
	return s, nil
}
```

Les requêtes sont écrites une seule fois, avec des `?`. Une méthode `q` les
convertit en `$1`, `$2`… pour PostgreSQL :

```go
s.db.ExecContext(ctx, s.q(`DELETE FROM taches WHERE id = ?`), id)
```

Un `*sql.DB` est un **pool de connexions** sûr pour un usage concurrent : on
l'ouvre une fois au démarrage, et on le partage.

Le même stockage est testé sur les **deux** bases : SQLite en mémoire à chaque
`go test`, et un vrai PostgreSQL démarré dans un conteneur par
**[Testcontainers](https://golang.testcontainers.org)**. Le chapitre sur le package `testing` détaille cette
technique.

### Les migrations

Le schéma est décrit par des fichiers SQL numérotés, embarqués dans
l'exécutable. Au démarrage, chaque fichier qui n'a pas encore été appliqué
l'est dans une transaction, et son nom est enregistré dans une table
`migrations` :

```go
//go:embed migrations
var migrations embed.FS

func (s *Store) migre(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS migrations (nom TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	fichiers, err := fs.Glob(migrations, s.d.migrations+"/*.sql") // triés par nom
	if err != nil {
		return err
	}
	for _, f := range fichiers {
		// … ignorer f s'il figure déjà dans la table migrations …
		requete, err := migrations.ReadFile(f)
		if err != nil {
			return err
		}
		if err := s.transaction(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(requete)); err != nil {
				return fmt.Errorf("%s : %w", f, err)
			}
			_, err := tx.ExecContext(ctx, s.q(`INSERT INTO migrations (nom) VALUES (?)`), f)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}
```

Pour un projet plus important, des outils comme [`goose`](https://github.com/pressly/goose) ou [`golang-migrate`](https://github.com/golang-migrate/migrate)
gèrent aussi les retours en arrière.

### Les requêtes

Les trois méthodes de requête de `*sql.DB` couvrent tous les cas :

| Méthode            | Usage                                       |
|--------------------|---------------------------------------------|
| `QueryContext`     | Plusieurs lignes, parcourues avec `Next`    |
| `QueryRowContext`  | Une seule ligne                             |
| `ExecContext`      | Pas de résultat : `INSERT`, `UPDATE`, `DELETE` |

```go
func (s *Store) Liste(ctx context.Context) ([]tache.Tache, error) {
	lignes, err := s.db.QueryContext(ctx,
		`SELECT id, titre, faite, creation FROM taches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer lignes.Close() // libère la connexion

	taches := []tache.Tache{} // une liste vide s'encode en [], pas en null
	for lignes.Next() {
		t, err := scanne(lignes)
		if err != nil {
			return nil, err
		}
		taches = append(taches, t)
	}
	return taches, lignes.Err() // erreur éventuelle pendant le parcours
}

func (s *Store) Trouve(ctx context.Context, id int64) (tache.Tache, error) {
	ligne := s.db.QueryRowContext(ctx,
		s.q(`SELECT id, titre, faite, creation FROM taches WHERE id = ?`), id)
	t, err := scanne(ligne)
	if errors.Is(err, sql.ErrNoRows) {
		return tache.Tache{}, tache.ErrIntrouvable // traduit en erreur du domaine
	}
	return t, err
}
```

Trois règles à respecter :

- toujours passer les valeurs en **paramètres** (`?`, `$1`), jamais par
  concaténation : c'est la protection contre les injections SQL ;
- toujours **fermer** les `Rows` (`defer lignes.Close()`) et vérifier
  `lignes.Err()` après la boucle ;
- traduire les erreurs techniques (`sql.ErrNoRows`) en erreurs du **domaine**
  (`tache.ErrIntrouvable`) : le reste du programme ne doit pas dépendre de SQL.

Une modification qui ne touche aucune ligne indique une tâche inexistante :

```go
func (s *Store) Supprime(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM taches WHERE id = ?`), id)
	return verifieUneLigne(res, err) // ErrIntrouvable si 0 ligne
}
```

## Le contrat OpenAPI

Une API est un **contrat** entre un service et ses clients. Le décrire au
format [OpenAPI](https://www.openapis.org) apporte une documentation lisible
par les humains et par les outils : documentation interactive, génération de
clients dans tous les langages, tests de conformité, passerelles d'API.

### Code d'abord ou contrat d'abord ?

Il existe deux façons d'obtenir ce contrat en Go :

| Approche             | Outil          | Principe                                         | Format         |
|----------------------|----------------|--------------------------------------------------|----------------|
| Code d'abord         | **[swag](https://github.com/swaggo/swag)**       | commentaires `// @Router …` au-dessus des handlers | Swagger 2.0 (v1) |
| Code d'abord         | **Huma**       | contrat déduit des types Go des entrées et sorties | OpenAPI 3.1    |
| **Contrat d'abord**  | **[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)** | code Go généré depuis le contrat               | OpenAPI 3.0    |

`swag` est l'outil historique : le contrat est produit à partir de
commentaires placés au-dessus des handlers. Il a deux défauts : sa version
stable ne produit que l'ancien format **Swagger 2.0** (la version 2, qui
vise OpenAPI 3, est en préversion depuis plusieurs années), et rien ne
garantit que les commentaires disent la vérité sur le code.

Ce chapitre utilise **oapi-codegen**, avec l'approche *contrat d'abord* : on
écrit le contrat, et le générateur produit les types et une **interface Go**
que le serveur doit implémenter. Si le code s'écarte du contrat, il **ne
compile plus**. Le contrat peut aussi être discuté et validé avec les équipes
clientes avant la première ligne de code.

> [!TIP]
> Si vous préférez écrire le code d'abord, **Huma** est l'alternative moderne
> à swag : il déduit un contrat OpenAPI 3.1 des types Go de chaque opération,
> valide les entrées, et fonctionne avec le routeur standard de Go.

### Le contrat

Le contrat décrit les chemins, les opérations, les paramètres, les corps et
les réponses. Chaque opération porte un `operationId`, qui deviendra le nom de
la méthode Go :

```yaml
openapi: 3.0.3
info:
  title: API des tâches
  version: 1.0.0
paths:
  /taches:
    post:
      operationId: creeTache
      summary: Crée une tâche
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/NouvelleTache" }
      responses:
        "201":
          description: La tâche créée
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Tache" }
        "400": { $ref: "#/components/responses/RequeteInvalide" }
        "422": { $ref: "#/components/responses/DonneeInvalide" }
  /taches/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema: { type: integer, format: int64, minimum: 1 }
    …
components:
  schemas:
    NouvelleTache:
      type: object
      additionalProperties: false
      required: [titre]
      properties:
        titre: { type: string, minLength: 1, maxLength: 200 }
    ModificationTache:
      type: object
      additionalProperties: false
      required: [faite]
      properties:
        faite: { type: boolean }
    …
```

### Générer le code

oapi-codegen est déclaré comme **outil** du module, grâce à la directive
`tool` de Go 1.24 : sa version est figée dans `go.mod`, et `go tool` le lance
sans installation.

```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2
```

La configuration demande les types, le serveur pour le routeur standard de Go
et le contrat embarqué :

```yaml
# api/oapi-codegen.yaml
package: web
output: ../internal/web/openapi.gen.go
generate:
  models: true          # les types des schémas
  std-http-server: true # l'interface du serveur et le routage net/http (Go 1.22)
  embedded-spec: true   # le contrat embarqué, pour valider les requêtes
```

Une directive `go:generate` relie le tout, et `go generate ./...` régénère le
code après chaque modification du contrat :

```go
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
```

### Le code généré

Le fichier `openapi.gen.go` contient un type par schéma, et une interface avec
une méthode par opération. Les paramètres de chemin sont déjà convertis :

```go
type NouvelleTache struct {
	Titre string `json:"titre"`
}

type ServerInterface interface {
	// Contrôle de santé
	// (GET /sante)
	Sante(w http.ResponseWriter, r *http.Request)
	// Liste les tâches
	// (GET /taches)
	ListeTaches(w http.ResponseWriter, r *http.Request)
	// Crée une tâche
	// (POST /taches)
	CreeTache(w http.ResponseWriter, r *http.Request)
	// Supprime une tâche
	// (DELETE /taches/{id})
	SupprimeTache(w http.ResponseWriter, r *http.Request, id int64)
	// Détail d'une tâche
	// (GET /taches/{id})
	DetailTache(w http.ResponseWriter, r *http.Request, id int64)
	// Marque une tâche comme faite, ou non
	// (PATCH /taches/{id})
	ModifieTache(w http.ResponseWriter, r *http.Request, id int64)
}
```

Il contient aussi `HandlerWithOptions`, qui enregistre chaque opération sur un
`http.ServeMux` avec les motifs de Go 1.22 (`GET /taches/{id}`…).

### Valider les requêtes par le contrat

Le contrat dit bien plus que les types : un titre fait au plus 200 caractères,
un identifiant vaut au moins 1, un champ inconnu est interdit, le corps doit
être du JSON. Le middleware [`nethttp-middleware`](https://github.com/oapi-codegen/nethttp-middleware) vérifie chaque requête
**par rapport au contrat**, avant qu'elle n'atteigne le handler :

```go
contrat, err := GetSpec() // le contrat embarqué par oapi-codegen
if err != nil {
	return nil, fmt.Errorf("contrat OpenAPI : %w", err)
}
valide := middleware.OapiRequestValidatorWithOptions(contrat, &middleware.Options{
	DoNotValidateServers: true, // le service peut être joint par n'importe quelle adresse
	ErrorHandler: func(w http.ResponseWriter, message string, statut int) {
		httpkit.EcritJSON(w, statut, Erreur{Erreur: message})
	},
})
```

Les réponses désignent précisément l'écart au contrat :

```
POST /taches {"titre": "xxx…"}         → 400 Error at "/titre": maximum string length is 200
POST /taches {"titre": "a", "priorite": 1} → 400 property "priorite" is unsupported
PATCH /taches/1 {}                      → 400 Error at "/faite": property "faite" is missing
GET /taches/0                           → 400 number must be at least 1
POST /taches (sans Content-Type)        → 400 header Content-Type has unexpected value ""
```

Le handler ne s'occupe plus que des **règles métier**, que le contrat ne peut
pas exprimer : un titre composé uniquement d'espaces, par exemple, reste
refusé par le domaine avec un statut 422.

### Servir le contrat et sa documentation

Le service publie son propre contrat, et une page de documentation
interactive basée sur **[Swagger UI](https://swagger.io/tools/swagger-ui/)**, chargée depuis un CDN : aucune
dépendance Go n'est nécessaire.

```go
mux.Handle("GET /openapi.yaml", httpkit.Fichier("application/yaml", api.Contrat))
mux.Handle("GET /docs", httpkit.SwaggerUI("API des tâches", "/openapi.yaml"))
```

En ouvrant `http://localhost:8080/docs`, on parcourt l'API et on l'essaie
directement depuis le navigateur. Le même contrat permet de générer des
clients, par exemple en TypeScript pour une application web, ou en Go avec
l'option `client: true` d'oapi-codegen.

### La variante « code d'abord » avec Huma

Pour comparer les deux approches, le projet contient une seconde version de
la même API, écrite avec [Huma](https://huma.rocks) : le package
`internal/webhuma` et l'exécutable `cmd/taches-huma`. Elle réutilise le même
domaine (`internal/tache`) et le même stockage (`internal/store`).

Avec Huma, il n'y a pas de fichier de contrat à écrire. Chaque opération
déclare ses **entrées** et ses **sorties** sous forme de structures Go, dont
les balises indiquent l'origine de chaque champ (chemin, en-tête, corps) et
ses règles de validation :

```go
type CreeEntree struct {
	Body struct {
		Titre string `json:"titre" minLength:"1" maxLength:"200" doc:"Titre de la tâche" example:"Relire le livre"`
	}
}

type TacheSortie struct {
	Location string `header:"Location" doc:"Adresse de la tâche"`
	Body     Tache
}

type ModifieEntree struct {
	ID   int64 `path:"id" minimum:"1" doc:"Identifiant de la tâche"`
	Body struct {
		Faite bool `json:"faite" doc:"Nouvel état de la tâche"` // obligatoire : pas d'omitempty
	}
}
```

Un handler est une fonction **typée**, qui reçoit son entrée déjà décodée et
validée, et renvoie sa sortie ou une erreur. Il n'écrit jamais lui-même dans
le `http.ResponseWriter` :

```go
huma.Register(api, huma.Operation{
	OperationID: "cree-tache", Method: http.MethodPost, Path: "/taches",
	Summary: "Crée une tâche", Tags: []string{"taches"},
	DefaultStatus: http.StatusCreated,
}, func(ctx context.Context, e *CreeEntree) (*TacheSortie, error) {
	titre, err := tache.ValideTitre(e.Body.Titre) // règle métier : pas de titre blanc
	if err != nil {
		return nil, erreur(err)
	}
	t, err := s.Cree(ctx, titre)
	if err != nil {
		return nil, erreur(err)
	}
	return &TacheSortie{Location: fmt.Sprintf("/taches/%d", t.ID), Body: depuis(t)}, nil
})
```

Les erreurs du domaine sont traduites par les fonctions de Huma, qui
produisent des réponses au format standard **« problem+json »** (RFC 9457) :

```go
func erreur(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, tache.ErrIntrouvable):
		return huma.Error404NotFound(err.Error())
	}
	if v, ok := errors.AsType[*tache.ErrValidation](err); ok {
		return huma.Error422UnprocessableEntity(v.Error())
	}
	return err // erreur 500
}
```

Le serveur s'appuie sur le routeur standard, grâce à l'adaptateur `humago` :

```go
mux := http.NewServeMux()
api := humago.New(mux, huma.DefaultConfig("API des tâches (Huma)", "1.0.0"))
Enregistre(api, s)
```

Huma **déduit** le contrat OpenAPI 3.1 de ces types, et le sert avec la
documentation, sans configuration : `/openapi.yaml`, `/openapi.json` et
`/docs`. Extrait du contrat produit :

```yaml
CreeEntreeBody:
  additionalProperties: false
  properties:
    titre:
      description: Titre de la tâche
      examples:
        - Relire le livre
      maxLength: 200
      minLength: 1
      type: string
```

Les requêtes invalides sont rejetées avant d'atteindre le handler, avec la
liste de **toutes** les erreurs :

```
POST /taches {"titre": "", "priorite": 1}

422 application/problem+json
{"title":"Unprocessable Entity","status":422,"detail":"validation failed",
 "errors":[{"message":"expected length >= 1","location":"body.titre","value":""},
           {"message":"unexpected property","location":"body.priorite",…}]}
```

Le package `humatest` teste les opérations sans réseau, et donne accès au
contrat déduit :

```go
func TestScenario(t *testing.T) {
	_, api := humatest.New(t)
	Enregistre(api, nouveauStore(t))

	r := api.Post("/taches", map[string]any{"titre": "  Relire  "})
	// r.Code == 201, r.Body contient "titre":"Relire"

	r = api.Patch("/taches/1", map[string]any{}) // « faite » manquant
	// r.Code == 422 : expected required property faite
}
```

### Quelle approche choisir ?

| Critère                         | oapi-codegen (contrat d'abord)            | Huma (code d'abord)                       |
|---------------------------------|-------------------------------------------|-------------------------------------------|
| Source de vérité                | le fichier `openapi.yaml`                 | les types Go                              |
| Format                          | OpenAPI 3.0                               | OpenAPI 3.1                               |
| Contrat discuté avant le code   | oui, c'est son principe                   | non, il découle du code                   |
| Écart entre code et contrat     | impossible : le code ne compile plus      | impossible : le contrat est déduit        |
| Handlers                        | `http.Handler` classiques                 | fonctions typées `entrée → sortie`        |
| Erreurs                         | format libre                              | « problem+json » (RFC 9457)               |
| Client généré                   | oui (Go)                                  | via le contrat, avec un autre outil       |
| Dépendances                     | générateur + validateur                   | une bibliothèque                          |

Les deux approches éliminent le problème de fond de swag : un contrat qui ne
dit pas la vérité. Choisissez **oapi-codegen** lorsque le contrat est négocié
avec d'autres équipes, ou partagé par plusieurs langages ; choisissez
**Huma** lorsque l'API naît dans le code Go et que vous voulez écrire le moins
de code possible.

## La couche HTTP

### L'interface du stockage

Le package `web` déclare l'interface **dont il a besoin**. Il ne dépend pas du
package `store` : n'importe quelle implémentation convient, y compris un
double de test.

```go
type Store interface {
	Liste(ctx context.Context) ([]tache.Tache, error)
	Trouve(ctx context.Context, id int64) (tache.Tache, error)
	Cree(ctx context.Context, titre string) (tache.Tache, error)
	MarqueFaite(ctx context.Context, id int64, faite bool) (tache.Tache, error)
	Supprime(ctx context.Context, id int64) error
	Ping(ctx context.Context) error
}

type Serveur struct {
	store Store
	log   *slog.Logger
}

// Le compilateur vérifie que Serveur implémente toutes les opérations du
// contrat : une opération ajoutée au contrat sans être codée ne compile pas.
var _ ServerInterface = (*Serveur)(nil)

func (s *Serveur) Routes() (http.Handler, error) {
	… // préparation du middleware de validation (voir plus haut)

	mux := http.NewServeMux()
	HandlerWithOptions(s, StdHTTPServerOptions{
		BaseRouter:  mux,
		Middlewares: []MiddlewareFunc{valide}, // chaque requête est validée par le contrat
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			s.erreur(w, r, &httpkit.ErrRequete{Raison: "paramètre invalide : " + err.Error()})
		},
	})
	mux.Handle("GET /openapi.yaml", httpkit.Fichier("application/yaml", api.Contrat))
	mux.Handle("GET /docs", httpkit.SwaggerUI("API des tâches", "/openapi.yaml"))

	// Le premier middleware de la liste est le plus extérieur.
	return httpkit.Chaine(mux,
		httpkit.Identifiant(),
		httpkit.Journal(s.log),
		httpkit.Recuperation(s.log),
	), nil
}
```

Les handlers sont les méthodes de l'interface générée. Les identifiants
arrivent déjà convertis en `int64`, et validés :

```go
// DetailTache implémente GET /taches/{id}.
func (s *Serveur) DetailTache(w http.ResponseWriter, r *http.Request, id int64) {
	t, err := s.store.Trouve(r.Context(), id)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, t)
}
```

### Lire et écrire du JSON

La lecture du corps d'une requête est l'endroit où un service est le plus
exposé. Le décodeur limite la taille, refuse les champs inconnus et exige un
unique objet. Ce code ne dépend pas du métier : il est dans `pkg/httpkit`.

```go
// ErrRequete signale une requête mal formée, à traduire en statut 400.
type ErrRequete struct{ Raison string }

func (e *ErrRequete) Error() string { return e.Raison }

// LitJSON décode un unique objet JSON, de 1 Mo au plus, sans champ inconnu.
func LitJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, TailleMax))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return &ErrRequete{"JSON invalide : " + err.Error()}
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return &ErrRequete{"un seul objet JSON est attendu"}
	}
	return nil
}
```

Le corps est décodé dans le type généré depuis le contrat. Comme le contrat
déclare `faite` obligatoire et que le middleware l'a vérifié, un simple
`bool` suffit :

```go
var entree ModificationTache // « faite » est obligatoire : vérifié par le contrat
if err := httpkit.LitJSON(w, r, &entree); err != nil {
	s.erreur(w, r, err)
	return
}
```

Sans contrat, il aurait fallu un pointeur (`*bool`) pour distinguer un champ
**absent** d'un champ à `false`.

### Traduire les erreurs

Les handlers ne choisissent pas eux-mêmes les statuts d'erreur : ils
transmettent l'erreur à une fonction unique, qui la traduit. Les erreurs
inattendues sont journalisées, mais leur détail n'est **jamais** envoyé au
client, car il pourrait révéler des informations internes :

```go
func (s *Serveur) erreur(w http.ResponseWriter, r *http.Request, err error) {
	statut, message := http.StatusInternalServerError, "erreur interne"
	if v, ok := errors.AsType[*tache.ErrValidation](err); ok {
		statut, message = http.StatusUnprocessableEntity, v.Error()
	} else if e, ok := errors.AsType[*httpkit.ErrRequete](err); ok {
		statut, message = http.StatusBadRequest, e.Error()
	} else if errors.Is(err, tache.ErrIntrouvable) {
		statut, message = http.StatusNotFound, err.Error()
	} else {
		s.log.ErrorContext(r.Context(), "erreur interne", "erreur", err)
	}
	httpkit.EcritJSON(w, statut, Erreur{Erreur: message}) // type généré depuis le contrat
}
```

Un handler devient alors une suite d'étapes lisible :

```go
func (s *Serveur) CreeTache(w http.ResponseWriter, r *http.Request) {
	var entree NouvelleTache // type généré depuis le contrat
	if err := httpkit.LitJSON(w, r, &entree); err != nil {
		s.erreur(w, r, err)
		return
	}
	titre, err := tache.ValideTitre(entree.Titre)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	t, err := s.store.Cree(r.Context(), titre)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/taches/%d", t.ID))
	httpkit.EcritJSON(w, http.StatusCreated, t)
}
```

### Les middlewares

Trois middlewares de `pkg/httpkit` s'appliquent à toutes les requêtes :

- **`Identifiant`** attribue à chaque requête un identifiant unique (un UUID
  v7, avec le package `uuid` de Go 1.27), placé dans le contexte et renvoyé
  dans l'en-tête `X-Request-ID`. Il permet de retrouver tous les messages liés
  à une requête ;
- **`Journal`** écrit une ligne de journal structurée par requête ;
- **`Recuperation`** transforme une panique en réponse 500.

Un middleware qui a besoin d'une dépendance, comme le journal, est une
fonction qui la reçoit et renvoie le middleware :

```go
type Middleware func(http.Handler) http.Handler

func Chaine(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
```

Pour journaliser le statut de la réponse, le middleware enveloppe le
`ResponseWriter` :

```go
type enregistreur struct {
	http.ResponseWriter
	statut int
}

func (e *enregistreur) WriteHeader(statut int) {
	e.statut = statut
	e.ResponseWriter.WriteHeader(statut)
}

// Unwrap permet à http.ResponseController d'atteindre le ResponseWriter d'origine.
func (e *enregistreur) Unwrap() http.ResponseWriter { return e.ResponseWriter }

func Journal(log *slog.Logger) Middleware {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debut := time.Now()
			rec := &enregistreur{ResponseWriter: w, statut: http.StatusOK}
			suivant.ServeHTTP(rec, r)
			log.InfoContext(r.Context(), "requête",
				slog.String("requete_id", RequeteID(r.Context())),
				slog.String("methode", r.Method),
				slog.String("chemin", r.URL.Path),
				slog.Int("statut", rec.statut),
				slog.Duration("duree", time.Since(debut)),
			)
		})
	}
}
```

```json
{"time":"…","level":"INFO","msg":"requête","requete_id":"0199…","methode":"POST","chemin":"/taches","statut":201,"duree":412000}
```

## Le programme principal

### La configuration

La configuration vient de variables d'environnement, avec des valeurs par
défaut. La fonction reçoit `getenv` en paramètre, ce qui la rend testable :

```go
func Charge(getenv func(string) string) (Config, error) {
	c := Config{
		Adresse: cmp.Or(getenv("TACHES_ADRESSE"), ":8080"),
		Base:    cmp.Or(getenv("TACHES_BASE"), "file:taches.db"),
	}
	delai, err := time.ParseDuration(cmp.Or(getenv("TACHES_DELAI_ARRET"), "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("TACHES_DELAI_ARRET : %w", err)
	}
	c.DelaiArret = delai
	…
}
```

### main et run

`main` se contente de créer le contexte annulé par les signaux et d'appeler
`run`. Tout le programme est dans `run`, qui reçoit ses dépendances en
paramètres et renvoie une erreur : il se teste comme n'importe quelle fonction.

```go
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### L'arrêt propre

À la réception de `SIGTERM` (envoyé par [Kubernetes](https://kubernetes.io) ou [systemd](https://systemd.io) avant
d'arrêter un service), le serveur cesse d'accepter des connexions, termine les
requêtes en cours, puis ferme la base de données :

```go
erreurs := make(chan error, 1)
go func() { erreurs <- srv.Serve(ecoute) }()

select {
case err := <-erreurs: // le serveur s'est arrêté seul : c'est une erreur
	return err
case <-ctx.Done(): // signal reçu : arrêt propre
}

arret, annule := context.WithTimeout(context.Background(), cfg.DelaiArret)
defer annule()
if err := srv.Shutdown(arret); err != nil { // attend les requêtes en cours
	return fmt.Errorf("arrêt : %w", err)
}
```

Le contexte d'arrêt part de `context.Background()`, et non du contexte
annulé par le signal : sinon, `Shutdown` abandonnerait immédiatement.
`defer s.Close()` ferme la base après l'arrêt du serveur, puisque les `defer`
s'exécutent en ordre inverse.

## Les tests

Le test principal joue un **scénario complet** à travers le handler HTTP, avec
une vraie base SQLite en mémoire. Il ne demande ni réseau ni fichier, et
s'exécute en quelques millisecondes :

```go
func nouveauServeur(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()
	s, err := store.Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	var journal bytes.Buffer
	return Nouveau(s, slog.New(slog.NewJSONHandler(&journal, nil))).Routes(), &journal
}

func TestScenario(t *testing.T) {
	h, _ := nouveauServeur(t)
	etapes := []struct {
		nom, methode, chemin, corps string
		statut                      int
		contient                    string
	}{
		{"liste vide", "GET", "/taches", "", 200, "[]"},
		{"création", "POST", "/taches", `{"titre": "  Relire le livre  "}`, 201, `"titre":"Relire le livre"`},
		{"modification", "PATCH", "/taches/1", `{"faite": true}`, 200, `"faite":true`},
		{"titre vide", "POST", "/taches", `{"titre": " "}`, 422, "titre : obligatoire"},
		{"champ inconnu (contrat)", "POST", "/taches", `{"titre": "a", "priorite": 1}`, 400, `property \"priorite\" is unsupported`},
		{"identifiant nul (contrat)", "GET", "/taches/0", "", 400, "number must be at least 1"},
		{"introuvable", "GET", "/taches/99", "", 404, "tâche introuvable"},
		{"méthode interdite", "PUT", "/taches/1", "", 405, ""},
		…
	}
	…
}
```

Les briques de `pkg/httpkit` ont leurs propres tests, indépendants du
service : ordre des middlewares, récupération d'une panique, lecture du JSON.
Un dernier test démarre le **vrai programme** sur un port libre
(`TACHES_ADRESSE=127.0.0.1:0`), envoie une requête, puis annule son contexte
pour simuler un `SIGTERM` et vérifier l'arrêt propre.

## Déployer

Le pilote SQLite étant écrit en Go, l'exécutable est **statique** : une image
[Docker](https://www.docker.com) minimale suffit.

```dockerfile
FROM golang:1.27 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /taches ./cmd/taches

FROM gcr.io/distroless/static-debian12
COPY --from=build /taches /taches
ENV TACHES_BASE=file:/data/taches.db
VOLUME /data
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/taches"]
```

L'image se construit et s'exécute avec Docker ou avec [Podman](https://podman.io), dont les
commandes sont identiques :

```bash
# Avec Docker
docker build -t taches .
docker run --rm -p 8080:8080 -v taches-data:/data taches

# Avec Podman (sans démon, sans droits root)
podman build -t taches .
podman run --rm -p 8080:8080 -v taches-data:/data taches
```

Pour la production, la même image se connecte à PostgreSQL en changeant une
seule variable :

```bash
docker run --rm -p 8080:8080 \
  -e TACHES_BASE=postgres://gopher:secret@db.example.com:5432/taches \
  taches
```

> [!TIP]
> Avec Podman, `alias docker=podman` permet de suivre sans modification la
> plupart des documentations écrites pour Docker. Le paquet `podman-docker`
> de nombreuses distributions installe cet alias pour tout le système.

## Pour aller plus loin

Ce service est volontairement simple, mais sa structure supporte la
croissance. Les évolutions classiques sont :

- le passage à **PostgreSQL** en production, sans changer le code :
  `TACHES_BASE=postgres://utilisateur:motdepasse@hote:5432/taches` ;
- l'**authentification**, par un middleware qui valide un jeton et place
  l'utilisateur dans le contexte ;
- la **pagination** de la liste (`?apres=42&limite=50`) ;
- les **métriques** ([Prometheus](https://prometheus.io), [OpenTelemetry](https://opentelemetry.io)) et le **traçage** distribué,
  par d'autres middlewares ;
- une **documentation OpenAPI**, écrite à la main ou générée.

Lorsque la logique métier grandit, il devient utile de la séparer plus
nettement de l'infrastructure : c'est l'objet du chapitre suivant,
l'architecture hexagonale.
