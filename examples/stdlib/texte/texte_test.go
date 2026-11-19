// Exemples du chapitre « Texte : fmt, strings, strconv ».
package texte

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"
)

type Point struct{ X, Y int }

func Example_verbes() {
	p := Point{1, 2}
	fmt.Printf("%v | %+v | %#v | %T\n", p, p, p, p)
	fmt.Printf("%d | %5d | %-5d| %05d | %x | %b\n", 42, 42, 42, 42, 255, 5)
	fmt.Printf("%f | %.2f | %8.3f | %e | %g\n", 3.14159, 3.14159, 3.14159, 1234.5678, 0.000012)
	fmt.Printf("%s | %q | %10s | %-10s|\n", "go", "go", "droite", "gauche")
	fmt.Printf("%t | %c | %U\n", true, 'é', 'é')
	// Output:
	// {1 2} | {X:1 Y:2} | texte.Point{X:1, Y:2} | texte.Point
	// 42 |    42 | 42   | 00042 | ff | 101
	// 3.141590 | 3.14 |    3.142 | 1.234568e+03 | 1.2e-05
	// go | "go" |     droite | gauche    |
	// true | é | U+00E9
}

func Example_sprint() {
	s := fmt.Sprintf("%s a %d ans", "Alice", 31)
	err := fmt.Errorf("utilisateur %q introuvable", "bob")
	b := fmt.Appendf(nil, "id=%d", 7)
	fmt.Println(s)
	fmt.Println(err)
	fmt.Println(string(b))
	// Output:
	// Alice a 31 ans
	// utilisateur "bob" introuvable
	// id=7
}

func Example_strings() {
	s := "  Le Gopher creuse  "
	fmt.Println(strings.TrimSpace(s))
	fmt.Println(strings.ToUpper("go"), strings.ToLower("GO"))
	fmt.Println(strings.Contains(s, "Gopher"), strings.HasPrefix("golang", "go"), strings.HasSuffix("main.go", ".go"))
	fmt.Println(strings.Index("chercher", "er"), strings.LastIndex("chercher", "er"), strings.Count("cheese", "e"))
	fmt.Println(strings.Replace("a-b-c", "-", "+", 1), strings.ReplaceAll("a-b-c", "-", "+"))
	fmt.Println(strings.Split("a,b,c", ","), strings.Fields(" un  deux trois "), len(strings.Fields(s)))
	fmt.Println(strings.Join([]string{"a", "b"}, "/"), strings.Repeat("ab", 3))
	fmt.Println(strings.EqualFold("Go", "GO"))
	// Output:
	// Le Gopher creuse
	// GO go
	// true true true
	// 2 6 3
	// a+b-c a+b+c
	// [a b c] [un deux trois] 3
	// a/b ababab
	// true
}

func Example_cut() {
	cle, valeur, ok := strings.Cut("port=8080", "=")
	fmt.Println(cle, valeur, ok)

	dossier, fichier, _ := strings.CutLast("/home/gopher/notes.txt", "/")
	fmt.Println(dossier, fichier)

	reste, ok := strings.CutPrefix("Bearer abc123", "Bearer ")
	fmt.Println(reste, ok)
	// Output:
	// port 8080 true
	// /home/gopher notes.txt
	// abc123 true
}

func Example_builder() {
	var b strings.Builder
	for i := range 3 {
		fmt.Fprintf(&b, "ligne %d\n", i)
	}
	b.WriteString("fin")
	fmt.Println(b.String())
	// Output:
	// ligne 0
	// ligne 1
	// ligne 2
	// fin
}

func Example_lines() {
	texte := "alpha\nbeta\ngamma\n"
	for ligne := range strings.Lines(texte) {
		fmt.Printf("%q;", ligne)
	}
	fmt.Println()
	for mot := range strings.FieldsSeq("un deux  trois") {
		fmt.Print("[", mot, "]")
	}
	fmt.Println()
	// Output:
	// "alpha\n";"beta\n";"gamma\n";
	// [un][deux][trois]
}

func Example_strconv() {
	n, _ := strconv.Atoi("42")
	f, _ := strconv.ParseFloat("3.14", 64)
	b, _ := strconv.ParseBool("true")
	i, _ := strconv.ParseInt("ff", 16, 64)
	fmt.Println(n, f, b, i)
	fmt.Println(strconv.Itoa(42), strconv.FormatFloat(3.14159, 'f', 2, 64), strconv.FormatInt(255, 2))
	fmt.Println(strconv.Quote("dit \"bonjour\""))

	_, err := strconv.ParseInt("999999999999999999999", 10, 64)
	fmt.Println(err)
	// Output:
	// 42 3.14 true 255
	// 42 3.14 11111111
	// "dit \"bonjour\""
	// strconv.ParseInt: parsing "999999999999999999999": value out of range
}

func Example_unicode() {
	compte := map[string]int{}
	for _, r := range "Go 1.27, été !" {
		switch {
		case unicode.IsLetter(r):
			compte["lettres"]++
		case unicode.IsDigit(r):
			compte["chiffres"]++
		case unicode.IsSpace(r):
			compte["espaces"]++
		default:
			compte["autres"]++
		}
	}
	fmt.Println(compte)
	// Output: map[autres:3 chiffres:3 espaces:3 lettres:5]
}

func Example_tabwriter() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NOM\tVERSION\tANNÉE")
	fmt.Fprintln(w, "Go\t1.27\t2026")
	fmt.Fprintln(w, "Generics\t1.18\t2022")
	w.Flush()
	// Output:
	// NOM       VERSION  ANNÉE
	// Go        1.27     2026
	// Generics  1.18     2022
}
