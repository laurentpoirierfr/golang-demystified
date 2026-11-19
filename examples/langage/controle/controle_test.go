// Exemples du chapitre « Contrôle de flux ».
package controle

import (
	"fmt"
	"os"
	"strings"
)

func Example_if() {
	temperature := 23
	if temperature > 30 {
		fmt.Println("chaud")
	} else if temperature > 15 {
		fmt.Println("doux")
	} else {
		fmt.Println("froid")
	}

	// Une instruction d'initialisation, dont la portée est limitée au if.
	if _, err := os.Stat("/fichier/inexistant"); err != nil {
		fmt.Println("introuvable")
	}
	// Output:
	// doux
	// introuvable
}

func Example_for() {
	// La forme classique, en trois clauses.
	for i := 0; i < 3; i++ {
		fmt.Print(i)
	}
	fmt.Println()

	// La forme « while ».
	n := 1
	for n < 100 {
		n *= 3
	}
	fmt.Println(n)

	// La boucle infinie, interrompue par break.
	compteur := 0
	for {
		compteur++
		if compteur == 5 {
			break
		}
	}
	fmt.Println(compteur)
	// Output:
	// 012
	// 243
	// 5
}

func Example_rangeEntier() {
	for i := range 3 {
		fmt.Print(i)
	}
	fmt.Println()
	// Output: 012
}

func Example_range() {
	langages := []string{"Go", "Rust", "Zig"}
	for i, l := range langages {
		fmt.Println(i, l)
	}
	for _, l := range langages {
		fmt.Print(strings.ToUpper(l))
	}
	fmt.Println()
	for i := range langages {
		fmt.Print(i)
	}
	fmt.Println()
	// Output:
	// 0 Go
	// 1 Rust
	// 2 Zig
	// GORUSTZIG
	// 012
}

func Example_variableDeBoucle() {
	var fonctions []func()
	for i := range 3 {
		fonctions = append(fonctions, func() { fmt.Print(i) })
	}
	for _, f := range fonctions {
		f()
	}
	fmt.Println()
	// Output: 012
}

func Example_labels() {
	grille := [][]int{{1, 2}, {3, -1}, {5, 6}}
recherche:
	for i, ligne := range grille {
		for j, v := range ligne {
			if v < 0 {
				fmt.Println("négatif en", i, j)
				break recherche
			}
		}
	}
	// Output: négatif en 1 1
}

func jour(n int) string {
	switch n {
	case 1, 2, 3, 4, 5:
		return "semaine"
	case 6, 7:
		return "week-end"
	default:
		return "inconnu"
	}
}

func Example_switch() {
	fmt.Println(jour(3), jour(7), jour(9))
	// Output: semaine week-end inconnu
}

func mention(note int) string {
	switch {
	case note >= 16:
		return "très bien"
	case note >= 14:
		return "bien"
	case note >= 10:
		return "passable"
	default:
		return "ajourné"
	}
}

func Example_switchSansExpression() {
	fmt.Println(mention(17), mention(12), mention(4))
	// Output: très bien passable ajourné
}

func Example_fallthrough() {
	switch n := 1; n {
	case 1:
		fmt.Println("un")
		fallthrough
	case 2:
		fmt.Println("deux")
	case 3:
		fmt.Println("trois")
	}
	// Output:
	// un
	// deux
}

func Example_defer() {
	fmt.Println("début")
	for i := range 3 {
		defer fmt.Println("defer", i)
	}
	fmt.Println("fin")
	// Output:
	// début
	// fin
	// defer 2
	// defer 1
	// defer 0
}
