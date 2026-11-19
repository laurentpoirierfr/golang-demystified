// Exemples du chapitre « Structures et méthodes ».
package structures

import (
	"fmt"
	"strings"
	"time"
)

type Personne struct {
	Nom     string
	Age     int
	Email   string
	actif   bool // non exporté
	Inscrit time.Time
}

func Example_litteraux() {
	alice := Personne{Nom: "Alice", Age: 31}
	bob := Personne{"Bob", 27, "bob@example.com", true, time.Time{}}
	var vide Personne

	alice.Age++
	fmt.Println(alice.Nom, alice.Age, bob.Email, vide.Nom == "")
	// Output: Alice 32 bob@example.com true
}

func Example_comparaison() {
	type Point struct{ X, Y int }
	a := Point{1, 2}
	b := Point{1, 2}
	fmt.Println(a == b, a)
	fmt.Printf("%+v\n", a)
	// Output:
	// true {1 2}
	// {X:1 Y:2}
}

func Example_anonyme() {
	config := struct {
		Hote string
		Port int
	}{"localhost", 8080}
	fmt.Println(config.Hote, config.Port)
	// Output: localhost 8080
}

// NouvellePersonne est un constructeur : une simple fonction, par convention
// nommée New… ou Nouveau…
func NouvellePersonne(nom string, age int) (*Personne, error) {
	if strings.TrimSpace(nom) == "" {
		return nil, fmt.Errorf("nom vide")
	}
	return &Personne{Nom: nom, Age: age, actif: true}, nil
}

func Example_constructeur() {
	p, err := NouvellePersonne("Carol", 45)
	fmt.Println(p.Nom, p.actif, err)
	_, err = NouvellePersonne(" ", 1)
	fmt.Println(err)
	// Output:
	// Carol true <nil>
	// nom vide
}

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

func Example_methodes() {
	c := Compte{Titulaire: "Alice"}
	c.Depose(100) // Go écrit (&c).Depose(100)
	c.DeposeRate(50)
	fmt.Println(c.Solde())
	// Output: 100
}

type Animal struct{ Nom string }

func (a Animal) Presente() string { return "Je suis " + a.Nom }

type Chien struct {
	Animal // champ intégré
	Race   string
}

func (c Chien) Aboie() string { return c.Nom + " : Wouf !" }

func Example_integration() {
	rex := Chien{Animal: Animal{Nom: "Rex"}, Race: "berger"}
	fmt.Println(rex.Presente()) // méthode promue
	fmt.Println(rex.Aboie())
	fmt.Println(rex.Animal.Nom == rex.Nom)
	// Output:
	// Je suis Rex
	// Rex : Wouf !
	// true
}

func Example_litteralChampPromu() {
	// Go 1.27 : un champ promu peut servir de clé dans un littéral.
	rex := Chien{Nom: "Rex", Race: "berger"}
	fmt.Println(rex.Presente())
	// Output: Je suis Rex
}

type Article struct {
	ID    int      `json:"id"`
	Titre string   `json:"titre"`
	Tags  []string `json:"tags,omitempty"`
}

func Example_tags() {
	type T = Article
	fmt.Printf("%+v\n", T{ID: 1, Titre: "Go"})
	// Output: {ID:1 Titre:Go Tags:[]}
}
