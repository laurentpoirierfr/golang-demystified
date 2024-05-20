---
title: 'Organisation packages clean architectures'
date: 
weight: 720
---

## Exemple de structure de package

```bash
/app
    /cmd
        /webapi
            main.go
        /consumer
            main.go
    /internal
        /domain
            /entity
                post.go
                user.go
            /repository
                post_repository.go
                user_repository.go
        /usecase
            /post
                create_post.go
                get_post.go
                ...
            /user
                create_user.go
                get_user.go
                ...
        /delivery
            /http
                post_handler.go
                user_handler.go
            /kafka
                kafka_producer.go
                kafka_consumer.go
        /repository
            /postgres
                post_repository.go
                user_repository.go
            /mongo
                post_repository.go
                user_repository.go
        /framework
            /config
                config.go
            /middleware
                middleware.go
    /pkg
        /domain
            /entity
            /repository
        /usecase
        /delivery
            /http
            /kafka
        /repository
            /postgres
            /mongo
        /framework
            /config
            /middleware
```


Cette structure conserve la distinction entre le code interne et le code destiné à être utilisé par d'autres applications, tout en maintenant la flexibilité offerte par les répertoires **cmd**, **internal** et **pkg**.


#### Package domain

Ce package contient les entités de votre domaine ainsi que les interfaces des repositories.

```bash
/domain
    /entity
    /repository
```

* **entity** : Définit les entités de votre domaine.
* **repository** : Définit les interfaces des repositories pour accéder aux données.

#### Package usecase

Ce package contient les cas d'utilisation de votre application.

```bash
/usecase
    /auth
    /post
    /user
    ...
```

#### Package delivery

Ce package contient les handlers HTTP et les brokers Kafka pour gérer les requêtes entrantes et sortantes.

```bash
/delivery
    /http
    /kafka
```

* **http** : Contient les handlers HTTP pour votre API web.
* **kafka** : Contient les producers et consumers Kafka.

####  Package repository

Ce package contient les implémentations concrètes des repositories définies dans le package **domain/repository**.

```bash
/repository
    /postgres
    /mongo
    ...
```

Chaque sous-package contient les implémentations spécifiques des repositories pour chaque système de stockage de données.


#### Package framework

Ce package contient le code dépendant du framework, comme les configurations et les middlewares.

```bash
/framework
    /config
    /middleware
```

* **config** : Contient les configurations de votre application.
* **middleware** : Contient les middlewares HTTP.