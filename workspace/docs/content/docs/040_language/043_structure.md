---
title: 'Type Structure'
date: 
weight: 620
---

### Type structure

En Go, une structure (ou struct) est un type de données composite qui regroupe des champs sous un même nom. Une structure permet de définir des objets avec des propriétés et des méthodes associées.


#### Définition d'une structure

Une structure est définie en listant ses champs et leurs types. Voici un exemple simple :

```go
type Person struct {
    Name string
    Age  int
}

var john Person
```


#### Instanciation via une fabrique

Une fabrique (**factory**) est une fonction qui retourne une instance de la structure. Cela permet d'encapsuler la logique de création de l'objet.

#### Exemple complet

Voici un exemple complet montrant la définition d'une structure, une fabrique pour créer des instances, l'instanciation via un pointeur, et quelques méthodes associées à la structure :


```go
package main

import (
    "fmt"
)

// Définition de la structure Person
type Person struct {
    Name string
    Age  int
}

// Fabrique pour créer une nouvelle instance de Person
func NewPerson(name string, age int) *Person {
    return &Person{Name: name, Age: age}
}

// Méthode pour afficher les détails de la personne
func (p *Person) Display() {
    fmt.Printf("Name: %s, Age: %d\n", p.Name, p.Age)
}

// Méthode pour faire vieillir la personne d'un an
func (p *Person) Birthday() {
    p.Age++
}

func main() {
    // Instanciation d'une nouvelle personne via la fabrique
    person := NewPerson("Alice", 30)

    // Utilisation des méthodes de la structure
    person.Display()  // Affiche: Name: Alice, Age: 30
    person.Birthday() // Fait vieillir Alice d'un an
    person.Display()  // Affiche: Name: Alice, Age: 31
}
```

#### Explications

* Définition de la structure Person :

```go
type Person struct {
    Name string
    Age  int
}
```

La structure **Person** a deux champs : **Name** de type **string** et **Age** de type **int**.

* Fabrique pour créer une nouvelle instance de Person :

```go
func NewPerson(name string, age int) *Person {
    return &Person{Name: name, Age: age}
}
```

La fonction **NewPerson** prend des paramètres pour le nom et l'âge, crée une nouvelle instance de **Person** et retourne un pointeur vers cette instance.

* Méthode Birthday pour faire vieillir la personne d'un an :


```go
func (p *Person) Birthday() {
    p.Age++
}
```

La méthode Birthday incrémente l'âge de la personne.

* Instanciation et utilisation :

```go
person := NewPerson("Alice", 30)
person.Display()  // Affiche: Name: Alice, Age: 30
person.Birthday() // Fait vieillir Alice d'un an
person.Display()  // Affiche: Name: Alice, Age: 31
```

Nous créons une nouvelle personne nommée Alice âgée de 30 ans en utilisant la fabrique **NewPerson**. Nous utilisons ensuite les méthodes **Display** et **Birthday** pour afficher et modifier les propriétés de l'instance.

#### Résumé

Les structures en Go permettent de regrouper des données et des méthodes associées. L'utilisation d'une fabrique pour créer des instances de structures encapsule la logique de création, et les méthodes associées à la structure permettent de manipuler les données de manière organisée.