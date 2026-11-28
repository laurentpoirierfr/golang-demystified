---
title: "III.8 Annulation et délais"
weight: 80
---

# Chapitre III.8 — Annulation et délais : context

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/contexte`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/contexte).

## Le problème

Un serveur reçoit une requête, qui interroge une base de données, puis appelle
un autre service. Si le client abandonne, ou si la requête dépasse son délai,
il faut arrêter **tout** ce travail, dans toutes les goroutines concernées.

Le package `context` (**Go 1.7**) résout ce problème. Un `context.Context`
transporte, à travers les appels de fonctions et les goroutines :

- un signal d'**annulation** ;
- une **échéance** (*deadline*) ;
- des **valeurs** liées à la requête (identifiant de requête, utilisateur
  authentifié…).

## Les conventions

- Le contexte est le **premier paramètre** d'une fonction, nommé `ctx` :
  `func Charge(ctx context.Context, id int) (*Client, error)`.
- On ne le stocke **pas** dans une structure : il accompagne une opération, pas
  un objet.
- On ne passe jamais un contexte `nil`. En l'absence de contexte, on utilise
  `context.Background()` (racine, dans `main` ou un test) ou
  `context.TODO()` (en attendant mieux).

Les contextes forment un **arbre** : chaque fonction `With…` crée un enfant.
Annuler un contexte annule tous ses descendants, jamais ses parents.

## Délais et annulation

```go
// calcul simule un traitement long qui respecte l'annulation.
func calcul(ctx context.Context, duree time.Duration) (int, error) {
	select {
	case <-time.After(duree):
		return 42, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
defer cancel()

_, err := calcul(ctx, time.Second)
fmt.Println(err, errors.Is(err, context.DeadlineExceeded))
// context deadline exceeded true
```

| Fonction                        | Le contexte est annulé…                          |
|---------------------------------|--------------------------------------------------|
| `WithCancel(parent)`            | à l'appel de `cancel()`                          |
| `WithTimeout(parent, durée)`    | après la durée, ou à l'appel de `cancel()`       |
| `WithDeadline(parent, instant)` | à l'instant donné, ou à l'appel de `cancel()`    |
| `WithCancelCause(parent)`       | à l'appel de `cancel(err)`, avec une cause (Go 1.20) |

> [!WARNING]
> Appelez **toujours** la fonction `cancel` renvoyée, en général avec `defer`,
> même si l'opération se termine normalement. Sinon, les ressources du
> contexte ne sont libérées qu'à l'annulation de son parent. `go vet` signale
> les oublis.

Une fonction respecte l'annulation de deux façons :

- en surveillant `ctx.Done()`, un channel fermé à l'annulation, dans un
  `select` ;
- en transmettant `ctx` aux fonctions qu'elle appelle. La plupart des API qui
  font des entrées-sorties l'acceptent : `http.NewRequestWithContext`,
  `db.QueryContext`, `exec.CommandContext`, `net.Dialer.DialContext`…

`ctx.Err()` vaut `context.Canceled` ou `context.DeadlineExceeded` une fois le
contexte annulé, et `nil` avant.

### Indiquer la cause

`context.Cause` renvoie la raison précise de l'annulation, en plus de l'erreur
générique :

```go
ctx, cancel := context.WithCancelCause(context.Background())
cancel(errors.New("quota dépassé"))
fmt.Println(ctx.Err())          // context canceled
fmt.Println(context.Cause(ctx)) // quota dépassé
```

## Dans un serveur HTTP

Chaque requête porte un contexte, accessible par `r.Context()`. Il est annulé
lorsque le client ferme la connexion ou lorsque le serveur s'arrête :

```go
func (a *API) rapport(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	lignes, err := a.db.QueryContext(ctx, "SELECT …")
	if err != nil {
		http.Error(w, "service indisponible", http.StatusServiceUnavailable)
		return
	}
	…
}
```

Pour arrêter un programme proprement sur `Ctrl+C` ou `SIGTERM`, on crée un
contexte annulé par le signal :

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

## Les valeurs

`context.WithValue` attache une valeur au contexte. La clé doit être d'un
**type non exporté**, propre au package, pour éviter toute collision :

```go
type cleContexte struct{}

func avecRequeteID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, cleContexte{}, id)
}

func requeteID(ctx context.Context) string {
	id, _ := ctx.Value(cleContexte{}).(string)
	return id
}
```

Réservez les valeurs aux données **transversales** liées à la requête :
identifiant de trace, utilisateur authentifié. Les paramètres d'une fonction
doivent rester des paramètres explicites.

## Fonctions avancées

Go 1.21 a ajouté trois fonctions :

- `context.AfterFunc(ctx, f)` exécute `f` dans une goroutine à l'annulation de
  `ctx`, par exemple pour fermer une connexion ;
- `context.WithoutCancel(ctx)` crée un contexte qui garde les valeurs du
  parent, mais **pas** son annulation. Il sert à lancer un traitement qui doit
  se terminer même si la requête d'origine est annulée (envoi d'un e-mail de
  confirmation, écriture d'un journal d'audit) ;
- `context.WithDeadlineCause` et `context.WithTimeoutCause` associent une
  cause à l'expiration du délai.

```go
parent, cancel := context.WithCancel(context.Background())
detache := context.WithoutCancel(parent)
cancel()
fmt.Println(parent.Err(), detache.Err())
// context canceled <nil>
```

Dans un test, `t.Context()` (**Go 1.24**) fournit un contexte annulé
automatiquement à la fin du test.

## Exercices

Les solutions sont dans
[`examples/exercices/bibliotheque/context`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/bibliotheque/context),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Une attente annulable ★

```go
func Attends(ctx context.Context, d time.Duration) error
```

Attendez `d`, sauf si le contexte est annulé avant : renvoyez alors
`ctx.Err()`. Indice : `select` sur un timer et sur `ctx.Done()`.

### Une valeur typée ★★

```go
func AvecUtilisateur(ctx context.Context, nom string) context.Context
func Utilisateur(ctx context.Context) (string, bool)
```

Utilisez une clé d'un type non exporté : le test ajoute une valeur sous la
clé `"cleUtilisateur"` (une chaîne), qui ne doit pas écraser la vôtre.

### Le premier qui réussit ★★★

```go
func Premier(ctx context.Context, sources ...func(context.Context) (string, error)) (string, error)
```

Interrogez toutes les sources en parallèle, renvoyez la première réponse
réussie, et **annulez les autres** ; si toutes échouent, renvoyez leurs
erreurs réunies. Attention aux goroutines qui resteraient bloquées sur un
channel que plus personne ne lit.
