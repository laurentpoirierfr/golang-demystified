// Exemples du chapitre « Fonctions ».
package fonctions

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
)

func divise(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division par zéro")
	}
	return a / b, nil
}

func Example_retoursMultiples() {
	q, err := divise(7, 2)
	fmt.Println(q, err)
	_, err = divise(1, 0)
	fmt.Println(err)
	// Output:
	// 3 <nil>
	// division par zéro
}

func statistiques(nombres []float64) (moyenne, ecart float64) {
	if len(nombres) == 0 {
		return // renvoie les valeurs zéro
	}
	var somme float64
	for _, n := range nombres {
		somme += n
	}
	moyenne = somme / float64(len(nombres))
	ecart = slices.Max(nombres) - slices.Min(nombres)
	return moyenne, ecart
}

func Example_retoursNommes() {
	fmt.Println(statistiques([]float64{2, 4, 9}))
	fmt.Println(statistiques(nil))
	// Output:
	// 5 7
	// 0 0
}

func somme(nombres ...int) int {
	total := 0
	for _, n := range nombres {
		total += n
	}
	return total
}

func Example_variadique() {
	fmt.Println(somme(), somme(1, 2, 3))
	valeurs := []int{4, 5, 6}
	fmt.Println(somme(valeurs...))
	// Output:
	// 0 6
	// 15
}

func applique(s []string, f func(string) string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		out = append(out, f(x))
	}
	return out
}

func Example_fonctionsValeurs() {
	crier := func(s string) string { return strings.ToUpper(s) + " !" }
	fmt.Println(applique([]string{"go", "zig"}, crier))
	fmt.Println(applique([]string{"Go"}, strings.ToLower))
	// Output:
	// [GO ! ZIG !]
	// [go]
}

func compteur() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

func Example_closure() {
	c1 := compteur()
	c2 := compteur()
	fmt.Println(c1(), c1(), c1(), c2())
	// Output: 1 2 3 1
}

// Pairs renvoie un itérateur sur les nombres pairs de 0 à max inclus.
func Pairs(max int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i <= max; i += 2 {
			if !yield(i) {
				return
			}
		}
	}
}

func Example_iterateur() {
	for n := range Pairs(10) {
		if n > 6 {
			break
		}
		fmt.Print(n, ";")
	}
	fmt.Println()
	fmt.Println(slices.Collect(Pairs(8)))
	// Output:
	// 0;2;4;6;
	// [0 2 4 6 8]
}

// Indexe renvoie un itérateur à deux valeurs : la position et le mot.
func Indexe(texte string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, mot := range strings.Fields(texte) {
			if !yield(i+1, mot) {
				return
			}
		}
	}
}

func Example_iterateurSeq2() {
	for pos, mot := range Indexe("le gopher creuse") {
		fmt.Println(pos, mot)
	}
	// Output:
	// 1 le
	// 2 gopher
	// 3 creuse
}

func Example_iterateursStandard() {
	for part := range strings.SplitSeq("a,b,c", ",") {
		fmt.Print(part)
	}
	fmt.Println()
	for i, v := range slices.Backward([]string{"x", "y", "z"}) {
		fmt.Print(i, v, " ")
	}
	fmt.Println("!")
	// Output:
	// abc
	// 2z 1y 0x !
}

func Example_recover() {
	fmt.Println(divisionSure(10, 2))
	fmt.Println(divisionSure(1, 0))
	// Output:
	// 5 <nil>
	// 0 panic récupéré : runtime error: integer divide by zero
}

func divisionSure(a, b int) (resultat int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic récupéré : %v", r)
		}
	}()
	return a / b, nil
}
