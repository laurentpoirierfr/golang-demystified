// Exemples du chapitre « Génériques ».
package generiques

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// Map applique f à chaque élément de s.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

func Example_fonctionGenerique() {
	longueurs := Map([]string{"go", "rust", "zig"}, func(s string) int { return len(s) })
	textes := Map([]int{1, 2, 3}, strconv.Itoa)
	fmt.Println(longueurs, strings.Join(textes, "-"))
	// Output: [2 4 3] 1-2-3
}

// Plus grand renvoie le plus grand élément d'une slice non vide.
func PlusGrand[T cmp.Ordered](s []T) T {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func Example_contrainte() {
	fmt.Println(PlusGrand([]int{3, 9, 2}), PlusGrand([]string{"b", "c", "a"}))
	// Output: 9 c
}

// Nombre accepte les types dont le type sous-jacent est un entier ou un flottant.
type Nombre interface {
	~int | ~int64 | ~float64
}

func Somme[T Nombre](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

type Euros float64

func Example_ensembleDeTypes() {
	fmt.Println(Somme([]int{1, 2, 3}))
	fmt.Println(Somme([]Euros{9.99, 0.01}))
	// Output:
	// 6
	// 10
}

// Pile est un type générique.
type Pile[T any] struct {
	elements []T
}

func (p *Pile[T]) Empile(v T) { p.elements = append(p.elements, v) }

func (p *Pile[T]) Depile() (T, bool) {
	var zero T
	if len(p.elements) == 0 {
		return zero, false
	}
	v := p.elements[len(p.elements)-1]
	p.elements = p.elements[:len(p.elements)-1]
	return v, true
}

func Example_typeGenerique() {
	var p Pile[string]
	p.Empile("a")
	p.Empile("b")
	v, ok := p.Depile()
	fmt.Println(v, ok)
	v, _ = p.Depile()
	_, ok = p.Depile()
	fmt.Println(v, ok)
	// Output:
	// b true
	// a false
}

// Ensemble est un alias générique (Go 1.24).
type Ensemble[T comparable] = map[T]struct{}

func Example_aliasGenerique() {
	e := Ensemble[string]{"go": {}}
	_, ok := e["go"]
	fmt.Println(ok)
	// Output: true
}

// Liste illustre les méthodes génériques (Go 1.27).
type Liste[T any] struct{ elements []T }

// Transforme est une méthode qui déclare son propre paramètre de type U.
func (l Liste[T]) Transforme[U any](f func(T) U) Liste[U] {
	return Liste[U]{elements: Map(l.elements, f)}
}

func Example_methodeGenerique() {
	l := Liste[int]{elements: []int{1, 2, 3}}
	carres := l.Transforme(func(n int) string { return strconv.Itoa(n * n) })
	fmt.Println(carres.elements)
	// Output: [1 4 9]
}

// Vecteur est contraint par une interface qui se référence elle-même (Go 1.26).
type Additionnable[A Additionnable[A]] interface {
	Ajoute(A) A
}

type Vecteur struct{ X, Y int }

func (v Vecteur) Ajoute(w Vecteur) Vecteur { return Vecteur{v.X + w.X, v.Y + w.Y} }

func Total[A Additionnable[A]](valeurs ...A) A {
	var total A
	for _, v := range valeurs {
		total = total.Ajoute(v)
	}
	return total
}

func Example_contrainteRecursive() {
	fmt.Println(Total(Vecteur{1, 2}, Vecteur{3, 4}, Vecteur{5, 6}))
	// Output: {9 12}
}
