// Exemples du chapitre « Variables, constantes et types ».
package bases

import (
	"fmt"
	"math"
	"strconv"
	"unicode/utf8"
)

func Example_variables() {
	var nom string = "Gopher"
	var age = 15 // type déduit : int
	ville := "Paris"
	var x, y int = 1, 2
	a, b := "un", 2

	fmt.Println(nom, age, ville, x, y, a, b)
	// Output: Gopher 15 Paris 1 2 un 2
}

func Example_valeursZero() {
	var (
		i int
		f float64
		b bool
		s string
		p *int
	)
	fmt.Printf("%d %v %t %q %v\n", i, f, b, s, p)
	// Output: 0 0 false "" <nil>
}

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
	_         = iota
	Ko Taille = 1 << (10 * iota)
	Mo
	Go
)

func Example_iota() {
	fmt.Println(Lundi, Mercredi, Vendredi)
	fmt.Println(Ko, Mo, Go)
	// Output:
	// 0 2 4
	// 1024 1048576 1073741824
}

func Example_constantesNonTypees() {
	const grand = 1 << 100 // n'existe qu'à la compilation
	const petit = grand >> 98
	var f float64 = grand / 1e29
	fmt.Println(petit, f)
	// Output: 4 12.676506002282293
}

func Example_conversions() {
	i := 42
	f := float64(i) * 1.5
	u := uint8(300 % 256)
	s := strconv.Itoa(i)
	n, err := strconv.Atoi("123")
	fmt.Println(f, u, s+"!", n, err)

	_, err = strconv.Atoi("douze")
	fmt.Println(err)
	// Output:
	// 63 44 42! 123 <nil>
	// strconv.Atoi: parsing "douze": invalid syntax
}

func Example_debordement() {
	var b uint8 = 255
	b++
	fmt.Println(b, math.MaxInt64)
	// Output: 0 9223372036854775807
}

func Example_chaines() {
	s := "Gopher à Paris"
	fmt.Println(len(s), utf8.RuneCountInString(s))
	fmt.Println(s[0], string(s[0]), s[:6])

	for i, r := range "çà" {
		fmt.Println(i, string(r), r)
	}
	// Output:
	// 15 14
	// 71 G Gopher
	// 0 ç 231
	// 2 à 224
}

func Example_chainesBrutes() {
	chemin := `C:\Users\gopher`
	multi := `ligne 1
ligne 2`
	fmt.Println(chemin)
	fmt.Println(multi)
	// Output:
	// C:\Users\gopher
	// ligne 1
	// ligne 2
}
