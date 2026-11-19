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
avec `reflect.DeepEqual`, ou avec `github.com/google/go-cmp`, qui affiche les
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
utilisez `benchstat` (`golang.org/x/perf/cmd/benchstat`).

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
