---
title: "Accès aux données : sqlc et golang-migrate"
weight: 35
---

Le chapitre « Une API web complète » écrit ses requêtes SQL à la main, avec
`database/sql`. C'est simple et transparent, mais répétitif : chaque requête
demande son `Scan`, champ par champ, et une faute de frappe dans une colonne
n'apparaît qu'à l'exécution.

Deux outils complémentaires industrialisent l'accès aux données :

- **[sqlc](https://sqlc.dev)** génère du code Go **typé** à partir de requêtes
  SQL ordinaires. Il analyse le schéma et les requêtes **à la génération** :
  une colonne inexistante ou un type incompatible est une erreur avant même la
  compilation ;
- **[golang-migrate](https://github.com/golang-migrate/migrate)** applique les
  **migrations** de schéma, versionnées et réversibles, depuis la ligne de
  commande ou depuis le programme lui-même.

Ce chapitre les combine dans un exemple complet : un catalogue de livres sur
[PostgreSQL](https://www.postgresql.org). Le code est dans `examples/approfondissements/catalogue`.

## sqlc : le SQL d'abord

La plupart des ORM ([GORM](https://gorm.io), ent…) génèrent le SQL à partir du code Go. sqlc fait
l'inverse : on écrit du **vrai SQL**, et sqlc génère le code Go qui l'appelle.
On garde toute la puissance de PostgreSQL (jointures, `RETURNING`,
`ON CONFLICT`, fonctions, CTE), sans réflexion à l'exécution ni requête
cachée : le code généré est lisible, et aussi rapide que du code écrit à la
main.

### L'organisation

```
catalogue/
├── cmd/catalogue/main.go          migre la base, puis l'utilise
├── internal/
│   ├── db/
│   │   ├── migrations/            migrations golang-migrate (le schéma)
│   │   │   ├── 000001_auteurs_livres.up.sql
│   │   │   ├── 000001_auteurs_livres.down.sql
│   │   │   ├── 000002_stock.up.sql
│   │   │   └── 000002_stock.down.sql
│   │   ├── requetes.sql           les requêtes, écrites à la main
│   │   ├── *.gen.go               le code généré par sqlc
│   │   └── migrer.go              les migrations embarquées
│   └── catalogue/                 la logique métier et les transactions
└── sqlc.yaml                      la configuration de sqlc
```

Le schéma n'est écrit qu'**une fois**, dans les migrations : sqlc les lit pour
connaître les tables.

### Le schéma : les migrations

golang-migrate attend des paires de fichiers numérotés : `NNNNNN_nom.up.sql`
fait évoluer le schéma, `NNNNNN_nom.down.sql` défait cette évolution.

```sql
-- 000001_auteurs_livres.up.sql
CREATE TABLE auteurs (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nom  TEXT   NOT NULL UNIQUE
);

CREATE TABLE livres (
    id        BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    isbn      TEXT        NOT NULL UNIQUE,
    titre     TEXT        NOT NULL,
    auteur_id BIGINT      NOT NULL REFERENCES auteurs (id),
    annee     INTEGER,    -- facultative : NULL si inconnue
    ajout     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX livres_auteur_idx ON livres (auteur_id);
```

```sql
-- 000002_stock.up.sql
ALTER TABLE livres
    ADD COLUMN exemplaires INTEGER NOT NULL DEFAULT 1 CHECK (exemplaires >= 0);

-- 000002_stock.down.sql
ALTER TABLE livres DROP COLUMN exemplaires;
```

Une migration publiée n'est **jamais modifiée** : pour corriger le schéma, on
ajoute une nouvelle migration.

### Les requêtes

Chaque requête est précédée d'un commentaire qui lui donne un **nom** (celui
de la méthode Go) et un **type de résultat** :

| Annotation  | Méthode générée                                        |
|-------------|--------------------------------------------------------|
| `:one`      | renvoie une ligne (ou `pgx.ErrNoRows`)                 |
| `:many`     | renvoie une slice                                      |
| `:exec`     | ne renvoie qu'une erreur                               |
| `:execrows` | renvoie le nombre de lignes modifiées                  |
| `:copyfrom` | insertion en masse avec le protocole `COPY`            |

```sql
-- name: AssureAuteur :one
-- Crée l'auteur, ou renvoie celui qui porte déjà ce nom.
INSERT INTO auteurs (nom) VALUES ($1)
ON CONFLICT (nom) DO UPDATE SET nom = EXCLUDED.nom
RETURNING *;

-- name: LivresAvecAuteur :many
SELECT sqlc.embed(livres), auteurs.nom AS auteur
FROM livres
JOIN auteurs ON auteurs.id = livres.auteur_id
ORDER BY auteurs.nom, livres.titre;

-- name: RechercheLivres :many
-- Recherche par titre, et éventuellement par année minimale.
SELECT * FROM livres
WHERE titre ILIKE '%' || sqlc.arg(texte)::text || '%'
  AND (sqlc.narg(annee_min)::integer IS NULL OR annee >= sqlc.narg(annee_min))
ORDER BY titre
LIMIT sqlc.arg(limite);

-- name: PrendExemplaire :execrows
-- Décrémente le stock, seulement s'il reste un exemplaire.
UPDATE livres SET exemplaires = exemplaires - 1
WHERE isbn = $1 AND exemplaires > 0;
```

Quelques fonctions de sqlc enrichissent le SQL :

- **`sqlc.arg(nom)`** nomme un paramètre : il devient un champ `Nom` de la
  structure de paramètres, au lieu de `Column1` ;
- **`sqlc.narg(nom)`** déclare un paramètre qui peut être `NULL` : un filtre
  facultatif ;
- **`sqlc.embed(table)`** regroupe les colonnes d'une table dans un champ de
  type structure, pratique pour les jointures.

### La configuration

```yaml
version: "2"
sql:
  - engine: "postgresql"
    # sqlc lit les migrations golang-migrate (fichiers *.up.sql) pour connaître le schéma.
    schema: "internal/db/migrations"
    queries: "internal/db/requetes.sql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        # Colonnes NULL : *int32 plutôt que pgtype.Int4.
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
        output_files_suffix: ".gen"
```

- **`sql_package: "pgx/v5"`** génère du code pour le pilote natif [`pgx`](https://github.com/jackc/pgx), plus
  rapide et plus complet que `database/sql` avec PostgreSQL. sqlc sait aussi
  générer pour `database/sql`, [MySQL](https://www.mysql.com) et [SQLite](https://sqlite.org) ;
- **`emit_pointers_for_null_types`** représente une colonne qui peut être
  `NULL` par un pointeur (`*int32`), plus idiomatique que les types
  `pgtype.Int4` ;
- **`overrides`** choisit le type Go d'un type SQL, ici `time.Time` pour
  `timestamptz`.

### Générer le code

```bash
sqlc generate     # régénère internal/db/*.gen.go
sqlc vet          # vérifie les requêtes
```

L'analyseur PostgreSQL de sqlc est une bibliothèque C : `go install
github.com/sqlc-dev/sqlc/cmd/sqlc@latest` demande donc un compilateur C. Sans
compilateur, l'image officielle fait le même travail, avec [Docker](https://www.docker.com) comme avec
[Podman](https://podman.io) :

```bash
docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 generate
```

Dans le dépôt du livre, `make sqlc` lance cette commande. Le code généré est
**versionné** : on n'a pas besoin de sqlc pour compiler le projet. Dans
l'intégration continue, une étape `sqlc generate` suivie de
`git diff --exit-code` vérifie que le code généré est à jour.

### Le code généré

sqlc produit trois fichiers. Le premier contient les **modèles**, un par
table :

```go
type Livre struct {
	ID          int64     `json:"id"`
	Isbn        string    `json:"isbn"`
	Titre       string    `json:"titre"`
	AuteurID    int64     `json:"auteur_id"`
	Annee       *int32    `json:"annee"`
	Ajout       time.Time `json:"ajout"`
	Exemplaires int32     `json:"exemplaires"`
}
```

Le deuxième définit le type `Queries`, qui accepte un pool de connexions **ou**
une transaction, grâce à une petite interface :

```go
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func (q *Queries) WithTx(tx pgx.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}
```

Le troisième contient une méthode par requête, avec ses paramètres et son
résultat typés. C'est exactement le code qu'on aurait écrit à la main :

```go
// Décrémente le stock, seulement s'il reste un exemplaire.
func (q *Queries) PrendExemplaire(ctx context.Context, isbn string) (int64, error) {
	result, err := q.db.Exec(ctx, prendExemplaire, isbn)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

type RechercheLivresParams struct {
	Texte    string `json:"texte"`
	AnneeMin *int32 `json:"annee_min"`
	Limite   int32  `json:"limite"`
}

type LivresAvecAuteurRow struct {
	Livre  Livre  `json:"livre"` // grâce à sqlc.embed(livres)
	Auteur string `json:"auteur"`
}
```

Si une requête mentionne une colonne qui n'existe pas, ou compare un texte à
un entier, `sqlc generate` échoue avec un message qui indique la ligne
fautive : l'erreur est détectée sur le poste du développeur, et non en
production.

## golang-migrate : des migrations embarquées

golang-migrate s'utilise en ligne de commande, ou comme **bibliothèque**. Dans
le second cas, le programme met à jour son propre schéma au démarrage : il
suffit de déployer l'exécutable, qui embarque ses migrations avec `embed`.

```go
import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pilote PostgreSQL, schéma « pgx5:// »
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

func NouveauMigrateur(dsn string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	// Le pilote pgx v5 de golang-migrate s'enregistre sous le schéma « pgx5 ».
	url := "pgx5://" + strings.TrimPrefix(strings.TrimPrefix(dsn, "postgres://"), "postgresql://")
	return migrate.NewWithSourceInstance("iofs", source, url)
}

func Migre(dsn string) (uint, error) {
	m, err := NouveauMigrateur(dsn)
	if err != nil {
		return 0, err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, fmt.Errorf("migrations : %w", err)
	}
	version, sale, err := m.Version()
	if err != nil {
		return 0, err
	}
	if sale {
		return version, fmt.Errorf("schéma dans un état incohérent à la version %d", version)
	}
	return version, nil
}
```

golang-migrate enregistre la version courante dans une table
`schema_migrations`, et pose un verrou pendant la migration : plusieurs
instances du service qui démarrent en même temps ne migrent pas la base deux
fois.

L'état « sale » (*dirty*) signale une migration interrompue par une erreur :
le schéma est peut-être à moitié modifié. Le programme refuse alors de
démarrer, et une intervention manuelle est nécessaire : corriger la base, puis
indiquer la bonne version avec `migrate force`.

### La ligne de commande

La même commande existe en image Docker, pratique en développement ou dans un
pipeline de déploiement :

```bash
# Créer une nouvelle paire de fichiers de migration
docker run --rm -v "$PWD/internal/db/migrations:/migrations" migrate/migrate:v4.20.1 \
  create -ext sql -dir /migrations -seq ajoute_editeur

# Appliquer, annuler la dernière, consulter la version
migrate -path internal/db/migrations -database "$DSN" up
migrate -path internal/db/migrations -database "$DSN" down 1
migrate -path internal/db/migrations -database "$DSN" version
```

## Utiliser le code généré

### Le programme principal

```go
func run(ctx context.Context, dsn string) error {
	// 1. Le programme met son schéma à jour lui-même, au démarrage.
	version, err := db.Migre(dsn)
	if err != nil {
		return err
	}
	slog.Info("schéma à jour", "version", version)

	// 2. Un pool de connexions pgx, partagé par tout le programme.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	stats, err := catalogue.Nouveau(pool).Requetes().StatistiquesParAuteur(ctx)
	…
}
```

### Les transactions

Ajouter un livre demande deux écritures : l'auteur, puis le livre. Elles
doivent réussir ou échouer **ensemble**. `WithTx` lie les requêtes générées à
une transaction :

```go
func (c *Catalogue) Ajoute(ctx context.Context, l NouveauLivre) (db.Livre, error) {
	var livre db.Livre
	err := c.transaction(ctx, func(q *db.Queries) error {
		auteur, err := q.AssureAuteur(ctx, l.Auteur)
		if err != nil {
			return fmt.Errorf("auteur %q : %w", l.Auteur, err)
		}
		livre, err = q.CreeLivre(ctx, db.CreeLivreParams{
			Isbn: l.ISBN, Titre: l.Titre, AuteurID: auteur.ID,
			Annee: l.Annee, Exemplaires: l.Exemplaires,
		})
		if violeUnicite(err) {
			return ErrISBNDejaUtilise
		}
		return err
	})
	return livre, err
}

func (c *Catalogue) transaction(ctx context.Context, f func(*db.Queries) error) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // sans effet après un Commit réussi

	if err := f(c.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
```

Les erreurs de PostgreSQL sont traduites en erreurs métier en examinant leur
code (`23505` : violation d'unicité) :

```go
func violeUnicite(err error) bool {
	e, ok := errors.AsType[*pgconn.PgError](err)
	return ok && e.Code == "23505"
}
```

### Les colonnes facultatives

Avec `emit_pointers_for_null_types`, une colonne `NULL` est un pointeur. La
forme `new(expr)` de **Go 1.26** rend ces valeurs faciles à écrire :

```go
c.Ajoute(ctx, NouveauLivre{
	ISBN:        "978-0-13-419044-0",
	Titre:       "The Go Programming Language",
	Auteur:      "Alan Donovan",
	Annee:       new(int32(2015)),
	Exemplaires: 2,
})

recents, err := c.Recherche(ctx, "go", new(int32(2016)))  // à partir de 2016
tous, err := c.Recherche(ctx, "go", nil)                   // sans filtre d'année
```

## Tester avec Testcontainers

Le code généré par sqlc est du SQL PostgreSQL : on le teste contre un **vrai**
PostgreSQL, démarré par [Testcontainers](https://golang.testcontainers.org) (voir le chapitre sur le package
`testing`). Les migrations sont appliquées exactement comme en production :

```go
func TestPostgresCatalogue(t *testing.T) {
	dsn := basePostgres(t) // conteneur jetable, ignoré avec -short
	version, err := db.Migre(dsn)
	if err != nil || version != 2 {
		t.Fatalf("Migre = %d, %v ; attendu la version 2", version, err)
	}
	…
	t.Run("ISBN unique, sans effet partiel", func(t *testing.T) {
		_, err := c.Ajoute(ctx, NouveauLivre{ISBN: livres[0].ISBN, Titre: "Doublon", Auteur: "Nouvel auteur"})
		if !errors.Is(err, ErrISBNDejaUtilise) {
			t.Fatalf("erreur = %v, attendu ErrISBNDejaUtilise", err)
		}
		// La transaction a été annulée : « Nouvel auteur » n'existe pas.
		…
	})
}
```

Un second test vérifie que les migrations sont **réversibles** : il les
applique, les annule toutes, puis les réapplique. Une migration `down`
oubliée ou incorrecte est ainsi détectée avant le jour où l'on en aurait
besoin.

```go
func TestPostgresMigrationsReversibles(t *testing.T) {
	m, err := db.NouveauMigrateur(basePostgres(t))
	…
	if err := m.Up(); err != nil {
		t.Fatalf("up : %v", err)
	}
	if err := m.Down(); err != nil {
		t.Fatalf("down : %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up après down : %v", err)
	}
}
```

Enfin, un test sans base de données vérifie à chaque `go test` que chaque
fichier de migration a un nom conforme et sa migration inverse.

## Quand utiliser sqlc ?

| Approche              | Points forts                                   | Limites                              |
|-----------------------|------------------------------------------------|--------------------------------------|
| `database/sql` à la main | aucune dépendance, contrôle total           | répétitif, erreurs à l'exécution     |
| **sqlc**              | SQL complet, code typé et vérifié, performant  | requêtes dynamiques peu pratiques    |
| ORM (GORM, ent…)      | requêtes dynamiques, relations automatiques    | SQL caché, réflexion, performances   |

sqlc convient à la grande majorité des services : les requêtes y sont connues
à l'avance. Pour les rares requêtes réellement dynamiques (un moteur de
recherche aux filtres multiples, par exemple), un constructeur de requêtes
comme [`squirrel`](https://github.com/Masterminds/squirrel), ou du SQL écrit à la main, complète sqlc sans difficulté :
les deux utilisent le même pool de connexions.
