---
title: "Les opérateurs"
linkTitle: "2. Les opérateurs"
weight: 20
---

## Opérateurs arithmétiques

| Opérateur | Rôle           | Exemple   | Résultat |
|-----------|----------------|-----------|----------|
| `+`       | Addition (et concaténation de chaînes) | `7 + 2` | `9` |
| `-`       | Soustraction   | `7 - 2`   | `5`      |
| `*`       | Multiplication | `7 * 2`   | `14`     |
| `/`       | Division       | `7 / 2`   | `3`      |
| `%`       | Reste          | `7 % 2`   | `1`      |

La division entre entiers est **entière** et tronque vers zéro :
`7 / 2` vaut `3`, `-7 / 2` vaut `-3`. Le reste a le signe du dividende :
`-7 % 2` vaut `-1`. Pour une division décimale, convertissez d'abord :
`float64(7) / 2` vaut `3.5`.

Une division entière par zéro provoque un `panic` à l'exécution, ou une erreur
de compilation si le diviseur est une constante. Une division flottante par
zéro donne `+Inf`, `-Inf` ou `NaN`.

## Opérateurs de comparaison

| Opérateur | Rôle                |
|-----------|---------------------|
| `==`      | Égal à              |
| `!=`      | Différent de        |
| `<`, `<=` | Inférieur (ou égal) |
| `>`, `>=` | Supérieur (ou égal) |

Les deux opérandes doivent être du **même type**. Les chaînes se comparent
octet par octet (ordre lexicographique). Les structures et les tableaux sont
comparables avec `==` si tous leurs champs le sont. Les slices, les maps et les
fonctions ne sont comparables qu'à `nil`.

## Opérateurs logiques

| Opérateur | Rôle |
|-----------|------|
| `&&`      | Et   |
| `\|\|`    | Ou   |
| `!`       | Non  |

`&&` et `||` sont évalués **en court-circuit** : l'opérande de droite n'est
évaluée que si nécessaire. On peut donc écrire sans risque
`if p != nil && p.Actif { … }`.

## Opérateurs de bits

| Opérateur | Rôle                    | Exemple (`a = 0b1100`, `b = 0b1010`) | Résultat |
|-----------|-------------------------|--------------------------|----------|
| `&`       | Et bit à bit            | `a & b`                  | `0b1000` |
| `\|`      | Ou bit à bit            | `a \| b`                 | `0b1110` |
| `^`       | Ou exclusif (binaire), complément (unaire) | `a ^ b` | `0b0110` |
| `&^`      | Et non (*bit clear*)    | `a &^ b`                 | `0b0100` |
| `<<`      | Décalage à gauche       | `1 << 4`                 | `16`     |
| `>>`      | Décalage à droite       | `16 >> 2`                | `4`      |

`&^` est propre à Go : `a &^ b` met à zéro dans `a` les bits qui sont à 1 dans
`b`. C'est l'opérateur idéal pour retirer un drapeau :

```go
const (
	Lecture  = 1 << iota // 0b001
	Ecriture             // 0b010
	Execution            // 0b100
)

droits := Lecture | Ecriture | Execution
droits &^= Execution                // retire le droit d'exécution
peutEcrire := droits&Ecriture != 0  // teste un drapeau
```

Les littéraux binaires (`0b1010`), octaux (`0o755`) et le séparateur `_`
(`1_000_000`) existent depuis **Go 1.13**.

## Opérateurs d'affectation

`=` affecte une valeur à une variable existante, `:=` déclare et affecte.
Chaque opérateur binaire a sa forme composée : `+=`, `-=`, `*=`, `/=`, `%=`,
`&=`, `|=`, `^=`, `&^=`, `<<=`, `>>=`.

L'affectation multiple évalue toutes les expressions de droite **avant**
d'affecter. Échanger deux variables ne demande pas de variable temporaire :

```go
a, b = b, a
```

## Incrémentation et décrémentation

`i++` et `i--` sont des **instructions**, et non des expressions : on ne peut pas
écrire `x := i++` ni `f(i++)`. Il n'existe pas de forme préfixe (`++i`).

## Opérateurs d'adresse et de channel

| Opérateur | Rôle                                     |
|-----------|------------------------------------------|
| `&x`      | Adresse de `x` (crée un pointeur)        |
| `*p`      | Valeur pointée par `p`                   |
| `<-ch`    | Reçoit une valeur depuis le channel `ch` |
| `ch <- v` | Envoie `v` dans le channel `ch`          |

Ils sont détaillés dans les chapitres sur les pointeurs et la concurrence.

## Ce qui n'existe pas

- **Pas d'opérateur ternaire** (`c ? a : b`) : on écrit un `if`. Pour choisir
  le minimum ou le maximum, `min` et `max` sont intégrés depuis Go 1.21, et
  `cmp.Or` (Go 1.22) renvoie la première valeur non nulle :
  `port := cmp.Or(os.Getenv("PORT"), "8080")`.
- **Pas de surcharge d'opérateurs** : `+` sur un type défini par l'utilisateur
  n'est possible que si son type sous-jacent le permet.
- **Pas de conversion implicite** : `int32 + int64` ne compile pas.

## Priorité des opérateurs

Il n'y a que cinq niveaux de priorité pour les opérateurs binaires, du plus
fort au plus faible :

| Priorité | Opérateurs                        |
|----------|-----------------------------------|
| 5        | `*` `/` `%` `<<` `>>` `&` `&^`    |
| 4        | `+` `-` `\|` `^`                  |
| 3        | `==` `!=` `<` `<=` `>` `>=`       |
| 2        | `&&`                              |
| 1        | `\|\|`                            |

À priorité égale, l'évaluation se fait de gauche à droite. Contrairement au C,
`&` est plus prioritaire que `==` : `x&masque != 0` se lit
`(x&masque) != 0`. Dans le doute, ajoutez des parenthèses ; `gofmt` ajuste les
espaces pour refléter la priorité.

## Exercices

Les solutions sont dans
[`examples/exercices/langage/operateurs`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/operateurs),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Pair ou impair ★

```go
func EstPair(n int) bool
```

Attention aux nombres négatifs : en Go, `-3 % 2` vaut `-1`.

### Des droits en bits ★★

Un ensemble de droits tient dans un entier, un bit par droit :

```go
type Droit uint8

const (
	Lecture Droit = 1 << iota
	Ecriture
	Execution
)

func A(d, demandes Droit) bool      // d contient-il tous les droits demandés ?
func Retire(d, retires Droit) Droit // d, privé des droits retirés
```

`A(Lecture|Ecriture, Lecture)` est vrai, `A(Lecture, Lecture|Execution)` est
faux. Indice : l'opérateur `&^` (ET NON) efface des bits.

### Compter les bits ★★★

Comptez les bits à 1 d'un entier, sans le package `math/bits` :

```go
func Bits(n uint64) int
```

`Bits(0b1011)` vaut 3. Indice : `n & (n - 1)` efface le bit à 1 le plus faible
de `n`. Le test compare votre résultat à celui de `bits.OnesCount64`.
