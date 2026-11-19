---
title: "Tests : testing"
weight: 90
---

Les tests font partie de l'outillage standard : pas de framework à choisir ni à
installer. Un fichier `xxx_test.go` contient des fonctions de test, et
`go test` les compile et les exécute.

```bash
go test ./...                   # tous les tests du module
go test -run TestSlug ./...     # les tests dont le nom correspond
go test -v ./...                # affichage détaillé
go test -count=1 ./...          # sans le cache des résultats
go test -race ./...             # avec le détecteur de courses
go test -cover ./...            # avec le taux de couverture
```

## Les tests unitaires

Une fonction de test s'appelle `TestXxx` et reçoit un `*testing.T` :

```go
func TestSlug(t *testing.T) {
	if got := Slug("Le gopher creuse"); got != "le-gopher-creuse" {
		t.Errorf("Slug() = %q, attendu %q", got, "le-gopher-creuse")
	}
}
```

| Méthode                  | Effet                                                    |
|--------------------------|----------------------------------------------------------|
| `t.Errorf`, `t.Error`    | Signale un échec et **continue** le test                 |
| `t.Fatalf`, `t.Fatal`    | Signale un échec et **arrête** le test                   |
| `t.Logf`                 | Message affiché si le test échoue (ou avec `-v`)         |
| `t.Skip`                 | Ignore le test (prérequis absent, test long…)            |
| `t.Helper()`             | Marque une fonction d'aide : l'erreur pointe vers l'appelant |
| `t.Cleanup(f)`           | Exécute `f` à la fin du test                             |
| `t.TempDir()`            | Répertoire temporaire supprimé à la fin du test          |
| `t.Setenv(k, v)`         | Définit une variable d'environnement le temps du test    |
| `t.Context()`            | Contexte annulé à la fin du test (Go 1.24)               |
| `t.Chdir(dir)`           | Change de répertoire le temps du test (Go 1.24)          |
| `t.ArtifactDir()`        | Répertoire pour les fichiers produits par le test (Go 1.26) |

Le style Go privilégie des messages explicites (`got … attendu …`) plutôt que
des bibliothèques d'assertions. La comparaison de structures complexes se fait
avec `reflect.DeepEqual`, ou avec [`github.com/google/go-cmp`](https://github.com/google/go-cmp), qui affiche les
différences.

## Les tests de table

La forme idiomatique d'un test Go est la **table** : une slice de cas, et une
boucle qui crée un **sous-test** par cas avec `t.Run`.

```go
func TestSlug(t *testing.T) {
	cas := []struct {
		nom     string
		entree  string
		attendu string
	}{
		{"simple", "Bonjour", "bonjour"},
		{"espaces", "Le gopher creuse", "le-gopher-creuse"},
		{"ponctuation", "Go 1.27 !", "go-1-27"},
		{"accents ignorés", "Été à Paris", "t-paris"},
		{"vide", "", ""},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			t.Parallel()
			if got := Slug(c.entree); got != c.attendu {
				t.Errorf("Slug(%q) = %q, attendu %q", c.entree, got, c.attendu)
			}
		})
	}
}
```

Ajouter un cas est une ligne. Chaque sous-test a un nom
(`TestSlug/ponctuation`), et peut être exécuté seul :
`go test -run 'TestSlug/ponctuation'`. `t.Parallel()` exécute les sous-tests
en parallèle.

Le cas « accents ignorés » documente une limite de l'implémentation : les
tests servent aussi de spécification.

## Les exemples

Une fonction `ExampleXxx` est à la fois un test et de la documentation. `go
test` compare sa sortie au commentaire `// Output:`, et `go doc` l'affiche avec
la documentation de `Xxx` :

```go
func ExampleSlug() {
	fmt.Println(Slug("Golang Demystified : le livre"))
	// Output: golang-demystified-le-livre
}
```

`// Unordered output:` accepte les lignes dans n'importe quel ordre. Tous les
exemples de ce livre sont écrits ainsi.

## Les benchmarks

Une fonction `BenchmarkXxx` mesure les performances :

```go
func BenchmarkSlug(b *testing.B) {
	titre := strings.Repeat("Le gopher creuse ", 10)
	for b.Loop() {
		Slug(titre)
	}
}
```

```bash
$ go test -run '^$' -bench . -benchmem ./...
BenchmarkSlug-16    	  651722	      2105 ns/op	     680 B/op	       7 allocs/op
```

> [!NOTE]
> **Go 1.24** — `b.Loop()` remplace la boucle `for i := 0; i < b.N; i++`. La
> préparation située avant la boucle n'est exécutée qu'une fois et n'est pas
> mesurée, et le compilateur ne peut pas éliminer le code du corps de la
> boucle.

`-benchmem` affiche les allocations, souvent la première piste
d'optimisation. Pour comparer deux versions de manière statistiquement fiable,
utilisez [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) (`golang.org/x/perf/cmd/benchstat`).

## Le fuzzing

> [!NOTE]
> **Go 1.18** — le fuzzing est intégré à `go test`.

Le fuzzing génère automatiquement des entrées pour chercher les cas qui font
échouer un test. On vérifie des **propriétés** qui doivent toujours être
vraies, plutôt que des résultats précis :

```go
func FuzzSlug(f *testing.F) {
	f.Add("Go 1.27 !")        // corpus initial
	f.Add("Été à Paris")
	f.Fuzz(func(t *testing.T, titre string) {
		s := Slug(titre)
		if !utf8.ValidString(s) {
			t.Fatalf("slug invalide : %q", s)
		}
		if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
			t.Errorf("Slug(%q) = %q : tiret mal placé", titre, s)
		}
	})
}
```

```bash
go test -fuzz FuzzSlug -fuzztime 30s ./...
```

Sans l'option `-fuzz`, `go test` exécute seulement le corpus initial, comme un
test ordinaire. Lorsqu'une entrée fait échouer le test, elle est enregistrée
dans `testdata/fuzz/FuzzSlug/` et rejouée à chaque `go test` : elle devient un
test de non-régression.

## Tester le temps et la concurrence : synctest

> [!NOTE]
> **Go 1.25** — le package `testing/synctest` exécute un test dans une
> **bulle** dont l'horloge est virtuelle.

Tester un délai d'une heure ne doit pas prendre une heure. Dans une bulle,
le temps n'avance que lorsque toutes les goroutines de la bulle sont
bloquées : il saute alors directement au prochain événement.

```go
func TestAttendAvecDelai(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		travail := make(chan struct{}) // jamais terminé
		debut := time.Now()

		err := AttendAvecDelai(t.Context(), travail, time.Hour)

		if !errors.Is(err, ErrDelai) {
			t.Fatalf("erreur = %v, attendu ErrDelai", err)
		}
		if ecoule := time.Since(debut); ecoule != time.Hour {
			t.Errorf("écoulé = %v, attendu 1h", ecoule)
		}
	})
}
```

Ce test s'exécute instantanément, et le temps écoulé vaut **exactement** une
heure. `synctest.Wait()` attend que toutes les goroutines de la bulle soient
bloquées, ce qui permet de vérifier un état intermédiaire sans `time.Sleep`.
Depuis Go 1.27, `synctest.Sleep` combine les deux.

## La couverture

```bash
go test -coverprofile=couverture.out ./...
go tool cover -html=couverture.out   # rapport dans le navigateur
go tool cover -func=couverture.out   # couverture par fonction
```

Depuis **Go 1.20**, `go build -cover` produit aussi un exécutable instrumenté,
pour mesurer la couverture de tests d'intégration ou de bout en bout.

## Tests d'intégration avec Testcontainers

Un double de test ou une base [SQLite](https://sqlite.org) en mémoire ne disent pas tout : le code
qui parle à [PostgreSQL](https://www.postgresql.org), [Redis](https://redis.io) ou [Kafka](https://kafka.apache.org) doit aussi être testé contre le **vrai**
logiciel, dans la **vraie** version utilisée en production. Une différence de
syntaxe SQL, une contrainte, un type de colonne : autant de bugs qu'aucun
double ne révèle.

[Testcontainers](https://golang.testcontainers.org) répond à ce besoin. Le
module `github.com/testcontainers/testcontainers-go` démarre, depuis le test
lui-même, un conteneur jetable ([Docker](https://www.docker.com) ou [Podman](https://podman.io)), attend qu'il soit prêt,
fournit son adresse, puis le supprime à la fin du test. Il n'y a ni base
partagée à maintenir, ni fichier `docker-compose` à lancer avant les tests.

### Un exemple : le stockage de l'API

Le stockage de l'API du chapitre « Une API web complète » fonctionne avec
SQLite et avec PostgreSQL. Ses tests de comportement sont regroupés dans une
**suite** qui reçoit le stockage à tester :

```go
// suite regroupe les tests de comportement du stockage. Elle s'exécute à
// l'identique sur chaque base.
func suite(t *testing.T, s *Store) {
	t.Run("liste vide", func(t *testing.T) { … })
	t.Run("cycle de vie", func(t *testing.T) { … })
	t.Run("migrations idempotentes", func(t *testing.T) { … })
}

func TestSQLite(t *testing.T) {
	suite(t, ouvre(t, ":memory:")) // rapide, toujours disponible
}
```

Le test PostgreSQL démarre un conteneur grâce au module `postgres` de
Testcontainers, et exécute **la même suite** :

```go
import (
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t) // ni Docker ni Podman : ignoré

	ctx := t.Context()
	conteneur, err := tcpostgres.Run(ctx, "docker.io/postgres:17-alpine",
		tcpostgres.WithDatabase("taches"),
		tcpostgres.WithUsername("gopher"),
		tcpostgres.WithPassword("secret"),
		tcpostgres.BasicWaitStrategies(), // attend que la base accepte les connexions
	)
	testcontainers.CleanupContainer(t, conteneur) // supprimé à la fin du test
	if err != nil {
		t.Fatal(err)
	}

	dsn, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	suite(t, ouvre(t, dsn))
}
```

```
$ go test -run TestPostgres -v ./internal/store
🔔 Container is ready: acbd218c7c05
    postgres_test.go:37: PostgreSQL : postgres://gopher:secret@localhost:41975/taches?sslmode=disable
=== RUN   TestPostgres/liste_vide
=== RUN   TestPostgres/cycle_de_vie
=== RUN   TestPostgres/migrations_idempotentes
--- PASS: TestPostgres/liste_vide (0.00s)
--- PASS: TestPostgres/cycle_de_vie (0.00s)
--- PASS: TestPostgres/migrations_idempotentes (0.00s)
```

Quelques points à retenir :

- **`CleanupContainer`** est appelé juste après `Run`, **avant** le test de
  l'erreur : même un conteneur partiellement démarré est supprimé.
- **Les stratégies d'attente** (`BasicWaitStrategies`, ou `wait.ForLog`,
  `wait.ForListeningPort`, `wait.ForHTTP` pour d'autres images) évitent les
  tests qui échouent parce que le service n'était pas encore prêt.
- **Le port est aléatoire** : plusieurs tests, ou plusieurs exécutions en
  parallèle dans l'intégration continue, ne se gênent pas.
- **L'image est versionnée** : le test utilise la même version de PostgreSQL
  que la production.
- **Le test se saute proprement** avec `-short`, ou si aucun moteur de
  conteneurs n'est disponible. `go test -short ./...` reste rapide pour le
  développement quotidien ; les tests d'intégration tournent avant chaque
  fusion.

> [!TIP]
> Démarrer un conteneur prend quelques secondes. Pour de nombreux tests,
> démarrez-le **une seule fois** dans `TestMain`, et isolez les tests entre eux :
> une base par test, une transaction annulée à la fin de chaque test, ou les
> fonctions `Snapshot` et `Restore` du module PostgreSQL, qui sauvegardent et
> restaurent l'état de la base.

### Le moteur de conteneurs : Docker ou Podman

Testcontainers dialogue avec le moteur de conteneurs à travers l'**API
Docker**. Il fonctionne donc avec Docker, mais aussi avec Podman, qui propose
une API compatible. Il utilise aussi un petit conteneur, « Ryuk », chargé de
supprimer les conteneurs orphelins si un test est interrompu brutalement.

#### Avec Docker

Avec **Docker Desktop** (macOS, Windows) ou **Docker Engine** (Linux), il n'y a
rien à configurer : Testcontainers trouve le moteur tout seul.

```bash
docker info                     # vérifie que le moteur répond
go test -run TestPostgres ./...
```

Quelques cas particuliers :

- sur Linux, l'utilisateur doit pouvoir accéder au moteur sans `sudo`, en
  appartenant au groupe `docker` :

  ```bash
  sudo usermod -aG docker $USER   # puis se reconnecter
  ```

- avec Docker en mode *rootless*, indiquez l'emplacement de son socket :

  ```bash
  export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/docker.sock
  ```

- avec **[Colima](https://github.com/abiosoft/colima)** (macOS), indiquez le socket de la machine virtuelle, et
  celui que Ryuk doit monter :

  ```bash
  export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
  export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
  ```

- dans **[GitHub Actions](https://docs.github.com/actions)**, les machines `ubuntu-latest` disposent de Docker :
  les tests d'intégration fonctionnent sans configuration.

#### Avec Podman

Podman n'a pas de démon : son API compatible Docker est fournie par un service
à activer. En mode *rootless*, Ryuk ne fonctionne pas : on le désactive, et
c'est `CleanupContainer` qui fait le ménage.

```bash
systemctl --user enable --now podman.socket
export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock
export TESTCONTAINERS_RYUK_DISABLED=true
go test -run TestPostgres ./...
```

Sur macOS, avec `podman machine`, la commande
`podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}'`
donne le chemin du socket à placer dans `DOCKER_HOST`.

| Variable                                  | Rôle                                            |
|-------------------------------------------|-------------------------------------------------|
| `DOCKER_HOST`                             | adresse de l'API du moteur, si elle n'est pas celle par défaut |
| `TESTCONTAINERS_RYUK_DISABLED`            | désactive Ryuk (Podman *rootless*)              |
| `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE`   | socket à monter dans Ryuk (Colima)              |

Dans le dépôt du livre, `make test-integration` fonctionne avec les deux
moteurs : avec Docker, il n'y a rien à faire ; avec Podman, il définit ces
variables automatiquement.

Testcontainers propose des modules prêts à l'emploi pour la plupart des
logiciels courants : [MySQL](https://www.mysql.com), [MongoDB](https://www.mongodb.com), Redis, Kafka, [RabbitMQ](https://www.rabbitmq.com), [Elasticsearch](https://www.elastic.co/elasticsearch),
[LocalStack](https://www.localstack.cloud) (services AWS)… et `testcontainers.GenericContainer` pour n'importe
quelle image.

## Organiser les tests

- Un fichier de test peut déclarer le package testé (`package tests`, accès aux
  éléments privés) ou le package externe `package tests_test`, qui ne voit que
  l'API publique, comme un utilisateur.
- Le répertoire `testdata/` est ignoré par l'outil `go` : on y range les
  fichiers d'entrée et les résultats attendus.
- `func TestMain(m *testing.M)` permet une préparation globale (démarrer une
  base de données…), avant d'appeler `m.Run()`.
- `go test -short` permet de sauter les tests longs, qui testent
  `testing.Short()`.
- Pour les dépendances externes, injectez une **interface** et fournissez un
  double de test (*fake*) écrit à la main : c'est souvent plus simple qu'une
  bibliothèque de *mocks*.
