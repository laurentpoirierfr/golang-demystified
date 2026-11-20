---
title: "Variables, constantes et types"
linkTitle: "1. Variables, constantes et types"
weight: 10
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/bases`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/bases).

## Déclarer des variables

Go est un langage **statiquement typé** : chaque variable a un type fixé à la
compilation. Le compilateur sait cependant souvent le déduire.

```go
var nom string = "Gopher"
var age = 15 // type déduit : int
ville := "Paris"
var x, y int = 1, 2
a, b := "un", 2

fmt.Println(nom, age, ville, x, y, a, b)
// Gopher 15 Paris 1 2 un 2
```

- **`var nom type = valeur`** : la forme complète, utilisable partout, y compris
  hors des fonctions (variables de package).
- **`nom := valeur`** : la *déclaration courte*, réservée à l'intérieur des
  fonctions. C'est la forme la plus courante.

Plusieurs déclarations peuvent être regroupées :

```go
var (
	hote    = "localhost"
	port    = 8080
	verbeux bool
)
```

> [!WARNING]
> Une variable locale déclarée mais jamais utilisée est une **erreur de
> compilation**. L'identifiant `_` (*blank identifier*) permet d'ignorer
> explicitement une valeur : `_, err := f()`.

## La valeur zéro

En Go, une variable n'est **jamais** non initialisée. Sans valeur explicite,
elle reçoit la *valeur zéro* de son type :

```go
var (
	i int
	f float64
	b bool
	s string
	p *int
)
fmt.Printf("%d %v %t %q %v\n", i, f, b, s, p)
// 0 0 false "" <nil>
```

| Type                                          | Valeur zéro |
|-----------------------------------------------|-------------|
| nombres                                       | `0`         |
| `bool`                                        | `false`     |
| `string`                                      | `""`        |
| pointeurs, slices, maps, channels, fonctions, interfaces | `nil` |
| structures, tableaux                          | chaque champ ou élément à sa valeur zéro |

Les bibliothèques Go sont conçues pour que la valeur zéro soit **utilisable** :
un `sync.Mutex`, un `bytes.Buffer` ou un `strings.Builder` fonctionnent sans
initialisation.

## Les types de base

| Catégorie        | Types                                                    |
|------------------|----------------------------------------------------------|
| Booléen          | `bool`                                                   |
| Entiers signés   | `int`, `int8`, `int16`, `int32`, `int64`                 |
| Entiers non signés | `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr` |
| Flottants        | `float32`, `float64`                                     |
| Complexes        | `complex64`, `complex128`                                |
| Texte            | `string`, `byte` (alias de `uint8`), `rune` (alias de `int32`) |

`int` et `uint` font 64 bits sur les plateformes 64 bits. Utilisez `int` par
défaut, et les tailles explicites seulement lorsqu'un format l'impose (protocole
binaire, fichier…).

Les débordements d'entiers ne sont pas des erreurs : la valeur boucle.

```go
var b uint8 = 255
b++
fmt.Println(b, math.MaxInt64)
// 0 9223372036854775807
```

## Les conversions

Go ne fait **aucune conversion implicite**, même entre `int` et `int64`. Une
conversion s'écrit `T(valeur)` :

```go
i := 42
f := float64(i) * 1.5
u := uint8(300 % 256)
```

Les conversions entre texte et nombres passent par le package `strconv`, et
peuvent échouer :

```go
s := strconv.Itoa(i)             // "42"
n, err := strconv.Atoi("123")    // 123, nil
_, err = strconv.Atoi("douze")
fmt.Println(err)
// strconv.Atoi: parsing "douze": invalid syntax
```

> [!CAUTION]
> `string(65)` ne donne pas `"65"` mais `"A"` : la conversion d'un entier en
> chaîne interprète l'entier comme un point de code Unicode. `go vet` signale
> cette erreur fréquente.

## Les chaînes de caractères

Une `string` est une suite d'octets **immuable**, encodée par convention en
UTF-8. `len` renvoie le nombre d'**octets**, et non de caractères :

```go
s := "Gopher à Paris"
fmt.Println(len(s), utf8.RuneCountInString(s))
// 15 14
fmt.Println(s[0], string(s[0]), s[:6])
// 71 G Gopher
```

Un caractère Unicode est une `rune`. Parcourir une chaîne avec `range` la
décode en runes, et l'indice donne la position en octets :

```go
for i, r := range "çà" {
	fmt.Println(i, string(r), r)
}
// 0 ç 231
// 2 à 224
```

Les chaînes entre *backquotes* sont **brutes** : pas d'échappement, et elles
peuvent s'étendre sur plusieurs lignes. Elles sont pratiques pour les chemins
Windows, les expressions régulières ou les gabarits :

```go
chemin := `C:\Users\gopher`
requete := `
	SELECT nom, age
	FROM utilisateurs
	WHERE age > $1`
```

Pour construire une chaîne par morceaux, utilisez `strings.Builder`, qui évite
de recopier la chaîne à chaque concaténation (voir le chapitre sur `strings`).

## Les constantes

Une constante est évaluée à la **compilation**. Elle peut être un nombre, une
chaîne, un booléen ou une rune :

```go
const Pi = 3.14159
const (
	NomApplication = "golang-demystified"
	Version        = 2
)
```

Une constante sans type explicite est **non typée** : elle a une précision
arbitraire et prend le type requis par son utilisation. Elle peut donc
dépasser la capacité de tous les types numériques, du moment que le résultat
final tient dans le type cible :

```go
const grand = 1 << 100 // n'existe qu'à la compilation
const petit = grand >> 98
var f float64 = grand / 1e29
fmt.Println(petit, f)
// 4 12.676506002282293
```

### Les énumérations avec iota

Go n'a pas de type énuméré. On utilise un type nommé et des constantes
numérotées avec `iota`, qui vaut 0 dans la première ligne d'un bloc `const` et
s'incrémente à chaque ligne. Une ligne sans expression répète la précédente :

```go
type JourSemaine int

const (
	Lundi JourSemaine = iota
	Mardi
	Mercredi
	Jeudi
	Vendredi
)

type Taille int64

const (
	_         = iota // on ignore la valeur 0
	Ko Taille = 1 << (10 * iota)
	Mo
	Go
)

fmt.Println(Lundi, Mercredi, Vendredi)
fmt.Println(Ko, Mo, Go)
// 0 2 4
// 1024 1048576 1073741824
```

Pour afficher le nom d'une valeur plutôt que son numéro, ajoutez une méthode
`String()`, que l'outil [`stringer`](https://pkg.go.dev/golang.org/x/tools/cmd/stringer) peut générer :

```bash
go get -tool golang.org/x/tools/cmd/stringer
go tool stringer -type=JourSemaine
```

## Types nommés et alias

`type` crée un **nouveau type**, distinct de son type sous-jacent. On ne peut
pas mélanger les deux sans conversion, ce qui évite de confondre des valeurs de
même représentation :

```go
type Celsius float64
type Fahrenheit float64

func CVersF(c Celsius) Fahrenheit { return Fahrenheit(c*9/5 + 32) }
```

Un **alias** (`=`) donne simplement un autre nom au même type. Il sert surtout
à déplacer un type d'un package à un autre sans casser le code existant :

```go
type Octet = byte // Octet et byte sont interchangeables
```

> [!NOTE]
> **Go 1.24** — un alias peut avoir des paramètres de type :
> `type Ensemble[T comparable] = map[T]struct{}`.

## La portée

Une variable est visible dans le **bloc** qui la déclare : une fonction, un
`if`, un `for`, un `case`… Une déclaration dans un bloc interne peut **masquer**
une variable externe du même nom. C'est une source classique de bugs avec
`:=` :

```go
var err error
if condition {
	resultat, err := calcul() // nouveau err, qui masque l'externe !
	_ = resultat
}
// ici, err vaut toujours nil
```

Dans ce cas, déclarez `resultat` avant le bloc et utilisez `=`.

## Exercices

Les solutions sont dans
[`examples/exercices/langage/bases`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/bases),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Celsius et Fahrenheit ★

Écrivez la conversion d'une température, sachant que F = C × 9/5 + 32 :

```go
func Fahrenheit(celsius float64) float64
```

`Fahrenheit(100)` vaut 212, et `Fahrenheit(-40)` vaut -40.

### Des unités avec iota ★★

Déclarez, avec `iota`, les constantes `Octet`, `Kio`, `Mio`, `Gio` et `Tio`,
chacune 1 024 fois plus grande que la précédente (indice : `1 << (10 * iota)`).
Puis écrivez une fonction qui affiche une taille avec la plus grande unité qui
convient et une décimale :

```go
func FormateTaille(n int64) string
```

`FormateTaille(512)` renvoie `"512 o"`, `FormateTaille(1536)` renvoie `"1.5 Kio"`.

### Détecter un dépassement ★★★

Un `int8` contient les entiers de -128 à 127 ; au-delà, le calcul « fait le
tour » sans erreur. Écrivez une addition qui signale le dépassement :

```go
func AdditionneInt8(a, b int8) (int8, bool)
```

`AdditionneInt8(100, 28)` renvoie `-128, true`. Indice : faites le calcul dans
un type plus grand, puis convertissez.
