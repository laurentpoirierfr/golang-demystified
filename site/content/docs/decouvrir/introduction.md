---
title: "Introduction"
weight: 10
---

## Pourquoi Go ?

Go est né chez Google en 2007, d'une frustration partagée par trois ingénieurs :
Robert Griesemer, Rob Pike et Ken Thompson. Les programmes sur lesquels ils
travaillaient, écrits en C++ et en Java, mettaient de longues minutes à compiler,
leurs dépendances étaient difficiles à maîtriser, et les machines multicœurs
étaient mal exploitées par des langages pensés pour un seul processeur.

Leur objectif n'était pas d'inventer des concepts nouveaux, mais de **retirer**
tout ce qui ralentit une équipe : un langage petit, lisible, qui compile en
quelques secondes, produit un binaire autonome et rend la concurrence naturelle.

Go a été rendu public en novembre 2009. La version **Go 1.0**, publiée en mars
2012, a marqué le début de la stabilité. Depuis, Go est devenu le langage de
l'infrastructure moderne : [Docker](https://www.docker.com), [Kubernetes](https://kubernetes.io), [Terraform](https://www.terraform.io), [Prometheus](https://prometheus.io), [etcd](https://etcd.io) ou
[Caddy](https://caddyserver.com) sont écrits en Go. Le chapitre « Les grands
projets écrits en Go » en présente un panorama.

## Une philosophie de la simplicité

Quelques principes guident la conception de Go et reviennent tout au long de ce
livre :

- **Peu de mots-clés, une seule façon de faire.** Go compte 25 mots-clés. Il n'y a
  qu'une boucle (`for`), pas d'héritage, pas d'exceptions, pas de surcharge
  d'opérateurs.
- **La lisibilité avant l'élégance.** Le code est plus souvent lu qu'écrit. Le
  formatage est imposé par `gofmt` : il n'y a pas de débat sur le style.
- **La composition plutôt que l'héritage.** Les types se combinent par
  intégration, et les interfaces sont satisfaites implicitement.
- **Les erreurs sont des valeurs.** Une fonction qui peut échouer renvoie une
  erreur, que l'appelant traite explicitement.
- **La concurrence intégrée au langage.** Les goroutines et les channels font
  partie du langage, et non d'une bibliothèque.
- **Un outillage complet fourni d'office.** Compilation, tests, benchmarks,
  formatage, documentation, profilage et gestion des dépendances sont tous
  disponibles via la commande `go`.

> « Clear is better than clever. » — Rob Pike, *Go Proverbs*

## La promesse de compatibilité

Avec Go 1.0, l'équipe s'est engagée sur une **promesse de compatibilité** : un
programme qui compile avec Go 1.x continuera de compiler et de fonctionner avec
toutes les versions 1.y suivantes. Cette promesse a été tenue depuis plus de
quatorze ans, et c'est l'une des raisons du succès de Go en entreprise.

Le langage évolue pourtant : génériques, itérateurs, nouvelles fonctions de la
bibliothèque standard, nouveau ramasse-miettes… Ces évolutions sont ajoutées
sans casser l'existant. Lorsqu'un comportement doit changer, il dépend de la
version déclarée dans le fichier `go.mod` du module, ce qui permet à chaque
projet de migrer à son rythme.

## Un cycle de sortie régulier

Une nouvelle version majeure de Go sort **tous les six mois**, en février et en
août. Les deux dernières versions majeures reçoivent des correctifs de sécurité
(les versions mineures, comme `1.27.4`).

| Version | Date        | Apport marquant                                      |
|---------|-------------|------------------------------------------------------|
| 1.0     | mars 2012   | Stabilité et promesse de compatibilité               |
| 1.5     | août 2015   | Compilateur écrit en Go, ramasse-miettes concurrent  |
| 1.11    | août 2018   | Modules                                              |
| 1.18    | mars 2022   | Génériques, fuzzing, workspaces                      |
| 1.21    | août 2023   | `slices`, `maps`, `log/slog`, `min`, `max`, `clear`  |
| 1.22    | février 2024| Variable de boucle par itération, `range` sur entier |
| 1.23    | août 2024   | Itérateurs (`range` sur fonction)                    |
| 1.26    | février 2026| `new(expr)`, ramasse-miettes Green Tea               |
| 1.27    | août 2026   | Méthodes génériques, `encoding/json/v2`, `uuid`      |

La partie « L'évolution de Go » détaille ce qu'a apporté chaque version.

## Comment lire ce livre

Le livre est organisé en parties :

1. **Découvrir Go** : installation, premier programme et outil `go`.
2. **Le langage** : types, fonctions, structures, interfaces, erreurs,
   génériques.
3. **La bibliothèque standard** : les packages à connaître.
4. **L'évolution de Go** : ce que chaque version a apporté.
5. **Approfondissements** : des chapitres construits autour d'exemples complets
   (concurrence, templates, organisation d'un projet…).

Le livre décrit **Go 1.27**. Lorsqu'une fonctionnalité est apparue récemment,
la version qui l'a introduite est indiquée. Cela permet de savoir si elle est
disponible dans un projet qui utilise une version plus ancienne :

> [!NOTE]
> **Go 1.22** — `for i := range 10` parcourt les entiers de 0 à 9.

Le code des exemples se trouve dans le dossier `examples/` du
[dépôt du livre](https://github.com/laurentpoirierfr/golang-demystified). Il est
compilé et testé à chaque modification.
