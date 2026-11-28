---
title: "Glossaire français-anglais"
linkTitle: "2. Glossaire"
weight: 20
---

Ce livre traduit la plupart des termes techniques, mais en garde certains en
anglais : ceux que toute la communauté Go emploie tels quels, en français
comme ailleurs (*slice*, *channel*, *handler*, *middleware*…). Ce glossaire
donne, pour chaque terme du livre, son équivalent anglais, celui de la
documentation officielle et des messages d'erreur, une définition courte, et
le chapitre qui l'explique. La seconde table va dans l'autre sens, de
l'anglais au français.

## A

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| annulation | cancellation | arrêt demandé à un traitement en cours, transmis par un contexte | [Annulation et délais](../../bibliotheque/context/) |
| arrêt propre | graceful shutdown | arrêt d'un serveur qui termine les requêtes en cours avant de s'arrêter | [HTTP](../../bibliotheque/http/) |
| assertion de type | type assertion | `v.(T)` : extrait la valeur concrète contenue dans une interface | [Interfaces](../../langage/interfaces/) |

## B

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| bac à sable | sandbox | environnement isolé, sans accès au système hôte, sauf permission explicite | [WebAssembly et TinyGo](../../approfondissements/webassembly-tinygo/) |
| benchmark | benchmark | mesure de performance écrite comme un test : `BenchmarkX`, `b.Loop()` | [Tests](../../bibliotheque/testing/) |

## C

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| chaîne de compilation | toolchain | le compilateur et les outils d'une version de Go ; `go` télécharge celle que demande `go.mod` | [L'outil go](../../decouvrir/outil-go/) |
| channel | channel | conduit typé entre goroutines : envoi, réception, fermeture | [Goroutines et channels](../../langage/concurrence/) |
| constante non typée | untyped constant | constante sans type fixe, qui prend celui que demande son usage | [Variables, constantes et types](../../langage/bases/) |
| contexte | context | valeur `context.Context` qui porte l'annulation, l'échéance et des valeurs d'une requête | [Annulation et délais](../../bibliotheque/context/) |
| contrainte | constraint | interface qui limite les types acceptés par un paramètre de type | [Génériques](../../langage/generiques/) |
| contrôleur | controller | dans Kubernetes, boucle qui rapproche l'état réel de l'état voulu | [Kubernetes](../../approfondissements/kubernetes/) |
| course de données | data race | accès simultanés à une même variable, dont au moins une écriture, sans synchronisation | [Goroutines et channels](../../langage/concurrence/) |
| couverture | coverage | part du code exécutée par les tests : `go test -cover` | [Tests](../../bibliotheque/testing/) |
| cycle de vie | lifecycle | les étapes de démarrage et d'arrêt d'une application ou d'un composant | [Go idiomatique](../../approfondissements/go-idiomatique/) |

## D

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| délai | timeout | durée maximale accordée à une opération : `context.WithTimeout` | [Annulation et délais](../../bibliotheque/context/) |
| dépôt | repository | composant qui stocke et retrouve des données, derrière une interface ; aussi un dépôt git | [Architecture hexagonale](../../approfondissements/architecture-hexagonale/) |
| détecteur de courses | race detector | instrumentation activée par `-race`, qui signale les courses de données à l'exécution | [Goroutines et channels](../../langage/concurrence/) |
| doublure | test double | implémentation simplifiée d'une dépendance, utilisée dans les tests | [Go idiomatique](../../approfondissements/go-idiomatique/) |

## E

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| échappement | escape analysis | analyse par laquelle le compilateur décide si une variable reste sur la pile ou va sur le tas | [Performance](../../approfondissements/performance/) |
| échéance | deadline | instant au-delà duquel une opération est abandonnée : `context.WithDeadline` | [Annulation et délais](../../bibliotheque/context/) |
| envelopper | wrap | ajouter du contexte à une erreur en gardant sa cause : `fmt.Errorf("…: %w", err)` | [Gestion des erreurs](../../langage/erreurs/) |
| erreur sentinelle | sentinel error | variable d'erreur exportée, reconnue par `errors.Is` : `io.EOF`, `fs.ErrNotExist` | [Gestion des erreurs](../../langage/erreurs/) |
| exporté | exported | nom qui commence par une majuscule, visible hors de son package | [Packages et modules](../../langage/packages/) |

## F

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| fermeture | closure | fonction qui capture des variables de l'endroit où elle est définie | [Fonctions](../../langage/fonctions/) |
| fournisseur | provider | dans Wire ou Fx, constructeur qui fournit une dépendance | [Go idiomatique](../../approfondissements/go-idiomatique/) |
| fuite de goroutine | goroutine leak | goroutine bloquée pour toujours, qui ne libère jamais sa mémoire | [Go idiomatique](../../approfondissements/go-idiomatique/) |
| fuzzing | fuzzing | génération d'entrées pour trouver celles qui cassent une propriété : `go test -fuzz` | [Tests](../../bibliotheque/testing/) |

## G

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| génériques | generics | fonctions et types paramétrés par des types, depuis Go 1.18 | [Génériques](../../langage/generiques/) |
| goroutine | goroutine | fonction exécutée de façon concurrente, gérée par le runtime ; lancée par `go f()` | [Goroutines et channels](../../langage/concurrence/) |
| greffon | plugin | module chargé et exécuté par un programme hôte, ici en WebAssembly | [WebAssembly et TinyGo](../../approfondissements/webassembly-tinygo/) |

## H

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| handler | handler | ce qui traite une requête HTTP (`http.Handler`) ou un message de journal (`slog.Handler`) | [HTTP](../../bibliotheque/http/) |

## I

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| injection de dépendances | dependency injection | fournir à un composant ses dépendances, plutôt que de le laisser les créer | [Go idiomatique](../../approfondissements/go-idiomatique/) |
| intégration | embedding | inclure un type dans une structure ou une interface ; ses méthodes sont promues | [Structures et méthodes](../../langage/structures/) |
| interface | interface | ensemble de méthodes, satisfait implicitement par tout type qui les possède | [Interfaces](../../langage/interfaces/) |
| itérateur | iterator | fonction `iter.Seq` ou `iter.Seq2`, parcourue par `range` | [Collections](../../bibliotheque/collections/) |

## J

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| journal | log | suite de messages horodatés produits par un programme | [Journalisation](../../bibliotheque/slog/) |
| journalisation | logging | production de journaux, structurés avec `log/slog` | [Journalisation](../../bibliotheque/slog/) |

## M

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| map | map | table associative entre clés comparables et valeurs | [Tableaux, slices et maps](../../langage/collections/) |
| méthode | method | fonction attachée à un type par un récepteur | [Structures et méthodes](../../langage/structures/) |
| méthode promue | promoted method | méthode d'un type intégré, appelable directement sur la structure qui l'intègre | [Structures et méthodes](../../langage/structures/) |
| métrique | metric | mesure numérique exposée par un service : compteur, jauge, histogramme | [Observabilité](../../approfondissements/observabilite/) |
| middleware | middleware | fonction qui enveloppe un handler pour lui ajouter un comportement | [HTTP](../../bibliotheque/http/) |
| module | module | ensemble de packages versionnés ensemble, décrit par `go.mod` | [Packages et modules](../../langage/packages/) |
| motif | pattern | solution éprouvée à un problème courant : options fonctionnelles, middleware… | [Go idiomatique](../../approfondissements/go-idiomatique/) |
| mutex | mutex | verrou d'exclusion mutuelle : `sync.Mutex` | [Goroutines et channels](../../langage/concurrence/) |

## O

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| options fonctionnelles | functional options | réglages facultatifs passés à un constructeur sous forme de fonctions | [Go idiomatique](../../approfondissements/go-idiomatique/) |
| ordonnanceur | scheduler | partie du runtime qui répartit les goroutines sur les fils du système | [Les coulisses de Go](../../approfondissements/coulisses/) |

## P

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| package | package | unité de code : un répertoire, un espace de noms | [Packages et modules](../../langage/packages/) |
| panique | panic | arrêt brutal d'une goroutine, rattrapable par `recover` | [Gestion des erreurs](../../langage/erreurs/) |
| paramètre de type | type parameter | type passé à une fonction ou un type générique : le `T` de `Max[T]` | [Génériques](../../langage/generiques/) |
| pile | stack | mémoire propre à une goroutine, pour ses appels de fonctions ; elle grandit à la demande | [Pointeurs et mémoire](../../langage/pointeurs/) |
| pointeur | pointer | adresse d'une variable : `&x` la donne, `*p` la lit | [Pointeurs et mémoire](../../langage/pointeurs/) |
| pool de workers | worker pool | nombre fixe de goroutines qui se partagent une file de tâches | [Concurrence en pratique](../../approfondissements/concurrence/) |
| profil | profile | relevé de l'activité d'un programme (processeur, mémoire, goroutines) : `pprof` | [Performance](../../approfondissements/performance/) |

## R

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| ramasse-miettes | garbage collector | partie du runtime qui libère la mémoire que plus rien ne référence | [Les coulisses de Go](../../approfondissements/coulisses/) |
| récepteur | receiver | paramètre spécial d'une méthode, écrit avant son nom : `func (p *Pile) Empile(…)` | [Structures et méthodes](../../langage/structures/) |
| réconciliation | reconciliation | dans Kubernetes, travail d'un contrôleur qui ramène l'état réel vers l'état voulu | [Kubernetes](../../approfondissements/kubernetes/) |
| réflexion | reflection | inspection des types et des valeurs à l'exécution : `reflect` | [Les coulisses de Go](../../approfondissements/coulisses/) |
| runtime | runtime | code embarqué dans chaque binaire : ordonnanceur, ramasse-miettes, gestion de la mémoire | [Les coulisses de Go](../../approfondissements/coulisses/) |

## S

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| slice | slice | vue sur un tableau : pointeur, longueur et capacité | [Tableaux, slices et maps](../../langage/collections/) |
| sonde | probe | vérification périodique de l'état d'un conteneur par Kubernetes : *liveness*, *readiness* | [Kubernetes](../../approfondissements/kubernetes/) |
| span | span | une opération mesurée dans une trace, avec son début, sa durée et ses attributs | [Observabilité](../../approfondissements/observabilite/) |

## T

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| tableau | array | suite de taille fixe, qui fait partie du type : `[4]int` | [Tableaux, slices et maps](../../langage/collections/) |
| tag | struct tag | métadonnée d'un champ de structure, lue par réflexion : `` `json:"nom"` `` | [JSON](../../bibliotheque/json/) |
| tas | heap | mémoire partagée, gérée par le ramasse-miettes, pour les valeurs qui survivent à leur fonction | [Pointeurs et mémoire](../../langage/pointeurs/) |
| test de table | table-driven test | test qui parcourt une liste de cas, un sous-test par cas | [Tests](../../bibliotheque/testing/) |
| trace | trace | parcours d'une requête à travers les services, fait de spans | [Observabilité](../../approfondissements/observabilite/) |
| type switch | type switch | `switch v := x.(type)` : choisit une branche selon le type concret d'une interface | [Interfaces](../../langage/interfaces/) |

## V

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| valeur zéro | zero value | valeur d'une variable non initialisée : `0`, `""`, `false`, `nil` | [Variables, constantes et types](../../langage/bases/) |
| variadique | variadic | fonction qui accepte un nombre variable d'arguments : `func f(n ...int)` | [Fonctions](../../langage/fonctions/) |
| verrou | lock | mécanisme qui réserve une ressource à une seule goroutine à la fois | [Goroutines et channels](../../langage/concurrence/) |

## W

| Terme du livre | Anglais | Définition | Voir |
|----------------|---------|------------|------|
| workspace | workspace | `go.work` : plusieurs modules développés ensemble, sans les publier | [L'outil go](../../decouvrir/outil-go/) |

## De l'anglais au français

Les termes que le livre traduit, dans l'ordre de l'anglais :

| Anglais | Terme du livre |
|---------|----------------|
| array | tableau |
| cancellation | annulation |
| closure | fermeture |
| constraint | contrainte |
| context | contexte |
| controller | contrôleur |
| coverage | couverture |
| data race | course de données |
| deadline | échéance |
| dependency injection | injection de dépendances |
| embedding | intégration |
| escape analysis | échappement |
| exported | exporté |
| functional options | options fonctionnelles |
| garbage collector | ramasse-miettes |
| generics | génériques |
| goroutine leak | fuite de goroutine |
| graceful shutdown | arrêt propre |
| heap | tas |
| iterator | itérateur |
| lifecycle | cycle de vie |
| lock | verrou |
| log | journal |
| logging | journalisation |
| method | méthode |
| metric | métrique |
| panic | panique |
| pattern | motif |
| plugin | greffon |
| pointer | pointeur |
| probe | sonde |
| profile | profil |
| promoted method | méthode promue |
| provider | fournisseur |
| race detector | détecteur de courses |
| receiver | récepteur |
| reflection | réflexion |
| repository | dépôt |
| sandbox | bac à sable |
| scheduler | ordonnanceur |
| sentinel error | erreur sentinelle |
| stack | pile |
| struct tag | tag |
| table-driven test | test de table |
| test double | doublure |
| timeout | délai |
| toolchain | chaîne de compilation |
| type assertion | assertion de type |
| type parameter | paramètre de type |
| untyped constant | constante non typée |
| variadic | variadique |
| worker pool | pool de workers |
| wrap | envelopper |
| zero value | valeur zéro |
