---
title: "V.2 Concurrence en pratique"
description: "pipeline, fan-out et fan-in, pool de workers, errgroup, sémaphore et singleflight."
weight: 20
---

# Chapitre V.2 — Concurrence en pratique

Le chapitre « Goroutines et channels » a présenté les briques de base. Ce
chapitre les assemble en **motifs** éprouvés, que l'on retrouve dans la plupart
des programmes concurrents : pipelines, pools de workers, groupes d'erreurs,
limitation de débit. Il montre aussi comment tester ce code et détecter les
fuites de goroutines.

Tout le code de ce chapitre est dans [`examples/approfondissements/concurrence`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/concurrence).

## Trois règles d'or

Avant les motifs, trois règles qui évitent l'essentiel des bugs :

1. **Toute goroutine doit pouvoir se terminer.** Avant d'écrire `go`, sachez
   comment et quand la goroutine s'arrêtera. Une goroutine bloquée pour
   toujours est une fuite de mémoire.
2. **Celui qui écrit dans un channel le ferme.** Jamais le lecteur, et jamais
   deux fois.
3. **Toute opération bloquante respecte l'annulation.** Un envoi ou une
   réception de channel se fait dans un `select` avec `ctx.Done()`.

## Le pipeline

Un pipeline est une chaîne d'**étages** reliés par des channels. Chaque étage
lit son entrée, traite les valeurs et écrit sa sortie, dans sa propre
goroutine. Les étages travaillent en parallèle, comme une chaîne de montage.

Le premier étage produit les valeurs :

```go
// Genere émet les nombres de 1 à n, et s'arrête si ctx est annulé.
func Genere(ctx context.Context, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 1; i <= n; i++ {
			select {
			case out <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
```

Un étage générique applique une fonction à chaque valeur :

```go
// Etape applique f à chaque valeur reçue : c'est un étage de pipeline.
func Etape[E, S any](ctx context.Context, in <-chan E, f func(E) S) <-chan S {
	out := make(chan S)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- f(v):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
```

Les étages s'assemblent naturellement :

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

nombres := Genere(ctx, 5)
carres := Etape(ctx, nombres, func(n int) int { return n * n })
textes := Etape(ctx, carres, func(n int) string { return fmt.Sprint(n) })

for t := range textes {
	fmt.Print(t, " ")
}
// 1 4 9 16 25
```

Chaque étage respecte les trois règles : il ferme son channel de sortie
(`defer close(out)`), il s'arrête quand son entrée est fermée, et chaque envoi
surveille l'annulation. Le consommateur peut donc s'arrêter à tout moment
sans laisser de goroutine bloquée :

```go
ctx, cancel := context.WithCancel(context.Background())
somme := 0
for n := range Genere(ctx, 1_000_000) {
	somme += n
	if n == 10 {
		cancel() // le générateur s'arrête au prochain envoi
		break
	}
}
// somme vaut 55
```

Sans le `cancel()`, le générateur resterait bloqué sur son envoi pour
toujours.

> [!TIP]
> Pour un traitement séquentiel en mémoire, préférez les **itérateurs** (Go
> 1.23) : ils sont plus simples et plus rapides. Le pipeline à base de channels
> se justifie lorsque les étages font des entrées-sorties ou des calculs
> coûteux qui gagnent à s'exécuter en parallèle.

## Fan-out et fan-in

- Le **fan-out** répartit le travail d'un channel entre plusieurs goroutines,
  pour paralléliser un étage lent.
- Le **fan-in** fusionne plusieurs channels en un seul.

```go
// Fusionne regroupe plusieurs channels en un seul (fan-in).
func Fusionne[T any](ctx context.Context, entrees ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	for _, in := range entrees {
		wg.Go(func() {
			for v := range in {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	go func() {
		wg.Wait() // quand toutes les entrées sont épuisées…
		close(out) // … on ferme la sortie
	}()
	return out
}
```

La goroutine qui attend `wg.Wait()` avant de fermer la sortie est le motif
classique : on ne peut fermer un channel partagé par plusieurs écrivains que
lorsqu'ils ont **tous** terminé.

## Le pool de workers

Le pool de workers combine les deux : un nombre fixe de workers consomment
des tâches depuis un channel (fan-out) et écrivent leurs résultats dans un
channel commun (fan-in). Il borne le parallélisme, ce qui protège les
ressources : connexions à une base de données, appels à une API limitée,
mémoire.

```go
type Resultat[E, S any] struct {
	Entree E
	Sortie S
	Err    error
}

func Pool[E, S any](ctx context.Context, n int, taches []E,
	f func(context.Context, E) (S, error)) <-chan Resultat[E, S] {

	jobs := make(chan E)
	resultats := make(chan Resultat[E, S])

	var wg sync.WaitGroup
	for range n { // n workers
		wg.Go(func() {
			for t := range jobs {
				s, err := f(ctx, t)
				select {
				case resultats <- Resultat[E, S]{t, s, err}:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	go func() { // distribue les tâches
		defer close(jobs)
		for _, t := range taches {
			select {
			case jobs <- t:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() { // ferme les résultats quand tous les workers ont fini
		wg.Wait()
		close(resultats)
	}()
	return resultats
}
```

Chaque résultat porte sa tâche et son éventuelle erreur : une tâche en échec
n'interrompt pas les autres, et l'appelant décide quoi faire.

```go
for r := range Pool(ctx, 3, urls, telecharge) {
	if r.Err != nil {
		log.Printf("%s : %v", r.Entree, r.Err)
		continue
	}
	…
}
```

Les résultats arrivent dans l'ordre où ils sont **produits**, pas dans l'ordre
des tâches. Pour conserver l'ordre, écrivez chaque résultat à l'indice de sa
tâche dans une slice préallouée.

## errgroup : un groupe de tâches qui peuvent échouer

Lorsque l'échec d'une tâche doit **annuler toutes les autres**, le package
[`golang.org/x/sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup) est l'outil idéal. Maintenu par l'équipe Go, il
remplace avantageusement un `WaitGroup` accompagné d'une gestion d'erreurs
manuelle :

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(context.Background())
g.SetLimit(2) // au plus deux tâches simultanées

for i := range 6 {
	g.Go(func() error {
		if i == 2 {
			return fmt.Errorf("tâche %d : échec", i)
		}
		select {
		case <-time.After(50 * time.Millisecond):
			return nil
		case <-ctx.Done(): // l'échec de la tâche 2 annule les autres
			return ctx.Err()
		}
	})
}
err := g.Wait()
// tâche 2 : échec
```

- `g.Wait()` attend toutes les tâches et renvoie la **première** erreur ;
- le contexte renvoyé par `WithContext` est annulé dès la première erreur ;
- `SetLimit(n)` borne le parallélisme : `g.Go` bloque tant que `n` tâches sont
  en cours. C'est un pool de workers en une ligne.

Un usage typique : charger en parallèle les données d'une page (profil,
commandes, recommandations) et échouer dès que l'une manque.

## Limiter le parallélisme : le sémaphore

Un channel bufferisé de taille `n` est un **sémaphore** : on y écrit pour
prendre une place, et on lit pour la libérer.

```go
sem := make(chan struct{}, 3)
for _, f := range fichiers {
	wg.Go(func() {
		sem <- struct{}{}        // acquiert une place (bloque si 3 sont prises)
		defer func() { <-sem }() // la libère
		traite(f)
	})
}
wg.Wait()
```

Le package [`golang.org/x/sync/semaphore`](https://pkg.go.dev/golang.org/x/sync/semaphore) propose un sémaphore pondéré, dont
l'acquisition respecte un contexte.

## Limiter le débit

Limiter le **débit**, c'est limiter le nombre d'opérations par unité de temps :
appeler une API qui accepte 10 requêtes par seconde, par exemple. Le package
[`golang.org/x/time/rate`](https://pkg.go.dev/golang.org/x/time/rate) implémente un *token bucket* :

```go
import "golang.org/x/time/rate"

limiteur := rate.NewLimiter(rate.Every(100*time.Millisecond), 1) // 10/s, rafale de 1
for _, req := range requetes {
	if err := limiteur.Wait(ctx); err != nil { // attend un jeton
		return err
	}
	envoie(req)
}
```

`limiteur.Allow()` répond immédiatement sans attendre, ce qui convient à un
serveur qui rejette les clients trop pressés avec un statut `429 Too Many
Requests`.

Avec la seule bibliothèque standard, un `time.Ticker` suffit pour un débit
régulier :

```go
tic := time.NewTicker(100 * time.Millisecond)
defer tic.Stop()
for _, req := range requetes {
	<-tic.C
	envoie(req)
}
```

### Tester sans attendre

Grâce à `testing/synctest` (Go 1.25), ces tests s'exécutent instantanément,
avec une horloge virtuelle et une précision exacte :

```go
func TestLimiteurDeDebit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiteur := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
		debut := time.Now()
		for range 20 {
			if err := limiteur.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		// Le premier jeton est disponible tout de suite, puis un toutes les 100 ms.
		if ecoule := time.Since(debut); ecoule != 1900*time.Millisecond {
			t.Errorf("durée = %v, attendu 1,9 s", ecoule)
		}
	})
}
```

## Dédoublonner des appels : singleflight

Lorsqu'une donnée coûteuse (configuration distante, entrée de cache expirée)
est demandée simultanément par de nombreuses goroutines, `singleflight`
n'exécute qu'**un seul** appel et partage son résultat avec toutes celles qui
attendent :

```go
import "golang.org/x/sync/singleflight"

var groupe singleflight.Group

v, err, partage := groupe.Do("config", func() (any, error) {
	return chargeConfig(ctx)
})
```

C'est la protection classique contre l'effet « troupeau » (*thundering herd*)
lorsqu'un cache expire sous forte charge.

## Détecter les fuites de goroutines

Une goroutine bloquée pour toujours ne sera jamais libérée, pas plus que la
mémoire qu'elle référence. Sur un serveur, ces fuites s'accumulent jusqu'à
épuiser la mémoire.

> [!NOTE]
> **Go 1.27** — le profil **`goroutineleak`** détecte les goroutines bloquées
> sur une opération qui ne pourra plus jamais aboutir, grâce au ramasse-miettes :
> si plus aucune autre goroutine ne peut atteindre le channel attendu, la
> goroutine est perdue.

```go
func fuit() {
	ch := make(chan int)
	go func() { <-ch }() // personne n'écrira jamais dans ch
}

fuit()
runtime.GC()
pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
```

```
goroutineleak profile: total 1
1 @ 0x47d94a 0x4151ee 0x414d32 0x4dee99 0x483681
#	0x4dee98	main.fuit.func1+0x18	main.go:13
```

Sur un serveur, le profil est exposé par `net/http/pprof`, à l'adresse
`/debug/pprof/goroutineleak`. Le chapitre « Profilage et performance » montre
comment l'utiliser.

Dans les tests, `synctest` détecte aussi les fuites : à la fin de
`synctest.Test`, toutes les goroutines de la bulle doivent être terminées,
sinon le test échoue.

```go
func TestPipelineSansFuite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		sortie := Etape(ctx, Genere(ctx, 1000), func(n int) int { return n * 2 })
		<-sortie
		cancel()
		synctest.Wait() // toutes les goroutines doivent pouvoir se terminer
	})
}
```

## Choisir le bon outil

| Besoin                                              | Outil                                  |
|-----------------------------------------------------|----------------------------------------|
| Attendre un ensemble de goroutines                  | `sync.WaitGroup` et `wg.Go`            |
| Attendre des tâches qui peuvent échouer, annuler les autres | `errgroup.Group`                |
| Borner le nombre de tâches simultanées              | `errgroup.SetLimit`, sémaphore         |
| Borner le nombre d'opérations par seconde           | `rate.Limiter`, `time.Ticker`          |
| Enchaîner des traitements parallèles                | pipeline de channels                   |
| Protéger une donnée partagée                        | `sync.Mutex`, `sync/atomic`            |
| Initialiser une seule fois                          | `sync.OnceValue`                       |
| Dédoublonner des appels simultanés                  | `singleflight.Group`                   |
| Annuler, imposer un délai                           | `context`                              |

Et dans tous les cas : `go test -race` dans l'intégration continue, et
`synctest` pour tester le code qui dépend du temps.

## Les bibliothèques de traitement de flux

Les motifs de ce chapitre suffisent à la plupart des besoins. Pour le
traitement de flux d'événements (journaux, mesures, messages), des
bibliothèques proposent des briques prêtes à l'emploi : fenêtres
temporelles, regroupements, limitation de débit, connecteurs vers [Kafka](https://kafka.apache.org) ou
[NATS](https://nats.io). Avant d'en adopter une, il faut savoir ce qu'elle apporte et ce qu'elle
coûte.

### go-streams

[go-streams](https://github.com/reugn/go-streams) décrit un pipeline comme
une **source**, des **étapes** (*flows*) et une **destination** (*sink*),
reliées par `Via` et `To` :

```go
import (
	"github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
)

entree := make(chan any)
sortie := make(chan any)

source := extension.NewChanSource(entree)
analyseur := flow.NewMap(analyse, 1) // parallélisme 1 : l'ordre est conservé
erreurs := flow.NewFilter(func(r Requete) bool { return r.Statut >= 500 }, 1)
lots := flow.NewBatch[Requete](2, time.Hour) // lots de 2, ou toutes les heures

go source.Via(analyseur).Via(erreurs).Via(lots).To(extension.NewChanSink(sortie))

for lot := range sortie {
	for _, r := range lot.([]Requete) { // la sortie est un chan any : assertion de type
		…
	}
}
```

Son point fort est un catalogue d'opérateurs délicats à bien écrire soi-même :
fenêtres glissantes, fixes ou de session, lots par taille ou par durée,
limitation de débit adaptée à la charge CPU et mémoire, regroupement par clé.
Compter les requêtes par seconde tient en une ligne, et se teste
instantanément avec `synctest`, car la bibliothèque utilise les timers de la
bibliothèque standard :

```go
func TestFenetreTemporelle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entree := make(chan any)
		sortie := make(chan any)
		fenetre := flow.NewTumblingWindow[Requete](time.Second)
		go extension.NewChanSource(entree).Via(fenetre).To(extension.NewChanSink(sortie))

		go func() {
			defer close(entree)
			for _, pause := range []time.Duration{0, 100, 200, 900, 50, 1000} {
				time.Sleep(pause * time.Millisecond)
				entree <- Requete{Chemin: "/", Statut: 200}
			}
		}()

		var comptes []int
		for f := range sortie {
			comptes = append(comptes, len(f.([]Requete)))
		}
		// Instants d'arrivée : 0, 100, 300, 1200, 1250, 2250 ms.
		if got := fmt.Sprint(comptes); got != "[3 2 1]" {
			t.Errorf("requêtes par seconde = %s, attendu [3 2 1]", got)
		}
	})
}
```

Ses limites sont toutefois importantes pour un usage en production :

- **le typage s'arrête aux constructeurs.** `NewMap[T, R]` est générique, mais
  les étapes communiquent par des `chan any`, et la bibliothèque convertit
  chaque élément avec `element.(T)` sans vérification. Deux étapes mal
  assorties compilent, puis **paniquent** à l'exécution ;
- **pas de gestion des erreurs** : une transformation a la signature
  `func(T) R`. Une étape qui peut échouer, comme un appel HTTP, n'a pas de
  moyen propre de le signaler ;
- **pas de `context.Context`** dans les opérateurs, donc pas d'annulation ni
  d'échéance intégrées ;
- **une maintenance reposant sur une seule personne**, dont le rythme a
  ralenti (dernière version publiée en mai 2025).

go-streams convient donc bien à un outil interne, un prototype ou un
traitement de journaux. Pour un service critique, les pipelines typés de ce
chapitre, avec `context` et `errgroup`, restent préférables.

### Les alternatives

| Besoin                                              | Outil                                   |
|-----------------------------------------------------|-----------------------------------------|
| Traitement en mémoire d'une séquence                | itérateurs (`iter.Seq`, Go 1.23)        |
| Pipelines concurrents dans un service               | channels, `context`, `errgroup` (ce chapitre) |
| Opérateurs de flux prêts à l'emploi (fenêtres…)     | go-streams                              |
| Services pilotés par des messages (Kafka, NATS, [RabbitMQ](https://www.rabbitmq.com), SQL) | [Watermill](https://watermill.io) : routeur, reprises, middlewares |
| Traitement de flux décrit par configuration         | [Redpanda Connect](https://docs.redpanda.com/redpanda-connect/) (anciennement Benthos) |
| Traitement de flux Kafka avec état                  | [goka](https://github.com/lovoo/goka)   |

Watermill mérite une mention particulière pour les architectures
événementielles : il apporte aux messages ce que `net/http` apporte aux
requêtes (un routeur, des handlers, des middlewares de reprise, de
journalisation ou de dédoublonnage), indépendamment du système de messages
utilisé.
