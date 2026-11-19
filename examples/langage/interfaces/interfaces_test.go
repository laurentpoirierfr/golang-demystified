// Exemples du chapitre « Interfaces ».
package interfaces

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

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

func Example_implementation() {
	formes := []Forme{Rectangle{3, 4}, Cercle{1}}
	total := 0.0
	for _, f := range formes {
		total += f.Aire()
	}
	fmt.Printf("%.2f\n", total)
	// Output: 15.14
}

// Le compilateur vérifie que Rectangle satisfait Forme.
var _ Forme = Rectangle{}

type Temperature float64

func (t Temperature) String() string { return fmt.Sprintf("%.1f °C", float64(t)) }

func Example_stringer() {
	fmt.Println(Temperature(21.456))
	// Output: 21.5 °C
}

// compteMots accepte n'importe quelle source : fichier, réseau, chaîne…
func compteMots(r io.Reader) (int, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	return len(strings.Fields(string(data))), nil
}

func Example_ioReader() {
	n, _ := compteMots(strings.NewReader("le gopher creuse un terrier"))
	fmt.Println(n)
	// Output: 5
}

func Example_assertion() {
	var f Forme = Cercle{2}

	c := f.(Cercle) // panic si f n'est pas un Cercle
	fmt.Println(c.Rayon)

	if r, ok := f.(Rectangle); ok {
		fmt.Println("rectangle", r)
	} else {
		fmt.Println("pas un rectangle")
	}
	// Output:
	// 2
	// pas un rectangle
}

func decris(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case int, int64:
		return fmt.Sprintf("entier %v", x)
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

func Example_typeSwitch() {
	fmt.Println(decris(42))
	fmt.Println(decris("Go"))
	fmt.Println(decris(Temperature(3)))
	fmt.Println(decris(errors.New("oups")))
	fmt.Println(decris(3.14))
	fmt.Println(decris(nil))
	// Output:
	// entier 42
	// chaîne de 2 octets
	// stringer : 3.0 °C
	// erreur : oups
	// autre (float64)
	// nil
}

type Compteur struct{ n int }

func (c *Compteur) Incremente() { c.n++ }

type Incrementeur interface{ Incremente() }

func Example_recepteurPointeur() {
	var i Incrementeur = &Compteur{} // Compteur{} ne compile pas
	i.Incremente()
	fmt.Println(i.(*Compteur).n)
	// Output: 1
}

type MonErreur struct{}

func (*MonErreur) Error() string { return "mon erreur" }

func peutEchouer(echec bool) error {
	var e *MonErreur // nil
	if echec {
		e = &MonErreur{}
	}
	return e // piège : interface non nil contenant un pointeur nil
}

func Example_nilInterface() {
	err := peutEchouer(false)
	fmt.Println(err == nil)
	// Output: false
}

// majuscules décore un io.Writer en mettant le texte en majuscules.
type majuscules struct{ w io.Writer }

func (m majuscules) Write(p []byte) (int, error) {
	return m.w.Write([]byte(strings.ToUpper(string(p))))
}

func Example_composition() {
	var w io.Writer = majuscules{os.Stdout}
	fmt.Fprintln(w, "bonjour, gopher")
	// Output: BONJOUR, GOPHER
}
