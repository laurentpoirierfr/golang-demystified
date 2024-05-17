---
title: 'Exploration'
date: 
weight: 4
---

# Exploration


### $GOROOT

Voici une explication rapide des répertoires principaux présents dans **$GOROOT**, qui est le répertoire où est installé Go :

- **bin/** : Ce répertoire contient les exécutables binaires de Go, y compris le binaire de l'interpréteur Go (go) et d'autres outils comme le formateur de code (gofmt), l'outil de gestion de modules (go mod), etc.

- **pkg/** : Ce répertoire contient les packages précompilés, c'est-à-dire les fichiers objets résultant de la compilation des fichiers source Go. Ces fichiers sont utilisés lors de la construction de programmes Go pour accélérer les temps de compilation.

- **src/** : Bien que souvent vide dans le répertoire **$GOROOT** standard, ce répertoire est destiné à contenir les fichiers source des packages standard de Go. Ces fichiers source sont généralement inclus dans le code source de Go lors de l'installation.

- **include/** : Ce répertoire contient les fichiers d'en-tête nécessaires pour la construction de programmes Go qui utilisent des paquets C, C++ ou autres langages compatibles avec Go. Ces fichiers d'en-tête sont utilisés par le compilateur Go lors de la liaison avec des bibliothèques externes.

- **misc/** : Ce répertoire contient diverses ressources et outils supplémentaires liés à Go, tels que des exemples de code, des scripts de test, etc. Ces ressources peuvent être utiles pour explorer et apprendre Go plus en profondeur.


Ces répertoires constituent l'infrastructure de base de l'installation de Go dans le répertoire **$GOROOT**, et chacun joue un rôle spécifique dans le fonctionnement et le développement avec Go.


### $GOPATH

Voici une explication rapide des répertoires principaux présents dans **$GOPATH**, qui est le répertoire où sont stockés les projets et les dépendances pour les applications Go :

- **bin/** : Ce répertoire contient les exécutables binaires des programmes Go que vous avez installés localement à l'aide de la commande **go install**. Les binaires installés ici sont généralement des outils Go tiers que vous utilisez dans votre workflow de développement.

- **pkg/** : Similaire au répertoire **pkg/** dans **$GOROOT**, ce répertoire contient les packages précompilés des projets Go que vous avez construits. Ces packages sont compilés à partir du code source des projets situés dans le répertoire **src/**.

- **src/** : C'est le répertoire principal où vous stockez le code source de vos projets Go. Chaque sous-répertoire de src/ correspond généralement à un chemin d'importation pour les packages Go. Par exemple, si votre projet est **github.com/utilisateur/monprojet**, vous créerez un répertoire **github.com/utilisateur/monprojet** dans **src/** et placerez votre code source là-dedans.

- **pkg/mod/** : À partir de Go 1.11, ce répertoire est utilisé pour stocker les modules Go téléchargés. Lorsque vous utilisez les modules Go pour gérer les dépendances de vos projets, les fichiers source et les dépendances téléchargées sont stockés ici pour une gestion efficace des dépendances.


- **bin/** (à l'intérieur de **$GOPATH/pkg/mod/**) : Ce sous-répertoire contient également des binaires, mais spécifiquement pour les modules Go téléchargés. Ces binaires sont souvent des outils associés à des modules particuliers que vous avez téléchargés et qui sont nécessaires pour la construction ou la gestion de projets Go.

Ces répertoires forment la structure de base de **$GOPATH** et sont utilisés pour stocker les projets, les dépendances et les binaires associés à votre environnement de développement Go.
