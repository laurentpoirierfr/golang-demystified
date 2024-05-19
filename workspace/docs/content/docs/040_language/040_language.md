---
title: 'Language'
date: 
weight: 6
---

# Language

### Les types

En Go, les principaux types sont les types de base (comme int, float64, string), les types composites (comme les tableaux, les tranches, les cartes, les structures et les interfaces) et les types définis par l'utilisateur (alias de types et types structurés).

Voici un exemple d'instanciation pour quelques types :

* **Type de base** :

```go
var age int = 30
```

* **Type tableau** :

```go
var numbers [5]int
```
* **Type slice** : 

```go
var fruits []string
```

* **Type map** :

```go
var person map[string]string
```


* **Type structure** :

```go
type Person struct {
    Name string
    Age  int
}

var john Person
```

* **Type interface** :

```go
type Shape interface {
    Area() float64
}
```

* **Alias de type** :

```go
type Celsius float64

var temperature Celsius = 25.5
```

### time

En Go, il n'y a pas de type de données prédéfini pour les dates comme dans certains autres langages de programmation. Cependant, vous pouvez utiliser le type **time.Time** du package time pour représenter les dates et les heures. Voici un exemple d'instanciation d'une variable de type **time.Time* :

```go
import "time"

func main() {
    // Instanciation d'une variable de type time.Time représentant la date et l'heure actuelles
    currentTime := time.Now()
    
    fmt.Println("Date et heure actuelles :", currentTime)
}

```
Le package **time** offre également des fonctionnalités pour manipuler les dates, les heures, les durées et les intervalles de temps.


Pour formater une date en Go, vous pouvez utiliser la méthode Format du type time.Time. Voici un exemple de formatage de date :

```go
import (
    "fmt"
    "time"
)

func main() {
    currentTime := time.Now()
    
    // Formater la date selon le format RFC3339
    formattedDate := currentTime.Format(time.RFC3339)
    fmt.Println("Date formatée selon RFC3339:", formattedDate)
    
    // Formater la date selon un format personnalisé
    customFormat := "02 Jan 2006 15:04 MST"
    customFormattedDate := currentTime.Format(customFormat)
    fmt.Println("Date formatée selon un format personnalisé:", customFormattedDate)
}
```

Cet exemple utilise le format **RFC3339** pour formater la date selon un format standardisé. Vous pouvez également spécifier votre propre format en utilisant une chaîne de format personnalisée, où les composants de la date et de l'heure sont représentés par des symboles spécifiques (comme "02" pour le jour, "Jan" pour le mois en abrégé, etc.).

Pour convertir une valeur de type **time.Time** au format RFC3339 en JSON en utilisant Go, vous pouvez définir une structure avec un champ de type **time.Time**, puis utiliser l'encodage JSON de la bibliothèque standard pour encoder la structure. Voici un exemple :

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

// Structure pour représenter une date au format RFC3339
type RFC3339Date struct {
    Date time.Time `json:"date"`
}

func main() {
    // Date au format RFC3339
    rfc3339Str := "2024-05-12T15:04:05Z"
    rfc3339Time, err := time.Parse(time.RFC3339, rfc3339Str)
    if err != nil {
        fmt.Println("Erreur lors de l'analyse de la date RFC3339:", err)
        return
    }

    // Création d'une instance de la structure avec la date au format RFC3339
    rfc3339Date := RFC3339Date{Date: rfc3339Time}

    // Encodage de la structure en JSON
    jsonData, err := json.Marshal(rfc3339Date)
    if err != nil {
        fmt.Println("Erreur lors de l'encodage JSON:", err)
        return
    }

    // Affichage du JSON encodé
    fmt.Println("JSON encodé:", string(jsonData))
}
```

Dans cet exemple, nous définissons une structure **RFC3339Date** avec un champ Date de type **time.Time**. Nous analysons ensuite la chaîne au format RFC3339 en utilisant time.Parse, créons une instance de la structure avec cette date, puis encodons cette structure en JSON à l'aide de **json.Marshal**.


Pour décoder un JSON contenant une date au format RFC3339 en utilisant Go, vous pouvez définir une structure correspondante avec un champ de type **time.Time** et utiliser json.Unmarshal pour décompresser le JSON dans cette structure. Voici un exemple :

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

// Structure pour représenter une date au format RFC3339
type RFC3339Date struct {
    Date time.Time `json:"date"`
}

func main() {
    // JSON contenant une date au format RFC3339
    jsonData := []byte(`{"date":"2024-05-12T15:04:05Z"}`)

    // Déclaration d'une variable pour stocker les données JSON décodées
    var rfc3339Date RFC3339Date

    // Décodage JSON dans la structure
    err := json.Unmarshal(jsonData, &rfc3339Date)
    if err != nil {
        fmt.Println("Erreur lors du décodage JSON:", err)
        return
    }

    // Affichage de la date décodée
    fmt.Println("Date décodée:", rfc3339Date.Date)
}
```

Dans cet exemple, nous définissons une structure **RFC3339Date** avec un champ Date de type **time.Time**. Nous utilisons ensuite **json.Unmarshal** pour décompresser le JSON dans cette structure. La date décodée sera stockée dans le champ **Date** de la structure **RFC3339Date**.