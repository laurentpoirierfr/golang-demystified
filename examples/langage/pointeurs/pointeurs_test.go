// Exemples du chapitre « Pointeurs et mémoire ».
package pointeurs

import (
	"fmt"
	"time"
)

func Example_pointeur() {
	x := 10
	p := &x // p est de type *int
	*p = 20
	fmt.Println(x, *p, p == &x)
	// Output: 20 20 true
}

func doubleValeur(n int)    { n *= 2 }
func doublePointeur(n *int) { *n *= 2 }

func Example_passage() {
	a := 5
	doubleValeur(a)
	fmt.Println(a)
	doublePointeur(&a)
	fmt.Println(a)
	// Output:
	// 5
	// 10
}

func Example_new() {
	p := new(int) // pointeur vers un int initialisé à 0
	*p = 3
	q := new(42) // Go 1.26 : pointeur vers un int initialisé à 42
	fmt.Println(*p, *q)
	// Output: 3 42
}

type Utilisateur struct {
	Nom        string
	Age        *int       // optionnel
	Expiration *time.Time // optionnel
}

func anneesDepuis(t time.Time) int { return 2026 - t.Year() }

func Example_champOptionnel() {
	naissance := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	u := Utilisateur{
		Nom: "Alice",
		Age: new(anneesDepuis(naissance)),
	}
	fmt.Println(u.Nom, *u.Age, u.Expiration == nil)
	// Output: Alice 36 true
}

type Noeud struct {
	Valeur  int
	Suivant *Noeud
}

func Example_listeChainee() {
	var tete *Noeud
	for i := range 3 {
		tete = &Noeud{Valeur: i, Suivant: tete}
	}
	for n := tete; n != nil; n = n.Suivant {
		fmt.Print(n.Valeur)
	}
	fmt.Println()
	// Output: 210
}

func nouveauCompteur() *int {
	n := 0
	return &n // n « s'échappe » : il est alloué sur le tas
}

func Example_echappement() {
	c := nouveauCompteur()
	*c++
	fmt.Println(*c)
	// Output: 1
}
