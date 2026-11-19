---
title: "Une API web complète"
weight: 30
---

Ce chapitre construit un service web complet, prêt pour la production, avec la
seule bibliothèque standard et un pilote de base de données : une API JSON de
gestion de tâches, stockées dans SQLite. Il rassemble ce que les chapitres
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

Les erreurs sont toujours du JSON, `{"erreur": "…"}`, avec le statut adapté :
400 pour une requête mal formée, 404 pour une tâche absente, 422 pour une
donnée invalide, 500 pour une erreur interne.

## L'organisation du projet

```
api/
├── cmd/
│   └── taches/
│       ├── main.go          assemblage, démarrage, arrêt propre
│       └── main_test.go
└── internal/
    ├── config/              lecture de la configuration
    ├── tache/               le domaine : type, validation, erreurs
    ├── store/               stockage SQLite (database/sql)
    │   └── migrations/      schéma SQL, embarqué dans l'exécutable
    └── web/                 handlers HTTP et middlewares
```

Chaque package a une responsabilité unique, et les dépendances vont dans un
seul sens : `cmd` assemble tout, `web` et `store` dépendent de `tache`, et
`tache` ne dépend de rien. Le répertoire `internal` empêche les autres modules
d'importer ces packages : seule l'API HTTP est publique.

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
chaque base. Le pilote s'enregistre par un import « à blanc ». Ici,
`modernc.org/sqlite` est écrit entièrement en Go : il ne demande pas de
compilateur C, et l'exécutable reste autonome.

```go
import _ "modernc.org/sqlite" // pilote enregistré sous le nom « sqlite »

func Ouvre(ctx context.Context, dsn string) (*SQLite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite n'accepte qu'un écrivain à la fois

	if err := db.PingContext(ctx); err != nil { // sql.Open ne se connecte pas
		db.Close()
		return nil, fmt.Errorf("connexion à %s : %w", dsn, err)
	}
	s := &SQLite{db: db, maintenant: time.Now}
	if err := s.migre(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations : %w", err)
	}
	return s, nil
}
```

Avec PostgreSQL, seuls l'import du pilote (par exemple
`github.com/jackc/pgx/v5/stdlib`), son nom et la syntaxe des paramètres (`$1`
au lieu de `?`) changeraient. Un `*sql.DB` est un **pool de connexions** sûr
pour un usage concurrent : on l'ouvre une fois au démarrage, et on le partage.

### Les migrations

Le schéma est décrit par des fichiers SQL numérotés, embarqués dans
l'exécutable. Au démarrage, chaque fichier qui n'a pas encore été appliqué
l'est dans une transaction, et son nom est enregistré dans une table
`migrations` :

```go
//go:embed migrations/*.sql
var migrations embed.FS

func (s *SQLite) migre(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS migrations (nom TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	fichiers, err := fs.Glob(migrations, "migrations/*.sql") // triés par nom
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
			_, err := tx.ExecContext(ctx, `INSERT INTO migrations (nom) VALUES (?)`, f)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}
```

Pour un projet plus important, des outils comme `goose` ou `golang-migrate`
gèrent aussi les retours en arrière.

### Les requêtes

Les trois méthodes de requête de `*sql.DB` couvrent tous les cas :

| Méthode            | Usage                                       |
|--------------------|---------------------------------------------|
| `QueryContext`     | Plusieurs lignes, parcourues avec `Next`    |
| `QueryRowContext`  | Une seule ligne                             |
| `ExecContext`      | Pas de résultat : `INSERT`, `UPDATE`, `DELETE` |

```go
func (s *SQLite) Liste(ctx context.Context) ([]tache.Tache, error) {
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

func (s *SQLite) Trouve(ctx context.Context, id int64) (tache.Tache, error) {
	ligne := s.db.QueryRowContext(ctx,
		`SELECT id, titre, faite, creation FROM taches WHERE id = ?`, id)
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
func (s *SQLite) Supprime(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM taches WHERE id = ?`, id)
	return verifieUneLigne(res, err) // ErrIntrouvable si 0 ligne
}
```

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

func (s *Serveur) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /taches", s.liste)
	mux.HandleFunc("POST /taches", s.cree)
	mux.HandleFunc("GET /taches/{id}", s.detail)
	mux.HandleFunc("PATCH /taches/{id}", s.modifie)
	mux.HandleFunc("DELETE /taches/{id}", s.supprime)
	mux.HandleFunc("GET /sante", s.sante)

	// Le premier middleware de la liste est le plus extérieur.
	return chaine(mux, s.identifie, s.journalise, s.recupere)
}
```

### Lire et écrire du JSON

La lecture du corps d'une requête est l'endroit où un service est le plus
exposé. Le décodeur limite la taille, refuse les champs inconnus et exige un
unique objet :

```go
func litJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return &errRequete{"JSON invalide : " + err.Error()}
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return &errRequete{"un seul objet JSON est attendu"}
	}
	return nil
}
```

Pour une modification partielle, un pointeur distingue un champ **absent**
d'un champ à sa valeur zéro :

```go
var entree struct {
	Faite *bool `json:"faite"` // nil si absent, sinon true ou false
}
```

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
	} else if e, ok := errors.AsType[*errRequete](err); ok {
		statut, message = http.StatusBadRequest, e.Error()
	} else if errors.Is(err, tache.ErrIntrouvable) {
		statut, message = http.StatusNotFound, err.Error()
	} else {
		s.log.ErrorContext(r.Context(), "erreur interne", "erreur", err)
	}
	ecritJSON(w, statut, map[string]string{"erreur": message})
}
```

Un handler devient alors une suite d'étapes lisible :

```go
func (s *Serveur) cree(w http.ResponseWriter, r *http.Request) {
	var entree struct {
		Titre string `json:"titre"`
	}
	if err := litJSON(w, r, &entree); err != nil {
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
	ecritJSON(w, http.StatusCreated, t)
}
```

### Les middlewares

Trois middlewares s'appliquent à toutes les requêtes :

- **`identifie`** attribue à chaque requête un identifiant unique (un UUID v7,
  avec le package `uuid` de Go 1.27), placé dans le contexte et renvoyé dans
  l'en-tête `X-Request-ID`. Il permet de retrouver tous les messages liés à une
  requête ;
- **`journalise`** écrit une ligne de journal structurée par requête ;
- **`recupere`** transforme une panique en réponse 500.

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

func (s *Serveur) journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		rec := &enregistreur{ResponseWriter: w, statut: http.StatusOK}
		suivant.ServeHTTP(rec, r)
		s.log.InfoContext(r.Context(), "requête",
			slog.String("requete_id", RequeteID(r.Context())),
			slog.String("methode", r.Method),
			slog.String("chemin", r.URL.Path),
			slog.Int("statut", rec.statut),
			slog.Duration("duree", time.Since(debut)),
		)
	})
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

À la réception de `SIGTERM` (envoyé par Kubernetes ou systemd avant
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
		{"champ inconnu", "POST", "/taches", `{"titre": "a", "priorite": 1}`, 400, "unknown field"},
		{"introuvable", "GET", "/taches/99", "", 404, "tâche introuvable"},
		{"méthode interdite", "PUT", "/taches/1", "", 405, ""},
		…
	}
	…
}
```

Un double de test qui panique vérifie le middleware de récupération, et un
dernier test démarre le **vrai programme** sur un port libre
(`TACHES_ADRESSE=127.0.0.1:0`), envoie une requête, puis annule son contexte
pour simuler un `SIGTERM` et vérifier l'arrêt propre.

## Déployer

Le pilote SQLite étant écrit en Go, l'exécutable est **statique** : une image
Docker minimale suffit.

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

## Pour aller plus loin

Ce service est volontairement simple, mais sa structure supporte la
croissance. Les évolutions classiques sont :

- l'**authentification**, par un middleware qui valide un jeton et place
  l'utilisateur dans le contexte ;
- la **pagination** de la liste (`?apres=42&limite=50`) ;
- les **métriques** (Prometheus, OpenTelemetry) et le **traçage** distribué,
  par d'autres middlewares ;
- une **documentation OpenAPI**, écrite à la main ou générée.

Lorsque la logique métier grandit, il devient utile de la séparer plus
nettement de l'infrastructure : c'est l'objet du chapitre suivant,
l'architecture hexagonale.
