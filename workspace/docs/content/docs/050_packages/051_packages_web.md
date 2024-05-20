---
title: 'Organisation packages web'
date: 
weight: 710
---

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
