---
title: 'Opérateurs'
date: 
weight: 610
---

## Les opérateurs

Voici la liste des principaux opérateurs en Go, avec des explications et des exemples pour chacun :

### Opérateurs arithmétiques

#### Addition (+)

* Ajoute deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := a + b
    fmt.Println("Addition :", result) // Affiche: Addition : 15
}
```

#### Soustraction (-)

* Soustrait une valeur d'une autre.
* Exemple:

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := a - b
    fmt.Println("Soustraction :", result) // Affiche: Soustraction : 5
}
```

#### Multiplication (*)

* Multiplie deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := a * b
    fmt.Println("Multiplication :", result) // Affiche: Multiplication : 50
}
```

#### Division (/)

* Divise une valeur par une autre.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := a / b
    fmt.Println("Division :", result) // Affiche: Division : 2
}
```

#### Modulo (%)

* Retourne le reste de la division entière de deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 3
    result := a % b
    fmt.Println("Modulo :", result) // Affiche: Modulo : 1
}
```

### Opérateurs de comparaison

#### Égal à (==)

* Vérifie si deux valeurs sont égales.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 10
    result := (a == b)
    fmt.Println("Égal à :", result) // Affiche: Égal à : true
}
```

#### Différent de (!=)

* Vérifie si deux valeurs sont différentes.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := (a != b)
    fmt.Println("Différent de :", result) // Affiche: Différent de : true
}
```

#### Plus grand que (>)

* Vérifie si une valeur est plus grande qu'une autre.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    result := (a > b)
    fmt.Println("Plus grand que :", result) // Affiche: Plus grand que : true
}
```

#### Plus petit que (<)

* Vérifie si une valeur est plus petite qu'une autre.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 5
    b := 10
    result := (a < b)
    fmt.Println("Plus petit que :", result) // Affiche: Plus petit que : true
}
```

#### Plus grand ou égal à (>=)

* Vérifie si une valeur est plus grande ou égale à une autre.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 10
    result := (a >= b)
    fmt.Println("Plus grand ou égal à :", result) // Affiche: Plus grand ou égal à : true
}
```

#### Plus petit ou égal à (<=)

* Vérifie si une valeur est plus petite ou égale à une autre.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 5
    b := 10
    result := (a <= b)
    fmt.Println("Plus petit ou égal à :", result) // Affiche: Plus petit ou égal à : true
}
```

### Opérateurs logiques

#### Et logique (&&)

* Retourne vrai si les deux opérandes sont vrais.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := true
    b := false
    result := a && b
    fmt.Println("Et logique :", result) // Affiche: Et logique : false
}
```

#### Ou logique (||)

* Retourne vrai si au moins un des opérandes est vrai.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := true
    b := false
    result := a || b
    fmt.Println("Ou logique :", result) // Affiche: Ou logique : true
}
```

#### Non logique (!)

* Inverse la valeur booléenne.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := true
    result := !a
    fmt.Println("Non logique :", result) // Affiche: Non logique : false
}
```

### Opérateurs de bits

#### Et bit à bit (&)

* Effectue un ET logique bit à bit sur deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10 // 1010 en binaire
    b := 3  // 0011 en binaire
    result := a & b // 0010 en binaire
    fmt.Println("Et bit à bit :", result) // Affiche: Et bit à bit : 2
}
```

#### Ou bit à bit (|)

* Effectue un OU logique bit à bit sur deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10 // 1010 en binaire
    b := 3  // 0011 en binaire
    result := a | b // 1011 en binaire
    fmt.Println("Ou bit à bit :", result) // Affiche: Ou bit à bit : 11
}
```

#### XOR bit à bit (^)

* Effectue un XOR logique bit à bit sur deux valeurs.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10 // 1010 en binaire
    b := 3  // 0011 en binaire
    result := a ^ b // 1001 en binaire
    fmt.Println("XOR bit à bit :", result) // Affiche: XOR bit à bit : 9
}
```

#### Décalage à gauche (<<)

* Décale les bits d'une valeur vers la gauche.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 3 // 0011 en binaire
    result := a << 2 // 1100 en binaire
    fmt.Println("Décalage à gauche :", result) // Affiche: Décalage à gauche : 12
}
```

#### Décalage à droite (>>)

* Décalage à droite (>>)
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 12 // 1100 en binaire
    result := a >> 2 // 0011 en binaire
    fmt.Println("Décalage à droite :", result) // Affiche: Décalage à droite : 3
}
```

### Opérateurs d'affectation

#### Affectation (=)

* Assigne une valeur à une variable.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    b := 5
    a = b
    fmt.Println("Affectation :", a) // Affiche: Affectation : 5
}
```

#### Affectation et addition (+=)

* Ajoute la valeur de droite à la variable de gauche.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    a += 5
    fmt.Println("Affectation et addition :", a) // Affiche: Affectation et addition : 15
}
```

#### Affectation et soustraction (-=)

* Soustrait la valeur de droite à la variable de gauche.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    a -= 5
    fmt.Println("Affectation et soustraction :", a) // Affiche: Affectation et soustraction : 5
}
```

#### Affectation et multiplication (*=)

* Multiplie la variable de gauche par la valeur de droite.
* Exemple :


```go
package main
import "fmt"
func main() {
    a := 10
    a *= 5
    fmt.Println("Affectation et multiplication :", a) // Affiche: Affectation et multiplication : 50
}
```

#### Affectation et division (/=)

* Divise la variable de gauche par la valeur de droite.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    a /= 5
    fmt.Println("Affectation et division :", a) // Affiche: Affectation et division : 2
}
```

#### Affectation et modulo (%=)

* Assigne à la variable de gauche le reste de la division par la valeur de droite.
* Exemple :

```go
package main
import "fmt"
func main() {
    a := 10
    a %= 3
    fmt.Println("Affectation et modulo :", a) // Affiche: Affectation et modulo : 1
}
```

### Opérateurs divers

#### Opérateur d'incrément (++)

* Incrémente la valeur de la variable de 1.
* Exemple : 

```go
package main
import "fmt"
func main() {
    a := 10
    a++
    fmt.Println("Incrément :", a) // Affiche: Incrément : 11
}
```

#### Opérateur de décrément (--)

* Décrémente la valeur de la variable de 1.
* Exemple : 

```go
package main
import "fmt"
func main() {
    a := 10
    a--
    fmt.Println("Décrément :", a) // Affiche: Décrément : 9
}
```

Ces opérateurs couvrent la plupart des besoins de manipulation des variables et des expressions en Go. Chaque opérateur est conçu pour une opération spécifique, et leur utilisation correcte est essentielle pour écrire du code Go efficace et lisible.￼