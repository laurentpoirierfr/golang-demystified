---
title: Golang Demystified
bookToC: false
---

# Golang Demystified

*Le langage Go, ses outils et son évolution. Édition Go 1.27.*

## Préface

Ce livre présente Go dans son ensemble : le langage, ses outils, sa bibliothèque standard, et la façon dont il a évolué au fil des versions.
Chaque exemple est compilé et testé à chaque modification du dépôt ; les premières parties se lisent dans l'ordre,
les approfondissements, les exercices et les annexes peuvent se lire séparément.

## Table des matières

<!-- sommaire -->

### Partie I — Découvrir Go

- **I.1** [Introduction](docs/decouvrir/introduction/) : pourquoi Go, sa philosophie de la simplicité, la promesse de compatibilité et le rythme des versions.
- **I.2** [Les grands projets écrits en Go](docs/decouvrir/projets-go/) : ce que l'infrastructure moderne doit à Go, des conteneurs aux bases de données.
- **I.3** [Rust et Go](docs/decouvrir/rust-et-go/) : ramasse-miettes ou propriété, la même mesure dans les deux langages, et comment choisir.
- **I.4** [Installation](docs/decouvrir/installation/) : installer Go, gérer plusieurs versions et configurer son éditeur.
- **I.5** [Premier programme](docs/decouvrir/premier-programme/) : un module, un programme, un test et une dépendance, de bout en bout.
- **I.6** [L'outil go](docs/decouvrir/outil-go/) : les commandes du quotidien, les dépendances, l'analyse et la modernisation du code.

### Partie II — Le langage

- **II.1** [Variables, constantes et types](docs/langage/bases/) : déclarations, valeur zéro, types de base, conversions, chaînes et constantes.
- **II.2** [Les opérateurs](docs/langage/operateurs/) : arithmétique, comparaison, logique, bits, affectation, et ce que Go a choisi de ne pas avoir.
- **II.3** [Contrôle de flux](docs/langage/controle/) : if, la seule boucle for, switch et defer.
- **II.4** [Tableaux, slices et maps](docs/langage/collections/) : tableaux de taille fixe, slices qui partagent leur mémoire, et maps.
- **II.5** [Fonctions](docs/langage/fonctions/) : retours multiples, fonctions variadiques, closures et itérateurs.
- **II.6** [Structures et méthodes](docs/langage/structures/) : structures, constructeurs, méthodes, composition par intégration et balises de champs.
- **II.7** [Interfaces](docs/langage/interfaces/) : des contrats implicites et petits, le type concret retrouvé, et le piège de l'interface nil.
- **II.8** [Gestion des erreurs](docs/langage/erreurs/) : des erreurs qui sont des valeurs : les créer, les envelopper, les inspecter, et panic.
- **II.9** [Génériques](docs/langage/generiques/) : fonctions et types génériques, contraintes, et quand s'en passer.
- **II.10** [Pointeurs et mémoire](docs/langage/pointeurs/) : pointeurs, pile et tas, analyse d'échappement et ramasse-miettes.
- **II.11** [Goroutines et channels](docs/langage/concurrence/) : goroutines, channels, select, mutex et détecteur d'accès concurrents.
- **II.12** [Packages et modules](docs/langage/packages/) : visibilité, documentation, modules et organisation d'un projet.

### Partie III — La bibliothèque standard

- **III.1** [Texte](docs/bibliotheque/texte/) : formater avec fmt, manipuler avec strings, convertir avec strconv, et l'UTF-8.
- **III.2** [Collections](docs/bibliotheque/collections/) : trier, chercher et modifier avec slices et maps, itérateurs et cmp.
- **III.3** [Fichiers et flux](docs/bibliotheque/fichiers/) : io.Reader et io.Writer, fichiers, lecture ligne par ligne, io/fs, embed et os.Root.
- **III.4** [Le temps](docs/bibliotheque/temps/) : instants, durées, formats, fuseaux horaires, timers et tickers.
- **III.5** [JSON](docs/bibliotheque/json/) : encoder, décoder, personnaliser, traiter des flux, et encoding/json/v2.
- **III.6** [HTTP](docs/bibliotheque/http/) : serveur, routage, middlewares, client et tests avec net/http seul.
- **III.7** [Journalisation](docs/bibliotheque/slog/) : des journaux structurés, contextualisés, sans données sensibles, au niveau réglable à chaud.
- **III.8** [Annulation et délais](docs/bibliotheque/context/) : délais, annulation et valeurs transmis d'un appel à l'autre.
- **III.9** [Tests](docs/bibliotheque/testing/) : tests de table, exemples, benchmarks, fuzzing, synctest et Testcontainers.
- **III.10** [Tour d'horizon](docs/bibliotheque/divers/) : uuid, hasard, cryptographie, CSV, regexp, flag, sync.Once et database/sql.

### Partie IV — L'évolution de Go

- **IV.1** [De Go 1.0 à Go 1.10](docs/evolution/go1-0-a-1-10/) : la promesse de compatibilité, le compilateur réécrit en Go et le ramasse-miettes concurrent.
- **IV.2** [Go 1.11](docs/evolution/go1-11/) : les modules et WebAssembly.
- **IV.3** [Go 1.12](docs/evolution/go1-12/) : TLS 1.3 en option, l'affichage trié des maps et les outils.
- **IV.4** [Go 1.13](docs/evolution/go1-13/) : l'enveloppement des erreurs, les nouveaux littéraux numériques et le proxy des modules.
- **IV.5** [Go 1.14](docs/evolution/go1-14/) : la préemption asynchrone des goroutines et un defer presque gratuit.
- **IV.6** [Go 1.15](docs/evolution/go1-15/) : un éditeur de liens réécrit, des exécutables plus petits et time/tzdata.
- **IV.7** [Go 1.16](docs/evolution/go1-16/) : embed, io/fs, les modules par défaut et Apple Silicon.
- **IV.8** [Go 1.17](docs/evolution/go1-17/) : les appels par registres et le graphe de modules élagué.
- **IV.9** [Go 1.18](docs/evolution/go1-18/) : les génériques, le fuzzing et les workspaces.
- **IV.10** [Go 1.19](docs/evolution/go1-19/) : GOMEMLIMIT, le modèle mémoire et les types atomiques.
- **IV.11** [Go 1.20](docs/evolution/go1-20/) : les erreurs multiples, l'optimisation guidée par profil et la couverture des programmes.
- **IV.12** [Go 1.21](docs/evolution/go1-21/) : min, max et clear, les toolchains automatiques, slices, maps et slog.
- **IV.13** [Go 1.22](docs/evolution/go1-22/) : une variable par itération, range sur un entier et le routage HTTP.
- **IV.14** [Go 1.23](docs/evolution/go1-23/) : range sur des fonctions, les itérateurs et les nouveaux timers.
- **IV.15** [Go 1.24](docs/evolution/go1-24/) : les alias génériques, les maps en Swiss Tables, la directive tool et os.Root.
- **IV.16** [Go 1.25](docs/evolution/go1-25/) : GOMAXPROCS adapté aux conteneurs, WaitGroup.Go, synctest et le ramasse-miettes Green Tea.
- **IV.17** [Go 1.26](docs/evolution/go1-26/) : new(expr), le ramasse-miettes Green Tea par défaut et la renaissance de go fix.
- **IV.18** [Go 1.27](docs/evolution/go1-27/) : les méthodes génériques, encoding/json/v2 et de nouveaux packages.

### Partie V — Approfondissements

- **V.1** [Go idiomatique](docs/approfondissements/go-idiomatique/) : options fonctionnelles, middlewares, injection de dépendances, anti-motifs et grille de relecture.
- **V.2** [Concurrence en pratique](docs/approfondissements/concurrence/) : pipeline, fan-out et fan-in, pool de workers, errgroup, sémaphore et singleflight.
- **V.3** [Templates](docs/approfondissements/templates/) : text/template et html/template, échappement contextuel, mise en page commune et génération de code.
- **V.4** [Une API web complète](docs/approfondissements/api-web/) : du contrat OpenAPI aux tests, une API réelle avec la bibliothèque standard.
- **V.5** [Ligne de commande](docs/approfondissements/cli-cobra-viper/) : un outil en ligne de commande avec cobra, viper et un client généré.
- **V.6** [Accès aux données](docs/approfondissements/sqlc-migrate/) : du SQL typé avec sqlc, des migrations embarquées, et Testcontainers.
- **V.7** [Architecture hexagonale](docs/approfondissements/architecture-hexagonale/) : domaine, ports et adaptateurs, chaque cercle testé séparément.
- **V.8** [Sécurité](docs/approfondissements/securite/) : OpenID Connect et JWT, TLS, secrets, mots de passe et validation des entrées.
- **V.9** [Observabilité](docs/approfondissements/observabilite/) : traces, métriques et journaux reliés par OpenTelemetry.
- **V.10** [Profilage et performance](docs/approfondissements/performance/) : mesurer, profiler avec pprof, et corriger étape par étape.
- **V.11** [Les coulisses de Go](docs/approfondissements/coulisses/) : réflexion, unsafe, cgo, ordonnanceur et ramasse-miettes.
- **V.12** [gRPC et Connect](docs/approfondissements/grpc-connect/) : un contrat protobuf, buf, intercepteurs, et REST ou gRPC.
- **V.13** [Temps réel](docs/approfondissements/temps-reel/) : un hub, Server-Sent Events, WebSocket, et le passage à plusieurs instances.
- **V.14** [Architecture événementielle](docs/approfondissements/architecture-evenementielle/) : une saga de commande avec Watermill et NATS, sans double écriture ni doublon.
- **V.15** [CI/CD](docs/approfondissements/ci-cd/) : vérifications, Dependabot, GoReleaser, images de conteneurs et SBOM.
- **V.16** [Kubernetes](docs/approfondissements/kubernetes/) : une ressource personnalisée et sa boucle de réconciliation.
- **V.17** [WebAssembly et TinyGo](docs/approfondissements/webassembly-tinygo/) : Go dans le navigateur, en greffon côté serveur et sur microcontrôleur.
- **V.18** [Go et l'IA](docs/approfondissements/go-ia/) : modèles de langage, outils, un serveur MCP, et la sécurité face à un modèle.

### Partie VI — Exercices corrigés

### Annexes

- **A** [Aide-mémoire](docs/annexes/aide-memoire/) : la syntaxe et les commandes de Go sur quelques pages.
- **B** [Glossaire](docs/annexes/glossaire/) : les termes anglais de l'écosystème Go et leur traduction dans ce livre.

<!-- /sommaire -->

Le sommaire est toujours visible dans la marge de gauche : un clic sur une
partie l'ouvre, et les liens en bas de page mènent au chapitre précédent ou
suivant.
