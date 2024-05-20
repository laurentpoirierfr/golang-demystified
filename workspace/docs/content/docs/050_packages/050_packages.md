---
title: 'Organisation packages standard'
date: 
weight: 700
---

# Packages

## Exemple de structure du projet

```bash
myproject/
├── cmd/
│   └── myproject/
│       └── main.go
├── internal/
│   └── myproject/
│       ├── service/
│       │   └── service.go
│       └── repository/
│           └── repository.go
├── pkg/
│   └── myproject/
│       ├── models/
│       │   └── models.go
│       └── utils/
│           └── utils.go
└── go.mod
```

### Description des packages

* **cmd** :

    - Ce répertoire contient les programmes exécutables de votre projet. Chaque sous-répertoire dans **cmd** contient un point d'entrée principal (**main.go**) pour un binaire spécifique.

    - Exemple : **cmd/myproject/main.go** est le point d'entrée de l'application principale.


* **internal** :

    - Ce répertoire contient le code qui est destiné à être utilisé uniquement au sein de votre projet. Les packages dans **internal** ne peuvent pas être importés par des projets externes.
    
    - Exemple :
        * **internal/myproject/service/service.go** contient la logique métier de votre application.
        * **internal/myproject/repository/repository.go** contient la logique d'accès aux données (par exemple, accès à la base de données).


*  **pkg** :

    - Ce répertoire contient les packages qui peuvent être utilisés par d'autres projets ou par plusieurs composants de votre projet. Ces packages sont réutilisables et exportables.

    - Exemple :
        * **pkg/myproject/models/models.go** contient les définitions de structures de données utilisées dans votre application.
        * **pkg/myproject/utils/utils.go** contient des fonctions utilitaires réutilisables.


### Exemple de code

**cmd/myproject/main.go**

```go
package main

import (
	"fmt"
	"myproject/internal/myproject/service"
)

func main() {
	fmt.Println("Starting myproject")
	service.StartService()
}
```

**internal/myproject/service/service.go**

```go
package service

import (
	"fmt"
	"myproject/internal/myproject/repository"
	"myproject/pkg/myproject/models"
)

func StartService() {
	fmt.Println("Service started")
	user := repository.GetUserByID(1)
	fmt.Printf("User: %+v\n", user)
}
```


**internal/myproject/repository/repository.go**

```go
package repository

import "myproject/pkg/myproject/models"

func GetUserByID(id int) *models.User {
	// Simuler une récupération de base de données
	return &models.User{ID: id, Name: "John Doe"}
}
```

**pkg/myproject/models/models.go**

```go
package models

type User struct {
	ID   int
	Name string
}
```

**pkg/myproject/utils/utils.go**

```go
package utils

import "fmt"

func PrintMessage(msg string) {
	fmt.Println(msg)
}
```

### Explications supplémentaires

- **cmd** : Contient l'implémentation des points d'entrée principaux de votre application. Vous pouvez avoir plusieurs sous-répertoires dans cmd pour différentes applications/binaries.

- **internal** : Organise le code interne de votre application qui ne doit pas être exposé ou utilisé en dehors de ce projet. Cela assure l'encapsulation et la sécurité du code interne.

- **pkg** : Contient du code exportable et réutilisable. D'autres projets peuvent importer et utiliser les packages ici définis.

Cette organisation permet de structurer votre projet de manière modulaire et claire, en séparant les différentes responsabilités et en facilitant la maintenance et la scalabilité du code.


