---
title: 'Packages'
date: 
weight: 7
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


## Exemple d'organisation pour un projet web


Voici une structure et un exemple de code pour un projet Go utilisant le framework web Gin. Cette organisation inclut des packages pour les API, les services, les dépôts de données, et les connecteurs de données.

### Structure du projet

```bash
myproject/
├── cmd/
│   └── myproject/
│       └── main.go
├── internal/
│   └── myproject/
│       ├── api/
│       │   ├── handlers/
│       │   │   └── user_handler.go
│       │   └── router.go
│       ├── service/
│       │   └── user_service.go
│       ├── repository/
│       │   ├── user_repository.go
│       │   └── db.go
│       └── config/
│           └── config.go
├── pkg/
│   └── myproject/
│       ├── models/
│       │   └── user.go
│       ├── utils/
│       │   └── utils.go
│       └── connectors/
│           ├── redis/
│           │   └── redis_client.go
│           └── sqldb/
│               └── sqldb_client.go
└── go.mod
```

### Exemple de code

**cmd/myproject/main.go**

```go
package main

import (
	"log"
	"myproject/internal/myproject/api"
	"myproject/internal/myproject/config"
)

func main() {
	// Charger la configuration
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	// Initialiser le routeur
	router := api.NewRouter()

	// Démarrer le serveur
	log.Printf("Starting server on %s", conf.ServerAddress)
	if err := router.Run(conf.ServerAddress); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
```

**internal/myproject/api/router.go**

```go
package api

import (
	"myproject/internal/myproject/api/handlers"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// Définir les routes
	r.GET("/users/:id", handlers.GetUser)

	return r
}
```

**internal/myproject/api/handlers/user_handler.go**

```go
package handlers

import (
	"myproject/internal/myproject/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := service.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
```

**internal/myproject/service/user_service.go**

```go
package service

import (
	"myproject/internal/myproject/repository"
	"myproject/pkg/myproject/models"
)

func GetUserByID(id int) (*models.User, error) {
	return repository.GetUserByID(id)
}
```

**internal/myproject/repository/user_repository.go**

```go
package repository

import "myproject/pkg/myproject/models"

func GetUserByID(id int) (*models.User, error) {
	// Simuler une récupération de base de données
	return &models.User{ID: id, Name: "John Doe"}, nil
}
```

**internal/myproject/config/config.go**

```go
package config

import (
	"os"
)

type Config struct {
	ServerAddress string
}

func LoadConfig() (*Config, error) {
	return &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
```

### Explication des packages

- **cmd** :

    * Contient les programmes exécutables de votre projet. Le fichier main.go initialise la configuration et démarre le serveur Gin.

- **internal** :

    * **api** :
        - **handlers** : Contient les gestionnaires de requêtes (handlers) pour les différentes routes de l'API.
        - **router.go** : Initialise le routeur Gin et définit les routes.

    * **service** : Contient la logique métier de l'application.
    * **repository** : Contient la logique d'accès aux données.
    * **config** : Gère la configuration de l'application.

- **pkg** :

    * **models** : Contient les définitions de structures de données.
    * **utils** : Contient des fonctions utilitaires réutilisables.
    * **connectors** : Contient les connecteurs de données pour les différentes sources de données (par exemple, Redis, SQL).

Cette structure organise clairement les différentes parties de votre application web, en séparant les responsabilités et en assurant une modularité et une extensibilité efficaces.

￼
￼
￼
￼
￼
￼
