---
title: 'Language'
date: 
weight: 600
---

# Language

## Instructions de contrôle

### if

#### if - Conditionnel de base

L'instruction **if** en Go est utilisée pour exécuter un bloc de code si une condition est vraie. Elle peut être accompagnée d'un **else** ou de plusieurs **else if** pour gérer des cas supplémentaires.

##### Syntaxe de base

```go
if condition {
    // bloc de code si la condition est vraie
} else if autreCondition {
    // bloc de code si l'autre condition est vraie
} else {
    // bloc de code si aucune des conditions précédentes n'est vraie
}
```

##### Exemple

```go
package main

import "fmt"

func main() {
    a := 10
    b := 20

    if a > b {
        fmt.Println("a est plus grand que b")
    } else if a < b {
        fmt.Println("a est plus petit que b")
    } else {
        fmt.Println("a est égal à b")
    }
}
```

### switch

#### switch - Multiples conditions

L'instruction **switch** est utilisée pour sélectionner l'exécution d'un des nombreux blocs de code en fonction de la valeur d'une expression. Elle est souvent plus propre et plus lisible que l'utilisation de multiples **if-else** imbriqués.

##### Syntaxe de base

```go
switch expression {
case valeur1:
    // bloc de code si expression == valeur1
case valeur2:
    // bloc de code si expression == valeur2
default:
    // bloc de code si expression ne correspond à aucune des valeurs
}
```

##### Exemple

```go
package main

import "fmt"

func main() {
    day := "mardi"

    switch day {
    case "lundi":
        fmt.Println("Aujourd'hui c'est lundi")
    case "mardi":
        fmt.Println("Aujourd'hui c'est mardi")
    case "mercredi":
        fmt.Println("Aujourd'hui c'est mercredi")
    default:
        fmt.Println("Aujourd'hui c'est un autre jour de la semaine")
    }
}
```


#### Utilisation de la clause switch avec une expression vide

Dans Go, une expression vide dans la clause **switch** est équivalente à une série d'instructions **if-else**. Cela permet d'écrire un code plus concis lorsqu'il y a plusieurs conditions à vérifier.

##### Syntaxe de base

```go
switch {
case condition1:
    // bloc de code si condition1 est vraie
case condition2:
    // bloc de code si condition2 est vraie
default:
    // bloc de code si aucune des conditions précédentes n'est vraie
}
```

##### Exemple

```go
package main

import "fmt"

func main() {
    number := 10

    switch {
    case number < 0:
        fmt.Println("Le nombre est négatif")
    case number == 0:
        fmt.Println("Le nombre est zéro")
    default:
        fmt.Println("Le nombre est positif")
    }
}
```
Dans cet exemple, la clause **switch** utilise une expression vide (**switch { ... }**). Chaque case spécifie une condition à vérifier. Si aucune des conditions n'est vraie, le bloc **default** est exécuté.

Bien que cela puisse ressembler à un "switch conditionnel", en réalité, c'est simplement une utilisation de la syntaxe de **switch** sans une expression explicite à évaluer. Cependant, cela peut être pratique pour gérer plusieurs conditions de manière concise.


## Les types

En Go, les principaux types sont les types de base (comme int, float64, string), les types composites (comme les tableaux, les tranches, les cartes, les structures et les interfaces) et les types définis par l'utilisateur (alias de types et types structurés).

Voici un exemple d'instanciation pour quelques types :

### Type de base

| Catégorie  | Type   | Description | Exemple d'instanciation |
| ------- | -------- | -------- | -------- |
|Booléens	| **bool**	|Valeur booléenne (vrai ou faux)	| **var flag bool = true** |
|Numériques	|**int**	|Entier avec une taille dépendante de l'architecture	|**var age int = 30** |
||**int8**	|Entier sur 8 bits (de -128 à 127)	|**var smallInt int8 = 100** |
||**int16**	|Entier sur 16 bits (de -32768 à 32767)	|**var mediumInt int16 = 20000** |
||**int32**	|Entier sur 32 bits (de -2147483648 à 2147483647)	|**var largeInt int32 = 1000000** |
||**int64**	|Entier sur 64 bits (de -9223372036854775808 à 9223372036854775807)	|**var xLargeInt int64 = 1000000000** |
||**uint**	|Entier non signé, taille dépendante de l'architecture	|**var posInt uint = 42** |
||**uint8**	|Entier non signé sur 8 bits (de 0 à 255)	|**var byteVal uint8 = 255** |
||**uint16**	|Entier non signé sur 16 bits (de 0 à 65535)	|**var mediumUInt uint16 = 60000** |
||**uint32**	|Entier non signé sur 32 bits (de 0 à 4294967295)	|**var largeUInt uint32 = 4000000000** |
||**uint64**	|Entier non signé sur 64 bits (de 0 à 18446744073709551615)	|**var xLargeUInt uint64 = 10000000000** |
||**uintptr**	|Entier non signé pour stocker un pointeur	|**var ptr uintptr = uintptr(unsafe.Pointer(&x))** |
||**float32**	|Nombre à virgule flottante sur 32 bits	|**var pi float32 = 3.14** |
||**float64**	|Nombre à virgule flottante sur 64 bits	|**var e float64 = 2.71828** |
||**complex64**	|Nombre complexe avec des parties réelles et imaginaires en float32	|**var z complex64 = 1 + 2i** |
||**complex128**	|Nombre complexe avec des parties réelles et imaginaires en float64	|**var w complex128 = 2 + 3i**|
|Caractères	|**byte**	|Alias pour uint8, représente un octet	|**var b byte = 'a'** |
||**rune**	|Alias pour int32, représente un caractère Unicode	|**var r rune = 'e'** |
|Chaînes	|**string**	|Séquence de caractères	|**var name string = "Go"**|

Ce tableau inclut les types de base les plus courants en Go, couvrant les booléens, les types numériques (entiers signés et non signés, nombres flottants, nombres complexes), les caractères et les chaînes.

### Type tableau et slice

En Go, les tableaux et les tranches (slices) sont deux structures de données différentes, chacune avec ses propres caractéristiques et usages. Voici les principales différences entre les deux :

#### Tableaux

* Taille fixe :
    - La taille d'un tableau est fixe et définie lors de sa création.
    - Exemple : var arr [5]int

* Type :
    - Le type du tableau inclut sa taille. Par exemple, [5]int et [10]int sont des types différents.
    - Deux tableaux de tailles différentes ne sont pas compatibles, même s'ils contiennent les mêmes types d'éléments.

* Allocation :
    - Les tableaux sont alloués en mémoire statiquement. La mémoire pour un tableau est allouée au moment de la déclaration.

* Usage :
    - Les tableaux sont rarement utilisés directement en Go en raison de leur taille fixe et de leur manque de flexibilité.


#### Slices

* Taille dynamique :
    - Les tranches sont dynamiques et peuvent changer de taille.
    - Exemple : var slice []int

* Type :
    - Les tranches sont des vues flexibles sur des tableaux. Le type d'une tranche ne dépend pas de sa longueur.
    - Par exemple, []int est un type de tranche pour une tranche de n'importe quelle longueur d'entiers.

* Allocation :
    - Les tranches sont des références à des tableaux sous-jacents. Une tranche a trois composants : un pointeur vers un tableau, une longueur et une capacité.
    - La capacité d'une tranche est la taille du tableau sous-jacent à partir duquel la tranche est dérivée.

* Usage :
    - Les tranches sont largement utilisées en Go en raison de leur flexibilité. Vous pouvez facilement augmenter ou réduire leur taille et elles peuvent partager des parties d'un tableau sous-jacent.


#### Exemples 

##### Tableau

```go
package main

import "fmt"

func main() {
    // Déclaration d'un tableau de taille fixe
    var arr [5]int = [5]int{1, 2, 3, 4, 5}
    fmt.Println("Tableau :", arr)
}
```

##### Slice

```go
package main

import "fmt"

func main() {
    // Déclaration d'une tranche
    var slice []int = []int{1, 2, 3, 4, 5}
    fmt.Println("Slice :", slice)

    // Ajouter un élément à la tranche
    slice = append(slice, 6)
    fmt.Println("Slice après append :", slice)

    // Sous-tranchage
    subSlice := slice[1:4]
    fmt.Println("Sous-Slice :", subSlice)
}
```

##### Résumé

* **Tableaux** : Taille fixe, type incluant la taille, allocation statique.
* **Slices** : Taille dynamique, type indépendant de la longueur, référence à un tableau sous-jacent, très flexibles et couramment utilisées.

En général, vous utiliserez des tranches pour la plupart des travaux avec des collections en Go en raison de leur flexibilité et de leur facilité d'utilisation.


### Type tableau et Slice

Voici un exemple en Go montrant comment instancier un tableau, puis itérer sur ses éléments de manière traditionnelle et avec range :

```go
package main

import (
    "fmt"
)

func main() {
    // Instanciation d'un tableau de 5 entiers
    var numbers = [5]int{10, 20, 30, 40, 50}

    // Itération traditionnelle
    fmt.Println("Itération traditionnelle :")
    for i := 0; i < len(numbers); i++ {
        fmt.Printf("Index %d : %d\n", i, numbers[i])
    }

    // Itération avec range
    fmt.Println("\nItération avec range :")
    for index, value := range numbers {
        fmt.Printf("Index %d : %d\n", index, value)
    }
}
```
#### Explications

* Instanciation du tableau :

```go
var numbers = [5]int{10, 20, 30, 40, 50}
```
Ici, nous créons un tableau numbers de 5 entiers initialisés avec les valeurs spécifiées.

* Itération traditionnelle :

```go
for i := 0; i < len(numbers); i++ {
    fmt.Printf("Index %d : %d\n", i, numbers[i])
}
```
Cette boucle for utilise un index i pour accéder aux éléments du tableau en utilisant numbers[i].

* Itération avec range :

```go
for index, value := range numbers {
    fmt.Printf("Index %d : %d\n", index, value)
}
```
Cette boucle for utilise range pour itérer sur les éléments du tableau. La variable index représente l'indice actuel et value représente la valeur de l'élément à cet indice.

#### Résultat attendu :

```bash
Itération traditionnelle :
Index 0 : 10
Index 1 : 20
Index 2 : 30
Index 3 : 40
Index 4 : 50

Itération avec range :
Index 0 : 10
Index 1 : 20
Index 2 : 30
Index 3 : 40
Index 4 : 50

```


### Type map

Voici un exemple d'instanciation d'une map en Go, ainsi qu'une itération sur ses éléments :



```go
package main

import (
    "fmt"
)

func main() {
    // Instanciation d'une carte (map) avec des clés de type string et des valeurs de type int
    var ageMap = map[string]int{
        "Alice": 30,
        "Bob":   25,
        "Charlie": 35,
    }

    // Itération traditionnelle avec range
    fmt.Println("Itération avec range :")
    for key, value := range ageMap {
        fmt.Printf("Clé : %s, Valeur : %d\n", key, value)
    }
}

```


#### Explications :

* Instanciation d'une map :

```go
var ageMap = map[string]int{
    "Alice": 30,
    "Bob":   25,
    "Charlie": 35,
}
```

Ici, nous créons une map ageMap avec des clés de type **string** et des valeurs de type **int**. Nous initialisons cette carte avec quelques paires clé-valeur.


* Itération avec range :

```go
for key, value := range ageMap {
    fmt.Printf("Clé : %s, Valeur : %d\n", key, value)
}
```
Cette boucle **for** utilise **range** pour itérer sur les éléments de la map. La variable **key** représente la clé actuelle et **value** représente la valeur associée à cette clé.

#### Résultat attendu :

```go
Itération avec range :
Clé : Alice, Valeur : 30
Clé : Bob, Valeur : 25
Clé : Charlie, Valeur : 35
```

Cet exemple montre comment créer et initialiser une map en Go, et comment parcourir toutes les paires clé-valeur de la carte en utilisant une boucle **for** avec **range**.


### Alias de type

```go
type Celsius float64

var temperature Celsius = 25.5
```

### Type time vs date

En Go, il n'y a pas de type de données prédéfini pour les dates comme dans certains autres langages de programmation. Cependant, vous pouvez utiliser le type **time.Time** du package time pour représenter les dates et les heures. Voici un exemple d'instanciation d'une variable de type **time.Time** :

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


