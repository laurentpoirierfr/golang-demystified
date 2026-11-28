---
title: "II.7 Interfaces"
description: "des contrats implicites et petits, le type concret retrouvé, et le piège de l'interface nil."
weight: 70
---

# Chapitre II.7 — Interfaces

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/interfaces`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/interfaces).

## Un contrat de comportement

Une interface décrit un **ensemble de méthodes**. Tout type qui possède ces
méthodes satisfait l'interface, **implicitement** : il n'y a pas de mot-clé
`implements`.

```go
type Forme interface {
	Aire() float64
	Perimetre() float64
}

type Rectangle struct{ Largeur, Hauteur float64 }

func (r Rectangle) Aire() float64      { return r.Largeur * r.Hauteur }
func (r Rectangle) Perimetre() float64 { return 2 * (r.Largeur + r.Hauteur) }

type Cercle struct{ Rayon float64 }

func (c Cercle) Aire() float64      { return math.Pi * c.Rayon * c.Rayon }
func (c Cercle) Perimetre() float64 { return 2 * math.Pi * c.Rayon }

formes := []Forme{Rectangle{3, 4}, Cercle{1}}
total := 0.0
for _, f := range formes {
	total += f.Aire()
}
fmt.Printf("%.2f\n", total)
// 15.14
```

Cette satisfaction implicite a une conséquence importante : le package qui
**utilise** une interface peut la définir, sans que les types qui
l'implémentent la connaissent. On découple ainsi les packages sans créer de
dépendance.

Pour vérifier à la compilation qu'un type satisfait une interface, on utilise
une affectation à l'identifiant `_` :

```go
var _ Forme = Rectangle{}
```

## Les petites interfaces

Les interfaces idiomatiques de Go sont **petites**, souvent réduites à une
seule méthode, et nommées d'après cette méthode avec le suffixe *-er* :

| Interface      | Package | Méthode                                  |
|----------------|---------|------------------------------------------|
| `io.Reader`    | `io`    | `Read(p []byte) (n int, err error)`      |
| `io.Writer`    | `io`    | `Write(p []byte) (n int, err error)`     |
| `io.Closer`    | `io`    | `Close() error`                          |
| `fmt.Stringer` | `fmt`   | `String() string`                        |
| `error`        | intégré | `Error() string`                         |
| `sort.Interface` | `sort` | `Len`, `Less`, `Swap`                   |
| `http.Handler` | `net/http` | `ServeHTTP(ResponseWriter, *Request)` |

Plus une interface est petite, plus elle a d'implémentations. Une fonction qui
accepte un `io.Reader` peut lire un fichier, une connexion réseau, une
requête HTTP, une archive compressée ou une simple chaîne :

```go
func compteMots(r io.Reader) (int, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	return len(strings.Fields(string(data))), nil
}

n, _ := compteMots(strings.NewReader("le gopher creuse un terrier"))
// 5
```

> « The bigger the interface, the weaker the abstraction. » — Rob Pike

Un type qui implémente `fmt.Stringer` contrôle son affichage par les fonctions
de `fmt` :

```go
type Temperature float64

func (t Temperature) String() string { return fmt.Sprintf("%.1f °C", float64(t)) }

fmt.Println(Temperature(21.456))
// 21.5 °C
```

Les interfaces se composent par intégration :

```go
type ReadWriter interface {
	io.Reader
	io.Writer
}
```

## Accepter des interfaces, renvoyer des structures

Ce principe résume l'usage des interfaces en Go :

- les **paramètres** d'une fonction sont des interfaces, aussi petites que
  possible, pour accepter un maximum de types ;
- les **valeurs de retour** sont des types concrets, pour que l'appelant ait
  accès à toutes leurs méthodes.

Évitez de définir une interface « au cas où », à côté de son unique
implémentation. Définissez-la là où elle est utilisée, au moment où un second
cas d'usage apparaît, par exemple un double de test.

## any : l'interface vide

L'interface sans méthode, `interface{}`, est satisfaite par tous les types.
Depuis **Go 1.18**, elle a un alias plus lisible : `any`. Une valeur `any` peut
contenir n'importe quoi, mais on ne peut rien en faire sans retrouver son type
concret. À utiliser avec parcimonie : les génériques sont souvent une meilleure
solution.

## Retrouver le type concret

### L'assertion de type

`v.(T)` extrait la valeur concrète de type `T` contenue dans l'interface `v`.
La forme à une valeur provoque un `panic` si le type ne correspond pas. La
forme à deux valeurs ne panique jamais :

```go
var f Forme = Cercle{2}

c := f.(Cercle) // panic si f n'est pas un Cercle
fmt.Println(c.Rayon) // 2

if r, ok := f.(Rectangle); ok {
	fmt.Println("rectangle", r)
} else {
	fmt.Println("pas un rectangle")
}
```

`T` peut aussi être une interface : `v.(io.Closer)` teste si la valeur possède
une méthode `Close`. C'est ainsi que la bibliothèque standard détecte des
capacités optionnelles, comme `io.WriterTo`.

### Le type switch

Un `switch` sur `v.(type)` teste plusieurs types à la suite. Dans chaque
`case` à un seul type, la variable a le type correspondant :

```go
func decris(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case int, int64:
		return fmt.Sprintf("entier %v", x) // x est de type any
	case string:
		return "chaîne de " + fmt.Sprint(len(x)) + " octets"
	case fmt.Stringer:
		return "stringer : " + x.String()
	case error:
		return "erreur : " + x.Error()
	default:
		return fmt.Sprintf("autre (%T)", x)
	}
}
```

## Récepteurs pointeurs et interfaces

L'**ensemble de méthodes** d'un type `T` ne contient que ses méthodes à
récepteur valeur. Celui de `*T` contient toutes ses méthodes. Si une méthode
requise par une interface a un récepteur pointeur, seul `*T` satisfait
l'interface :

```go
type Compteur struct{ n int }

func (c *Compteur) Incremente() { c.n++ }

type Incrementeur interface{ Incremente() }

var i Incrementeur = &Compteur{} // Compteur{} ne compile pas
```

## Le piège de l'interface nil

Une valeur d'interface contient deux éléments : un **type** et une **valeur**.
Elle n'est égale à `nil` que si les deux sont nil. Une interface qui contient
un pointeur nil n'est donc **pas** nil :

```go
type MonErreur struct{}

func (*MonErreur) Error() string { return "mon erreur" }

func peutEchouer(echec bool) error {
	var e *MonErreur // nil
	if echec {
		e = &MonErreur{}
	}
	return e // interface non nil contenant un pointeur nil
}

err := peutEchouer(false)
fmt.Println(err == nil)
// false
```

La règle est simple : une fonction qui renvoie une interface doit renvoyer
`nil` **explicitement**, et non une variable d'un type concret qui vaut nil.

## Décorer un comportement

Une interface permet d'envelopper une implémentation pour lui ajouter un
comportement, sans la modifier. C'est le principe des middlewares HTTP, des
compresseurs ou du chiffrement en flux :

```go
// majuscules décore un io.Writer en mettant le texte en majuscules.
type majuscules struct{ w io.Writer }

func (m majuscules) Write(p []byte) (int, error) {
	return m.w.Write([]byte(strings.ToUpper(string(p))))
}

var w io.Writer = majuscules{os.Stdout}
fmt.Fprintln(w, "bonjour, gopher")
// BONJOUR, GOPHER
```

## Exercices

Les solutions sont dans
[`examples/exercices/langage/interfaces`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/interfaces),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Des formes ★

Déclarez une interface `Forme`, avec une méthode `Aire() float64`, et deux
types qui la satisfont : `Carre{Cote float64}` et `Cercle{Rayon float64}`.
Puis :

```go
func AireTotale(formes ...Forme) float64
```

### Une durée lisible ★★

```go
type Duree int // un nombre de secondes
```

Donnez à `Duree` une méthode `String() string`, pour que `fmt.Println`
affiche `5s`, `1m05s` ou `1h02m05s`. En satisfaisant `fmt.Stringer`, le type
choisit lui-même son affichage.

### Un lecteur en majuscules ★★★

```go
func Majuscules(source io.Reader) io.Reader
```

Le lecteur renvoyé lit `source` et met en majuscules ce qu'il lit :
`io.ReadAll(Majuscules(strings.NewReader("l'été")))` donne `L'ÉTÉ`. C'est un
décorateur : il satisfait `io.Reader`, et se combine donc avec tout ce qui
lit (fichiers, réponses HTTP). Indice : `unicode.ToUpper` et le package
`unicode/utf8`.
