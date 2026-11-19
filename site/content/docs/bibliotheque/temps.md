---
title: "Le temps : time"
weight: 40
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/temps`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/temps).

Le package `time` manipule des **instants** (`time.Time`), des **durées**
(`time.Duration`) et des **fuseaux horaires** (`time.Location`).

## Les instants

```go
maintenant := time.Now()
t := time.Date(2026, time.August, 12, 14, 30, 0, 0, time.UTC)
fmt.Println(t)
// 2026-08-12 14:30:00 +0000 UTC
fmt.Println(t.Year(), t.Month(), t.Day(), t.Weekday(), t.YearDay())
// 2026 August 12 Wednesday 224
```

Un `time.Time` est une valeur : passez-le par valeur, pas par pointeur. Sa
valeur zéro (1er janvier de l'an 1) se teste avec `t.IsZero()`.

Comparez deux instants avec `Equal`, `Before`, `After` ou `Compare` (Go 1.20),
et non avec `==` : deux `time.Time` qui représentent le même instant dans deux
fuseaux différents ne sont pas égaux au sens de `==`.

## Formater et analyser

Go n'utilise pas de codes comme `%Y-%m-%d`. Un format est un **exemple** écrit
avec une date de référence précise : le **lundi 2 janvier 2006 à 15:04:05,
fuseau -0700**. Chaque élément correspond à un nombre facile à retenir :
`01/02 03:04:05PM '06 -0700`, soit 1, 2, 3, 4, 5, 6, 7.

| Élément | Signification        | Élément | Signification             |
|---------|----------------------|---------|---------------------------|
| `2006`  | année                | `15`    | heure (24 h)              |
| `01`    | mois (numéro)        | `03`    | heure (12 h)              |
| `Jan`, `January` | mois (nom) | `04`   | minutes                   |
| `02`    | jour                 | `05`    | secondes                  |
| `Mon`, `Monday` | jour (nom)   | `PM`    | AM/PM                     |
| `-0700`, `MST` | fuseau        | `.000`  | millisecondes             |

```go
t := time.Date(2026, time.August, 12, 14, 30, 5, 0, time.UTC)
t.Format(time.RFC3339)              // 2026-08-12T14:30:05Z
t.Format(time.DateTime)             // 2026-08-12 14:30:05
t.Format(time.DateOnly)             // 2026-08-12
t.Format("02/01/2006 à 15h04")      // 12/08/2026 à 14h30
t.Format("Mon 2 Jan 06, 3:04 PM")   // Wed 12 Aug 26, 2:30 PM
```

Les constantes `time.DateTime`, `time.DateOnly` et `time.TimeOnly` existent
depuis **Go 1.20**. Pour les échanges entre programmes (JSON, API), utilisez
**RFC 3339**, le format par défaut de `encoding/json`.

`time.Parse` analyse une chaîne selon le même principe :

```go
t, err := time.Parse(time.DateOnly, "2012-03-28")
// 2012-03-28 00:00:00 +0000 UTC

_, err = time.Parse(time.DateOnly, "28/03/2012")
// parsing time "28/03/2012" as "2006-01-02": cannot parse "28/03/2012" as "2006"
```

Sans fuseau dans la chaîne, `Parse` suppose UTC. `time.ParseInLocation`
précise le fuseau à utiliser.

## Les durées

`time.Duration` est un nombre entier de **nanosecondes**. Les constantes
`time.Nanosecond` à `time.Hour` permettent d'écrire des durées lisibles :

```go
d := 90 * time.Minute
fmt.Println(d, d.Hours(), d.Seconds())
// 1h30m0s 1.5 5400

attente, _ := time.ParseDuration("1h15m30.5s")
fmt.Println(attente, attente.Round(time.Minute))
// 1h15m30.5s 1h16m0s
```

Multiplier une durée par une variable entière demande une conversion :

```go
n := 3
time.Duration(n) * time.Second // 3s
```

> [!WARNING]
> `time.Sleep(5)` attend 5 **nanosecondes**, pas 5 secondes. Écrivez toujours
> l'unité : `time.Sleep(5 * time.Second)`.

## Calculer avec les dates

```go
debut := time.Date(2026, time.January, 31, 9, 0, 0, 0, time.UTC)
debut.Add(36 * time.Hour)       // 2026-02-01 21:00:00
debut.AddDate(0, 1, 0)          // 2026-03-03 : le 31 février est normalisé !

fin := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
fin.Sub(debut)                  // 696h0m0s
debut.Truncate(24 * time.Hour)  // 2026-01-31 00:00:00
```

`Add` ajoute une durée exacte, `AddDate` ajoute des années, des mois et des
jours calendaires. Attention à la normalisation des dates inexistantes, comme
le 31 février ci-dessus.

Pour mesurer un temps écoulé, utilisez `time.Since` (et `time.Until` pour un
délai restant). `time.Now()` contient une horloge **monotone** : la mesure
reste juste même si l'horloge système est modifiée pendant ce temps.

```go
debut := time.Now()
traitement()
log.Printf("durée : %v", time.Since(debut))
```

## Les fuseaux horaires

```go
paris, err := time.LoadLocation("Europe/Paris")
tokyo, _ := time.LoadLocation("Asia/Tokyo")

reunion := time.Date(2026, time.July, 1, 10, 0, 0, 0, paris)
fmt.Println(reunion)            // 2026-07-01 10:00:00 +0200 CEST
fmt.Println(reunion.In(tokyo))  // 2026-07-01 17:00:00 +0900 JST
fmt.Println(reunion.UTC())      // 2026-07-01 08:00:00 +0000 UTC
```

`LoadLocation` lit la base de fuseaux horaires du système. Dans une image
[Docker](https://www.docker.com) minimale qui n'en contient pas, importez `time/tzdata` (ou compilez avec
`-tags timetzdata`) pour l'embarquer dans l'exécutable, au prix d'environ
450 Ko.

Une bonne pratique : stockez et échangez les instants en **UTC**, et ne
convertissez dans le fuseau de l'utilisateur qu'à l'affichage.

## Timers et tickers

```go
timer := time.NewTimer(2 * time.Second) // un événement unique
ticker := time.NewTicker(time.Second)   // un événement périodique
defer ticker.Stop()

for {
	select {
	case t := <-ticker.C:
		fmt.Println("tic", t)
	case <-timer.C:
		return
	}
}
```

`time.After(d)` renvoie directement le channel d'un timer, et
`time.AfterFunc(d, f)` exécute `f` après le délai, dans sa propre goroutine.

> [!NOTE]
> **Go 1.23** — les timers et tickers qui ne sont plus référencés sont
> libérés par le ramasse-miettes, même sans appel à `Stop`. Leurs channels
> sont devenus non bufferisés, ce qui évite de recevoir une valeur périmée
> après un `Reset` ou un `Stop`. Depuis **Go 1.27**, l'ancien comportement
> (`asynctimerchan`) n'est plus disponible.
