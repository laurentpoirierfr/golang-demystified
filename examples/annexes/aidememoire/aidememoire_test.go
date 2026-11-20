package aidememoire

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"
)

type point struct{ X, Y int }

func Example_verbes() {
	p := point{1, 2}
	fmt.Printf("%v|%+v|%#v|%T\n", p, p, p, p)
	fmt.Printf("%d|%5d|%-5d|%05d|%x|%X|%o|%b|%c|%U\n", 42, 42, 42, 42, 255, 255, 8, 5, 'G', 'G')
	fmt.Printf("%f|%.2f|%8.3f|%e|%g\n", 3.14159, 3.14159, 3.14159, 1234.5678, 0.000012)
	fmt.Printf("%s|%q|%x|%10s|%-10s|\n", "go", "go", "go", "go", "go")
	fmt.Printf("%t|%v|%v|%%\n", true, nil, []int(nil))
	// Output:
	// {1 2}|{X:1 Y:2}|aidememoire.point{X:1, Y:2}|aidememoire.point
	// 42|   42|42   |00042|ff|FF|10|101|G|U+0047
	// 3.141590|3.14|   3.142|1.234568e+03|1.2e-05
	// go|"go"|676f|        go|go        |
	// true|<nil>|[]|%
}

func Example_temps() {
	t := time.Date(2026, time.November, 20, 14, 5, 9, 0, time.UTC)
	fmt.Println(t.Format("2006-01-02 15:04:05"))
	fmt.Println(t.Format("02/01/2006 à 15h04"))
	fmt.Println(t.Format(time.RFC3339))
	fmt.Println(t.Format(time.DateOnly), t.Format(time.Kitchen))
	fmt.Println(t.Format("Mon 2 Jan"), t.Weekday())
	d := 90*time.Minute + 30*time.Second
	fmt.Println(d, d.Minutes(), d.Round(time.Hour))
	// Output:
	// 2026-11-20 14:05:09
	// 20/11/2026 à 14h05
	// 2026-11-20T14:05:09Z
	// 2026-11-20 2:05PM
	// Fri 20 Nov Friday
	// 1h30m30s 90.5 2h0m0s
}

func Example_slices() {
	s := make([]int, 3, 10)
	t := append(s, 4) // même tableau : la capacité suffit
	t[0] = 99
	fmt.Println(s, t, len(t), cap(t))

	u := s[0:2:2] // capacité limitée à 2 : le prochain append copie
	u = append(u, 7)
	u[1] = -1
	fmt.Println(s, u)

	var nulle []int
	fmt.Println(nulle == nil, len(nulle), append(nulle, 1))
	fmt.Println(slices.Index([]string{"a", "b"}, "b"), slices.Contains(s, 99), slices.Sorted(slices.Values([]int{3, 1, 2})))
	// Output:
	// [99 0 0] [99 0 0 4] 4 10
	// [99 0 0] [99 -1 7]
	// true 0 [1]
	// 1 true [1 2 3]
}

func Example_maps() {
	m := map[string]int{"a": 1}
	v, ok := m["absent"]
	fmt.Println(v, ok, len(m))
	delete(m, "a")
	delete(m, "absent") // sans effet, sans erreur
	fmt.Println(m)
	defer func() { fmt.Println("panique :", recover()) }()
	var nulle map[string]int
	fmt.Println(nulle["x"]) // lire une map nil : la valeur zéro
	nulle["x"] = 1          //nolint:staticcheck // le piège démontré : écrire dans une map nil panique
	// Output:
	// 0 false 1
	// map[]
	// 0
	// panique : assignment to entry in nil map
}

func Example_chaines() {
	s := "héllo"
	fmt.Println(len(s), len([]rune(s)), s[1], string(s[0]))
	for i, r := range "hé!" {
		fmt.Printf("%d:%c ", i, r)
	}
	fmt.Println("fin")
	fmt.Println(strings.Fields(" a  b "), strings.Split("a,b,,c", ","), strings.Repeat("ab", 3))
	// Output:
	// 6 5 195 h
	// 0:h 1:é 3:! fin
	// [a b] [a b  c] ababab
}

func Example_defer() {
	for i := range 3 {
		defer fmt.Print(i, " ") // ordre inverse ; i évalué à l'appel de defer
	}
	fmt.Print("corps ")
	// Output:
	// corps 2 1 0
}

type monErreur struct{ code int }

func (e *monErreur) Error() string { return fmt.Sprint("code ", e.code) }

func Example_erreurs() {
	err := fmt.Errorf("lecture de config : %w", fs.ErrNotExist)
	fmt.Println(err, errors.Is(err, fs.ErrNotExist))

	_, err = os.Open("/absent")
	var pe *fs.PathError
	fmt.Println(errors.As(err, &pe), pe.Op)

	err = fmt.Errorf("appel : %w", &monErreur{404})
	if e, ok := errors.AsType[*monErreur](err); ok {
		fmt.Println("code", e.code)
	}
	fmt.Println(errors.Join(errors.New("a"), nil, errors.New("b")))
	// Output:
	// lecture de config : file does not exist true
	// true open
	// code 404
	// a
	// b
}

func Example_interfaceNil() {
	var p *monErreur                  //nolint:staticcheck // pointeur nil, mis dans une interface : le piège démontré
	var err error = p                 //nolint:staticcheck // le piège démontré
	fmt.Println(p == nil, err == nil) //nolint:staticcheck // l'interface porte un type : elle n'est pas nil
	// Output:
	// true false
}

func Example_channels() {
	c := make(chan int, 2)
	c <- 1
	c <- 2
	close(c)
	v, ok := <-c
	fmt.Println(v, ok, len(c))
	<-c
	v, ok = <-c // fermé et vide : immédiat, valeur zéro
	fmt.Println(v, ok)
	for range c { // range s'arrête sur un channel fermé
	}
	defer func() { fmt.Println("panique :", recover()) }()
	c <- 3
	// Output:
	// 1 true 1
	// 0 false
	// panique : send on closed channel
}

func Example_conversions() {
	f := 3.99
	var u8 uint8 = 255
	u8++
	fmt.Println(int(f), int(-f), u8, 7/2, -7/2, 7%3, -7%3, 7.0/2)
	fmt.Println(new(42) != nil, *new("go")) // new accepte une expression (Go 1.26)
	// Output:
	// 3 -3 0 3 -3 1 -1 3.5
	// true go
}
