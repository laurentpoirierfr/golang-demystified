---
title: 'Commandes'
date: 
weight: 500
---

# Commandes

Voici une explication rapide des commandes les plus couramment utilisées avec Go :

- **go mod init** : Initialise un nouveau module Go dans le répertoire courant. Un module est une collection de packages Go qui sont gérés ensemble, généralement stockés dans un référentiel versionné.



Supposons que vous souhaitez créer un nouveau module Go pour un projet appelé **"monprojet"**. Vous pouvez utiliser go mod init pour initialiser un nouveau module dans le répertoire de votre projet.

- Créez un nouveau répertoire pour votre projet et accédez-y dans votre terminal :

```bash
mkdir monprojet
cd monprojet
```


- Initialisez un nouveau module Go en utilisant go mod init avec le nom de votre module. Par exemple, si vous avez l'intention de publier votre code sous **github.com/utilisateur/monprojet**, vous pouvez utiliser :

```bash
go mod init github.com/utilisateur/monprojet
```



- Après avoir exécuté cette commande, un fichier **go.mod** sera créé dans le répertoire de votre projet. Ce fichier contient des informations sur votre module, y compris son chemin d'importation et les dépendances (initialisées à ce stade comme étant les dépendances standard de Go).



```go
module github.com/utilisateur/monprojet

go 1.22

require (
    github.com/pkg1 v1.2.3
    github.com/pkg2 v0.4.0
)

replace (
    github.com/pkg2 => github.com/utilisateur/pkg2 v0.4.1
)

```


- **module github.com/utilisateur/monprojet** : Cette ligne spécifie le chemin d'importation de votre module Go. Cela permet à d'autres programmes Go de référencer votre module en important ce chemin.

- **go 1.22** : Cette ligne spécifie la version minimale de Go requise pour construire votre projet.



- **require** : Cette section liste les dépendances de votre projet, avec leurs noms de module et leurs versions. Par exemple, **github.com/pkg1 v1.2.3** signifie que votre projet dépend de la version 1.2.3 du module github.com/pkg1.


- **replace** : Cette section permet de remplacer les dépendances par d'autres. Cela peut être utile pour utiliser des versions modifiées ou locales de dépendances. Par exemple, **github.com/pkg2 => github.com/utilisateur/pkg2 v0.4.1** remplace la dépendance github.com/pkg2 par github.com/utilisateur/pkg2 à la version v0.4.1.


- Vous pouvez maintenant commencer à développer votre projet en Go ! Vous pouvez ajouter des dépendances à l'aide de la commande **go get** et utiliser des packages externes sans crainte de perturber votre environnement de développement grâce à la gestion des modules.

Cet exemple illustre comment utiliser **go mod init** pour initialiser un nouveau module Go dans un projet et démarrer le développement avec la gestion des modules.



- **go mod tidy** : Analyse les dépendances de votre projet et met à jour le fichier go.mod et, le cas échéant, le fichier go.sum pour refléter les dépendances réellement utilisées par votre code. Il supprime également les dépendances qui ne sont plus nécessaires.



- **go mod vendor** : Copie les dépendances de votre projet dans le répertoire **vendor/** de votre projet. Cela permet de garantir que toutes les dépendances nécessaires sont présentes localement et que le code peut être construit et exécuté de manière isolée.



- **go run** : Compile et exécute immédiatement le fichier Go spécifié. Cela permet d'exécuter rapidement un programme Go sans avoir à le construire séparément.



- **go install** : Compile et installe le package ou le programme Go spécifié. Si le package ou le programme est un exécutable, il sera installé dans le répertoire **bin/** de votre **$GOPATH** ou de votre module.



- **go test** : Exécute les tests unitaires pour le package Go spécifié. Il recherche les fichiers **_test.go** dans le package et exécute les fonctions de test qu'ils contiennent.

Ces commandes constituent un ensemble de base pour gérer les dépendances, construire, exécuter et tester des programmes Go. Elles sont largement utilisées dans le développement quotidien avec Go.


