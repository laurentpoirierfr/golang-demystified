---
title: "Goroutines et channels"
linkTitle: "11. Goroutines et channels"
weight: 110
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/concurrence`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/concurrence).

La concurrence est au cœur de Go. Ce chapitre en présente les briques de
base. Le chapitre « Concurrence en pratique » de la partie Approfondissements
les met en œuvre dans des motifs complets : pipelines, pools de workers,
annulation, limitation de débit…

> « Don't communicate by sharing memory; share memory by communicating. »
> — Go Proverbs

## Les goroutines

Une **goroutine** est une fonction qui s'exécute de manière concurrente. Il
suffit de préfixer un appel par le mot-clé `go` :

```go
go traiter(requete)
go func() {
	fmt.Println("en arrière-plan")
}()
```

Une goroutine est très légère : elle démarre avec une pile de quelques
kilooctets, qui grandit à la demande. Un programme peut en exécuter des
centaines de milliers. Le runtime de Go les répartit sur un petit nombre de
threads système (par défaut, un par cœur, selon `GOMAXPROCS`).

Le programme s'arrête dès que `main` se termine, **sans attendre** les
goroutines en cours. Il faut donc les synchroniser.

## Attendre des goroutines : sync.WaitGroup

```go
var wg sync.WaitGroup
resultats := make([]int, 5)
for i := range 5 {
	wg.Go(func() {
		resultats[i] = i * i
	})
}
wg.Wait()
fmt.Println(resultats)
// [0 1 4 9 16]
```

Chaque goroutine écrit dans sa propre case de la slice : il n'y a pas d'accès
concurrent à une même donnée.

> [!NOTE]
> **Go 1.25** — `wg.Go(f)` lance `f` dans une goroutine et la compte
> automatiquement. Avant, on écrivait :
>
> ```go
> wg.Add(1)
> go func() {
> 	defer wg.Done()
> 	…
> }()
> ```

## Les channels

Un **channel** est un tuyau typé qui permet à des goroutines de communiquer et
de se synchroniser :

```go
ch := make(chan string)
go func() {
	ch <- "ping" // envoi
}()
msg := <-ch // réception
fmt.Println(msg)
// ping
```

Sur un channel **non bufferisé**, l'envoi bloque jusqu'à ce qu'une autre
goroutine reçoive la valeur, et inversement. L'échange est donc aussi un point
de synchronisation.

Un channel **bufferisé** accepte un nombre fixe de valeurs sans destinataire :
l'envoi ne bloque que lorsque le buffer est plein.

```go
ch := make(chan int, 2)
ch <- 1
ch <- 2 // ne bloque pas : le buffer a deux places
fmt.Println(<-ch, <-ch)
// 1 2
```

### Fermer un channel

L'émetteur ferme un channel avec `close` pour signaler qu'il n'enverra plus
rien. `range` reçoit les valeurs jusqu'à la fermeture :

```go
func producteur(n int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := range n {
			ch <- i
		}
	}()
	return ch
}

somme := 0
for v := range producteur(5) {
	somme += v
}
// somme vaut 10
```

Les types `chan<- T` (envoi seul) et `<-chan T` (réception seule) restreignent
l'usage d'un channel. Ici, l'appelant de `producteur` ne peut que recevoir.

| Opération     | channel nil      | channel ouvert                   | channel fermé         |
|---------------|------------------|----------------------------------|-----------------------|
| `ch <- v`     | bloque toujours  | bloque si plein (ou sans buffer) | **panic**             |
| `<-ch`        | bloque toujours  | bloque si vide                   | valeur zéro, immédiat |
| `close(ch)`   | **panic**        | ferme                            | **panic**             |

La réception à deux valeurs, `v, ok := <-ch`, indique avec `ok == false` que le
channel est fermé et vide.

## select

`select` attend sur plusieurs opérations de channel à la fois, et exécute la
première qui est prête. Combiné à `time.After`, il permet d'imposer un délai :

```go
resultat := make(chan string, 1)
go func() {
	time.Sleep(time.Second) // un calcul trop long
	resultat <- "terminé"
}()

select {
case r := <-resultat:
	fmt.Println(r)
case <-time.After(10 * time.Millisecond):
	fmt.Println("délai dépassé")
}
// délai dépassé
```

Un `case default` rend le `select` non bloquant : il s'exécute si aucune
opération n'est prête.

Dans un vrai programme, l'annulation et les délais passent par le package
`context`, présenté dans la partie consacrée à la bibliothèque standard.

## Protéger une donnée partagée : sync.Mutex

Lorsque plusieurs goroutines modifient la même donnée, il faut en protéger
l'accès. Un `sync.Mutex` garantit qu'une seule goroutine à la fois exécute la
section critique :

```go
type Compteur struct {
	mu sync.Mutex
	n  map[string]int
}

func (c *Compteur) Incremente(cle string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[cle]++
}

c := Compteur{n: make(map[string]int)}
var wg sync.WaitGroup
for range 1000 {
	wg.Go(func() { c.Incremente("visites") })
}
wg.Wait()
fmt.Println(c.n["visites"])
// 1000
```

`sync.RWMutex` autorise plusieurs lecteurs simultanés, et le package
`sync/atomic` fournit des compteurs sans verrou (`atomic.Int64`…).

## Détecter les accès concurrents

Une **course de données** (*data race*) se produit lorsque deux goroutines
accèdent à la même variable sans synchronisation, et qu'au moins un accès est
une écriture. Le résultat est imprévisible. Go fournit un détecteur intégré,
à activer pendant les tests :

```bash
go test -race ./...
```

Le détecteur nécessite cgo, et donc un compilateur C (`gcc` ou `clang`). Il
ralentit l'exécution, mais il signale chaque course rencontrée avec la pile
d'appels des deux accès. Utilisez-le systématiquement dans l'intégration
continue.

> [!WARNING]
> Une goroutine bloquée pour toujours, par exemple sur un channel que personne
> ne lit, n'est jamais libérée : c'est une **fuite de goroutine**. Assurez-vous
> que chaque goroutine a un moyen de se terminer.
>
> **Go 1.27** — le profil `goroutineleak` du package `runtime/pprof` détecte
> ces fuites.

## Exercices

Les solutions sont dans
[`examples/exercices/langage/concurrence`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/concurrence),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

Lancez les tests avec `go test -race` : le détecteur de courses fait partie
de la spécification.

### Une somme en parallèle ★

```go
func SommeParallele(nombres []int, parts int) int
```

Découpez la slice en `parts` morceaux, additionnez chacun dans une goroutine,
puis réunissez les sommes. Le résultat doit être juste quel que soit `parts`,
même plus grand que la slice.

### Un pipeline ★★

```go
func Genere(n int) <-chan int          // 1, 2… n, puis ferme
func Carres(entree <-chan int) <-chan int // le carré de chaque valeur
```

`for v := range Carres(Genere(5))` reçoit 1, 4, 9, 16, 25, puis s'arrête :
chaque étape ferme sa sortie quand son entrée est épuisée.

### Un groupe de travailleurs ★★★

```go
func Traite[T, R any](taches []T, n int, f func(T) R) []R
```

Appliquez `f` à chaque tâche avec **au plus n** appels simultanés, et
renvoyez les résultats **dans l'ordre des tâches**. Le test mesure le nombre
d'appels simultanés.
