---
title: "Structures et méthodes"
linkTitle: "6. Structures et méthodes"
weight: 60
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/structures`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/structures).

## Définir une structure

Une structure (`struct`) regroupe des champs nommés. C'est la brique de base
pour modéliser des données en Go, qui n'a pas de classes.

```go
type Personne struct {
	Nom     string
	Age     int
	Email   string
	actif   bool // non exporté
	Inscrit time.Time
}
```

La règle de visibilité s'applique aux champs : `Nom` est accessible depuis les
autres packages, `actif` ne l'est pas. C'est le seul mécanisme
d'encapsulation de Go, et il opère au niveau du **package**, pas du type.

## Créer des valeurs

```go
alice := Personne{Nom: "Alice", Age: 31}        // champs nommés
bob := Personne{"Bob", 27, "bob@example.com", true, time.Time{}} // tous, dans l'ordre
var vide Personne                               // valeur zéro

alice.Age++
fmt.Println(alice.Nom, alice.Age, bob.Email, vide.Nom == "")
// Alice 32 bob@example.com true
```

Préférez **toujours** la forme avec les noms de champs : elle reste valide si
la structure gagne un champ, et les champs omis prennent leur valeur zéro. `go
vet` signale la forme positionnelle pour les types d'autres packages.

`&Personne{…}` crée la valeur et renvoie un pointeur vers elle. On accède aux
champs à travers un pointeur avec la même notation : `p.Nom` équivaut à
`(*p).Nom`.

Deux structures sont comparables avec `==` si tous leurs champs le sont. Le
verbe `%+v` affiche les noms des champs, ce qui est pratique pour déboguer :

```go
type Point struct{ X, Y int }
a := Point{1, 2}
fmt.Println(a == Point{1, 2}, a)
// true {1 2}
fmt.Printf("%+v\n", a)
// {X:1 Y:2}
```

Une **structure anonyme** convient pour une donnée locale à usage unique,
comme les cas d'un test ou un document JSON ponctuel :

```go
config := struct {
	Hote string
	Port int
}{"localhost", 8080}
```

## Les constructeurs

Go n'a pas de constructeur au sens des langages objets. Par convention, une
fonction `NewXxx` (ou `NouveauXxx`) crée une valeur valide, et peut renvoyer
une erreur si les paramètres ne le permettent pas :

```go
func NouvellePersonne(nom string, age int) (*Personne, error) {
	if strings.TrimSpace(nom) == "" {
		return nil, fmt.Errorf("nom vide")
	}
	return &Personne{Nom: nom, Age: age, actif: true}, nil
}
```

Lorsque la valeur zéro du type est directement utilisable, aucun constructeur
n'est nécessaire. C'est l'idéal à viser.

## Les méthodes

Une méthode est une fonction dotée d'un **récepteur**. On peut définir des
méthodes sur n'importe quel type nommé déclaré dans le même package, pas
seulement sur les structures.

```go
type Compte struct {
	Titulaire string
	solde     float64
}

// Solde a un récepteur valeur : elle lit le compte sans le modifier.
func (c Compte) Solde() float64 { return c.solde }

// Depose a un récepteur pointeur : elle modifie le compte.
func (c *Compte) Depose(montant float64) { c.solde += montant }

// DeposeRate a un récepteur valeur : elle modifie une copie.
func (c Compte) DeposeRate(montant float64) { c.solde += montant }

c := Compte{Titulaire: "Alice"}
c.Depose(100) // Go écrit (&c).Depose(100)
c.DeposeRate(50)
fmt.Println(c.Solde())
// 100
```

### Récepteur valeur ou pointeur ?

Le récepteur est un paramètre comme un autre : un récepteur valeur reçoit une
**copie**. Utilisez un récepteur pointeur lorsque :

- la méthode **modifie** le récepteur ;
- la structure est **grande**, et la copier serait coûteux ;
- la structure contient un champ qui ne doit pas être copié, comme un
  `sync.Mutex`.

Par cohérence, si une méthode du type a un récepteur pointeur, donnez-en un à
toutes. Le choix a une conséquence sur les interfaces : les méthodes à
récepteur pointeur n'appartiennent qu'au type pointeur `*T` (voir le chapitre
sur les interfaces).

## La composition par intégration

Go n'a pas d'héritage. On compose des types en **intégrant** (*embedding*) un
type dans une structure, en le déclarant sans nom de champ. Les champs et les
méthodes du type intégré sont alors **promus** : accessibles directement depuis
la structure englobante.

```go
type Animal struct{ Nom string }

func (a Animal) Presente() string { return "Je suis " + a.Nom }

type Chien struct {
	Animal // champ intégré
	Race   string
}

func (c Chien) Aboie() string { return c.Nom + " : Wouf !" }

rex := Chien{Animal: Animal{Nom: "Rex"}, Race: "berger"}
fmt.Println(rex.Presente()) // méthode promue
fmt.Println(rex.Aboie())
// Je suis Rex
// Rex : Wouf !
```

Il ne s'agit pas d'héritage : un `Chien` **n'est pas** un `Animal`, on ne peut
pas le passer à une fonction qui attend un `Animal`. Le champ intégré reste
accessible par le nom de son type (`rex.Animal`). Si `Chien` définit lui aussi
une méthode `Presente`, elle masque celle d'`Animal`, qui reste appelable avec
`rex.Animal.Presente()`.

> [!NOTE]
> **Go 1.27** — un champ promu peut servir de clé dans un littéral :
> `Chien{Nom: "Rex", Race: "berger"}`. Auparavant, il fallait écrire
> `Chien{Animal: Animal{Nom: "Rex"}, …}`.

L'intégration fonctionne aussi avec des interfaces, et c'est un moyen courant
de construire un type qui « décore » un autre type en ne redéfinissant que
certaines méthodes.

## Les balises de champs

Une **balise** (*struct tag*) est une chaîne attachée à un champ, lisible par
réflexion. Les packages d'encodage s'en servent pour connaître le nom à
utiliser et les options :

```go
type Article struct {
	ID    int      `json:"id"`
	Titre string   `json:"titre"`
	Tags  []string `json:"tags,omitempty"`
}
```

Les balises sont utilisées par `encoding/json`, `encoding/xml`, les bibliothèques
d'accès aux bases de données ou de validation. `go vet` vérifie leur syntaxe.

## La structure vide

`struct{}` n'a aucun champ et occupe **zéro octet**. On l'utilise comme valeur
d'un ensemble (`map[string]struct{}`) ou comme signal dans un channel
(`chan struct{}`), lorsque seule l'existence de l'événement compte.

## Exercices

Les solutions sont dans
[`examples/exercices/langage/structures`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/langage/structures),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Un rectangle ★

```go
type Rectangle struct {
	Largeur, Hauteur float64
}

func (r Rectangle) Aire() float64
func (r Rectangle) Perimetre() float64
```

### Une pile ★★

```go
type Pile struct{ /* à vous de choisir */ }

func (p *Pile) Empile(v int)
func (p *Pile) Depile() (int, bool) // faux si la pile est vide
func (p *Pile) Taille() int
```

La valeur zéro, `var p Pile`, doit être une pile vide utilisable. Pourquoi
ces méthodes ont-elles un récepteur pointeur ?

### Un compte avec historique ★★★

Écrivez un type `Historique` avec deux méthodes, `Note(format string, args
...any)` et `Lignes() []string`. Puis un type `Compte` qui **intègre**
`Historique` :

```go
var ErrSoldeInsuffisant = errors.New("solde insuffisant")

func (c *Compte) Depose(montant int)
func (c *Compte) Retire(montant int) error
func (c *Compte) Solde() int
```

Chaque opération est notée (« dépôt de 100 », « retrait de 30 », « retrait de
500 refusé »), et le test lit l'historique par `c.Lignes()`, une méthode
promue depuis `Historique`.
