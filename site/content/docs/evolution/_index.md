---
title: "IV. L'évolution de Go"
bookCollapseSection: true
weight: 40
---

# Partie IV — L'évolution de Go

Go a beaucoup évolué depuis sa version 1.0, sans jamais casser les programmes
existants. Cette partie retrace, version par version, ce qui a été ajouté au
langage, à la bibliothèque standard, aux outils et au runtime.

Chaque page suit le même plan : le contexte de la version, ses apports
principaux, puis ce qu'il faut en retenir aujourd'hui. Les nouveautés les plus
importantes sont détaillées dans les chapitres des parties précédentes.

<!-- sommaire -->

- **IV.1** [De Go 1.0 à Go 1.10](go1-0-a-1-10/) : la promesse de compatibilité, le compilateur réécrit en Go et le ramasse-miettes concurrent.
- **IV.2** [Go 1.11](go1-11/) : les modules et WebAssembly.
- **IV.3** [Go 1.12](go1-12/) : TLS 1.3 en option, l'affichage trié des maps et les outils.
- **IV.4** [Go 1.13](go1-13/) : l'enveloppement des erreurs, les nouveaux littéraux numériques et le proxy des modules.
- **IV.5** [Go 1.14](go1-14/) : la préemption asynchrone des goroutines et un defer presque gratuit.
- **IV.6** [Go 1.15](go1-15/) : un éditeur de liens réécrit, des exécutables plus petits et time/tzdata.
- **IV.7** [Go 1.16](go1-16/) : embed, io/fs, les modules par défaut et Apple Silicon.
- **IV.8** [Go 1.17](go1-17/) : les appels par registres et le graphe de modules élagué.
- **IV.9** [Go 1.18](go1-18/) : les génériques, le fuzzing et les workspaces.
- **IV.10** [Go 1.19](go1-19/) : GOMEMLIMIT, le modèle mémoire et les types atomiques.
- **IV.11** [Go 1.20](go1-20/) : les erreurs multiples, l'optimisation guidée par profil et la couverture des programmes.
- **IV.12** [Go 1.21](go1-21/) : min, max et clear, les toolchains automatiques, slices, maps et slog.
- **IV.13** [Go 1.22](go1-22/) : une variable par itération, range sur un entier et le routage HTTP.
- **IV.14** [Go 1.23](go1-23/) : range sur des fonctions, les itérateurs et les nouveaux timers.
- **IV.15** [Go 1.24](go1-24/) : les alias génériques, les maps en Swiss Tables, la directive tool et os.Root.
- **IV.16** [Go 1.25](go1-25/) : GOMAXPROCS adapté aux conteneurs, WaitGroup.Go, synctest et le ramasse-miettes Green Tea.
- **IV.17** [Go 1.26](go1-26/) : new(expr), le ramasse-miettes Green Tea par défaut et la renaissance de go fix.
- **IV.18** [Go 1.27](go1-27/) : les méthodes génériques, encoding/json/v2 et de nouveaux packages.

<!-- /sommaire -->

## La frise des versions

| Version | Date          | Ce qu'on en retient                                          |
|---------|---------------|--------------------------------------------------------------|
| 1.0     | mars 2012     | Première version stable, promesse de compatibilité           |
| 1.1     | mai 2013      | Détecteur de courses, `int` sur 64 bits                      |
| 1.5     | août 2015     | Compilateur écrit en Go, GC concurrent                       |
| 1.7     | août 2016     | `context` dans la bibliothèque standard, sous-tests          |
| 1.9     | août 2017     | Alias de types, `sync.Map`                                   |
| 1.10    | février 2018  | Cache de compilation et de tests                             |
| 1.11    | août 2018     | **Modules** (expérimentaux), WebAssembly                     |
| 1.12    | février 2019  | `go vet` réécrit, TLS 1.3 (optionnel)                        |
| 1.13    | septembre 2019 | **Enveloppement des erreurs**, proxy de modules par défaut  |
| 1.14    | février 2020  | Préemption asynchrone, `defer` quasi gratuit                 |
| 1.15    | août 2020     | Éditeur de liens plus rapide, `time/tzdata`                  |
| 1.16    | février 2021  | **`embed`**, `io/fs`, modules par défaut, Apple Silicon      |
| 1.17    | août 2021     | Appels par registres, graphe de modules élagué               |
| 1.18    | mars 2022     | **Génériques**, fuzzing, workspaces                          |
| 1.19    | août 2022     | `GOMEMLIMIT`, types atomiques, modèle mémoire révisé         |
| 1.20    | février 2023  | `errors.Join`, PGO (aperçu), couverture des programmes       |
| 1.21    | août 2023     | **`slog`, `slices`, `maps`**, `min`/`max`/`clear`, toolchains |
| 1.22    | février 2024  | **Variable de boucle par itération**, `range` sur entier, routage HTTP |
| 1.23    | août 2024     | **Itérateurs**, `unique`, nouveaux timers                    |
| 1.24    | février 2025  | Alias génériques, directive `tool`, maps [Swiss Tables](https://abseil.io/about/design/swisstables), `os.Root` |
| 1.25    | août 2025     | `GOMAXPROCS` adapté aux conteneurs, `synctest`, `WaitGroup.Go` |
| 1.26    | février 2026  | `new(expr)`, GC Green Tea, `go fix` modernisé, `errors.AsType` |
| 1.27    | août 2026     | **Méthodes génériques**, `encoding/json/v2`, `uuid`          |

## Quelques tendances

- **Le langage évolue lentement et prudemment.** En quatorze ans, les
  changements de langage tiennent en quelques lignes : alias de types,
  génériques, `range` sur entiers et sur fonctions, méthodes génériques…
  Chacun a fait l'objet d'années de discussions publiques.
- **La bibliothèque standard rattrape l'écosystème.** Des fonctions longtemps
  réécrites dans chaque projet sont devenues standard : `slices`, `maps`,
  `slog`, le routage HTTP, `uuid`.
- **L'outillage prend en charge la migration.** Les toolchains automatiques
  (1.21), la directive `go` de `go.mod` et `go fix` (1.26) permettent d'adopter
  les nouveautés sans effort.
- **Le runtime s'améliore à chaque version, gratuitement.** Pauses du GC
  divisées par mille entre Go 1.4 et Go 1.8, appels par registres, Swiss
  Tables, Green Tea : il suffit de recompiler pour en profiter.

## Rester à jour

Seules les **deux dernières** versions majeures reçoivent des correctifs de
sécurité. Mettre à jour Go tous les six mois est donc une bonne pratique, et
c'est en général sans risque :

```bash
go get go@1.27.4   # met à jour la directive go de go.mod
go fix ./...       # adopte les nouvelles fonctionnalités
go test ./...
```

Les notes de version complètes de chaque version sont publiées sur
[go.dev/doc/devel/release](https://go.dev/doc/devel/release).
