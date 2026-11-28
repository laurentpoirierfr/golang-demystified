---
title: "III.7 Journalisation"
description: "des journaux structurés, contextualisés, sans données sensibles, au niveau réglable à chaud."
weight: 70
---

# Chapitre III.7 — Journalisation : log/slog

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/slog`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/slog).

## log : le journal minimal

Le package `log`, présent depuis Go 1.0, écrit des messages horodatés sur la
sortie d'erreur :

```go
log.Printf("connexion de %s", utilisateur)
log.Fatal(err) // affiche puis appelle os.Exit(1)
```

Il suffit pour un petit outil, mais ses messages sont du texte libre,
difficiles à filtrer et à analyser par une plateforme de journaux.

## slog : la journalisation structurée

> [!NOTE]
> **Go 1.21** — le package `log/slog` apporte la journalisation
> **structurée** : chaque message est accompagné de paires clé-valeur.

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
logger.Info("serveur démarré", "port", 8080, "tls", false)
logger.Warn("disque presque plein", "libre", "2 Go")
logger.Debug("invisible : niveau Info par défaut")
```

```
time=2026-08-12T10:00:00.000+02:00 level=INFO msg="serveur démarré" port=8080 tls=false
time=2026-08-12T10:00:00.000+02:00 level=WARN msg="disque presque plein" libre="2 Go"
```

Les quatre niveaux standard sont `Debug`, `Info`, `Warn` et `Error`. Les
fonctions de niveau package (`slog.Info`…) utilisent le journal par défaut,
que l'on remplace au démarrage du programme :

```go
slog.SetDefault(logger)
```

`slog.SetDefault` redirige aussi le package `log` vers ce journal : les
bibliothèques qui utilisent `log.Printf` produisent alors des messages
structurés.

## Les handlers

Un **handler** décide du format et de la destination des messages. Deux sont
fournis :

| Handler                 | Format                                        | Usage                 |
|-------------------------|-----------------------------------------------|-----------------------|
| `slog.NewTextHandler`   | `clé=valeur`, lisible par un humain           | développement         |
| `slog.NewJSONHandler`   | une ligne JSON par message                    | production            |

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))
logger.Debug("requête", "méthode", "GET", "durée", 42*time.Millisecond)
logger.Error("échec", slog.String("table", "commandes"), slog.Int("tentatives", 3))
```

```json
{"time":"…","level":"DEBUG","msg":"requête","méthode":"GET","durée":42000000}
{"time":"…","level":"ERROR","msg":"échec","table":"commandes","tentatives":3}
```

Les fonctions `slog.String`, `slog.Int`, `slog.Duration`, `slog.Any`… créent
des attributs typés. Elles évitent les allocations et les erreurs de paires
clé-valeur dépareillées, que `go vet` détecte également.

`HandlerOptions` règle le niveau minimal (`Level`), l'ajout de la position
dans le code source (`AddSource`) et la transformation des attributs
(`ReplaceAttr`), par exemple pour renommer ou supprimer une clé.

> [!NOTE]
> **Go 1.26** — `slog.NewMultiHandler` envoie chaque message à plusieurs
> handlers, par exemple du texte sur la console et du JSON dans un fichier.

## Ajouter du contexte

`logger.With` crée un journal dérivé, qui ajoute des attributs à chaque
message. C'est l'idéal pour attacher l'identifiant d'une requête à tous les
messages produits pendant son traitement. `slog.Group` regroupe des
attributs sous un préfixe :

```go
reqLog := logger.With("requete_id", "a1b2", "utilisateur", 42)
reqLog.Info("commande créée", slog.Group("commande", "id", 1001, "montant", 59.9))
// level=INFO msg="commande créée" requete_id=a1b2 utilisateur=42 commande.id=1001 commande.montant=59.9
```

Les méthodes `InfoContext`, `ErrorContext`… reçoivent un `context.Context`. Un
handler personnalisé peut y lire des informations, comme l'identifiant de
trace [OpenTelemetry](https://opentelemetry.io) de la requête en cours.

## Masquer les données sensibles

Un type qui implémente `slog.LogValuer` contrôle sa propre représentation dans
les journaux. C'est le moyen le plus sûr de ne jamais journaliser un mot de
passe ou un jeton :

```go
type Secret string

func (Secret) LogValue() slog.Value { return slog.StringValue("***") }

logger.Info("connexion", "utilisateur", "alice", "mot_de_passe", Secret("hunter2"))
// level=INFO msg=connexion utilisateur=alice mot_de_passe=***
```

## Changer de niveau à chaud

`slog.LevelVar` est un niveau modifiable pendant l'exécution, par exemple
depuis un endpoint d'administration ou à la réception d'un signal :

```go
var niveau slog.LevelVar // Info par défaut
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &niveau}))

niveau.Set(slog.LevelDebug)
```

## Bonnes pratiques

- Le message décrit l'**événement** et reste constant ; les données variables
  vont dans les attributs. On peut ainsi compter les occurrences d'un message.
- Utilisez des noms de clés **cohérents** dans toute l'application
  (`utilisateur_id`, et non tantôt `user`, tantôt `uid`).
- Passez le `*slog.Logger` comme une dépendance plutôt que d'utiliser partout
  le journal global : c'est plus facile à tester.
- En production, journalisez en JSON sur la sortie standard, et laissez la
  plateforme ([Docker](https://www.docker.com), [Kubernetes](https://kubernetes.io), [systemd](https://systemd.io)) collecter les messages.

## Exercices

Les solutions sont dans
[`examples/exercices/bibliotheque/slog`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/bibliotheque/slog),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Un journal JSON ★

```go
func Nouveau(w io.Writer, niveau slog.Level, service string) *slog.Logger
```

Le journal écrit du JSON, ignore les messages sous `niveau`, et ajoute
l'attribut `service` à chaque message.

### Masquer un secret ★★

```go
type Secret string
```

Faites qu'un `Secret` s'affiche `***` dans tous les journaux, quel que soit le
handler. Indice : l'interface `slog.LogValuer`.

### Un handler qui compte ★★★

```go
func NouveauCompteur(suivant slog.Handler) *Compteur
func (c *Compteur) Nombre(l slog.Level) int
```

`Compteur` satisfait `slog.Handler` : il compte les messages par niveau, et
les transmet à `suivant`. Les journaux dérivés (`log.With(…)`,
`log.WithGroup(…)`) comptent dans les mêmes compteurs. Le test vérifie aussi
le contrat complet d'un handler avec `testing/slogtest`.
