// Exemples du chapitre « Tableaux, slices et maps ».
package collections

import (
	"fmt"
	"maps"
	"slices"
)

func Example_tableaux() {
	var a [3]int
	b := [...]string{"Go", "Rust", "Zig"}
	a[1] = 42

	c := a // copie complète
	c[0] = 7
	fmt.Println(a, c, len(b))
	// Output: [0 42 0] [7 42 0] 3
}

func Example_slices() {
	s := []int{1, 2, 3}
	s = append(s, 4, 5)
	fmt.Println(s, len(s))

	t := s[1:3] // partage le même tableau sous-jacent
	t[0] = 20
	fmt.Println(t, s)
	// Output:
	// [1 2 3 4 5] 5
	// [20 3] [1 20 3 4 5]
}

func Example_make() {
	s := make([]int, 0, 10)
	fmt.Println(len(s), cap(s))
	for i := range 4 {
		s = append(s, i*i)
	}
	fmt.Println(s, len(s), cap(s))
	// Output:
	// 0 10
	// [0 1 4 9] 4 10
}

func Example_appendPartage() {
	base := make([]int, 3, 10)
	a := append(base, 1)
	b := append(base, 2) // écrit au même emplacement que a !
	fmt.Println(a[3], b[3])

	// Une expression à trois indices limite la capacité et force la copie.
	c := append(base[:3:3], 1)
	d := append(base[:3:3], 2)
	fmt.Println(c[3], d[3])
	// Output:
	// 2 2
	// 1 2
}

func Example_copy() {
	src := []int{1, 2, 3}
	dst := make([]int, len(src))
	n := copy(dst, src)
	dst[0] = 100
	fmt.Println(n, src, dst)

	clone := slices.Clone(src)
	fmt.Println(clone)
	// Output:
	// 3 [1 2 3] [100 2 3]
	// [1 2 3]
}

func Example_slicesPackage() {
	s := []int{5, 2, 8, 1}
	slices.Sort(s)
	i, trouve := slices.BinarySearch(s, 5)
	fmt.Println(s, i, trouve)
	fmt.Println(slices.Contains(s, 8), slices.Index(s, 2), slices.Max(s))
	fmt.Println(slices.Equal(s, []int{1, 2, 5, 8}))
	// Output:
	// [1 2 5 8] 2 true
	// true 1 8
	// true
}

func Example_maps() {
	ages := map[string]int{
		"alice": 31,
		"bob":   27,
	}
	ages["carol"] = 45
	delete(ages, "bob")

	age, ok := ages["bob"]
	fmt.Println(age, ok, len(ages))

	for _, nom := range slices.Sorted(maps.Keys(ages)) {
		fmt.Println(nom, ages[nom])
	}
	// Output:
	// 0 false 2
	// alice 31
	// carol 45
}

func Example_mapNil() {
	var m map[string]int
	fmt.Println(m["absent"], len(m)) // la lecture fonctionne
	// m["x"] = 1 provoquerait un panic : assignment to entry in nil map
	m = make(map[string]int)
	m["x"] = 1
	fmt.Println(m)
	// Output:
	// 0 0
	// map[x:1]
}

func Example_compteur() {
	mots := []string{"go", "rust", "go", "zig", "go"}
	compte := make(map[string]int)
	for _, m := range mots {
		compte[m]++
	}
	fmt.Println(compte)
	// Output: map[go:3 rust:1 zig:1]
}

func Example_clear() {
	s := []int{1, 2, 3}
	m := map[string]int{"a": 1}
	clear(s)
	clear(m)
	fmt.Println(s, len(m))
	// Output: [0 0 0] 0
}
