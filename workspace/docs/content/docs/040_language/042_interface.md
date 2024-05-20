---
title: 'Type Interface'
date: 
weight: 620
---

### Type interface

En Go, une interface est un type qui spécifie un ensemble de méthodes que d'autres types doivent implémenter. Les interfaces sont utilisées pour définir le comportement attendu sans spécifier l'implémentation concrète. Un type qui implémente toutes les méthodes d'une interface est considéré comme une instance de cette interface.

#### Définition d'une interface

Une interface est définie en listant les méthodes qu'elle contient. Voici un exemple d'interface simple :

```go
type Speaker interface {
    Speak() string
}
```

#### Implémentation de l'interface

Un type implémente une interface en définissant toutes les méthodes de cette interface. Il n'y a pas de mot-clé explicite pour indiquer qu'un type implémente une interface ; cela se fait implicitement.

#### Exemple complet

Voici un exemple complet montrant la définition d'une interface, l'implémentation de cette interface par différents types, et l'utilisation de l'interface :

```go
package main

import (
    "fmt"
)

// Définition de l'interface Speaker
type Speaker interface {
    Speak() string
}

// Type Dog qui implémente l'interface Speaker
type Dog struct {
    Name string
}

func (d Dog) Speak() string {
    return "Woof! My name is " + d.Name
}

// Type Cat qui implémente l'interface Speaker
type Cat struct {
    Name string
}

func (c Cat) Speak() string {
    return "Meow! My name is " + c.Name
}

// Fonction qui prend un Speaker en paramètre
func MakeSpeak(s Speaker) {
    fmt.Println(s.Speak())
}

func main() {
    // Création d'instances de Dog et Cat
    dog := Dog{Name: "Buddy"}
    cat := Cat{Name: "Whiskers"}

    // Utilisation de la fonction MakeSpeak avec différentes implémentations de Speaker
    MakeSpeak(dog)
    MakeSpeak(cat)
}
```

#### Explications

* Définition de l'interface Speaker :

```go
type Speaker interface {
    Speak() string
}
```

L'interface Speaker contient une seule méthode Speak qui retourne une chaîne de caractères.

* Implémentation de l'interface par le type Dog :


```go
type Dog struct {
    Name string
}

func (d Dog) Speak() string {
    return "Woof! My name is " + d.Name
}
```

Le type Dog implémente la méthode Speak définie par l'interface Speaker.

* Implémentation de l'interface par le type Cat :

```go
type Cat struct {
    Name string
}

func (c Cat) Speak() string {
    return "Meow! My name is " + c.Name
}
```

Le type Cat implémente également la méthode Speak.

* Utilisation de l'interface :

```go
func MakeSpeak(s Speaker) {
    fmt.Println(s.Speak())
}
```

La fonction **MakeSpeak** prend un paramètre de type **Speaker** et appelle la méthode **Speak** sur ce paramètre.


* Appel de la fonction avec différentes implémentations :

```go
dog := Dog{Name: "Buddy"}
cat := Cat{Name: "Whiskers"}

MakeSpeak(dog)
MakeSpeak(cat)
```

Les instances de **Dog** et **Cat** sont passées à la fonction **MakeSpeak**, démontrant le polymorphisme en Go.


#### Résultat attendu

```go
Woof! My name is Buddy
Meow! My name is Whiskers
```

Dans cet exemple, l'interface **Speaker** est utilisée pour abstraire le comportement de **Dog** et **Cat**, permettant de traiter des instances de ces types de manière uniforme via l'interface.