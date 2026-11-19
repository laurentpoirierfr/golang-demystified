// Exemples du chapitre « Collections : slices, maps, iter, cmp ».
package collections

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

type Personne struct {
	Nom string
	Age int
}

var equipe = []Personne{
	{"Alice", 31},
	{"Bob", 27},
	{"Carol", 31},
	{"Dave", 45},
}

func Example_sortFunc() {
	p := slices.Clone(equipe)
	slices.SortFunc(p, func(a, b Personne) int {
		return cmp.Or(
			cmp.Compare(b.Age, a.Age), // âge décroissant
			strings.Compare(a.Nom, b.Nom),
		)
	})
	fmt.Println(p)
	// Output: [{Dave 45} {Alice 31} {Carol 31} {Bob 27}]
}

func Example_recherche() {
	i := slices.IndexFunc(equipe, func(p Personne) bool { return p.Age > 40 })
	fmt.Println(i, equipe[i].Nom)
	fmt.Println(slices.ContainsFunc(equipe, func(p Personne) bool { return p.Nom == "Eve" }))
	plusVieux := slices.MaxFunc(equipe, func(a, b Personne) int { return cmp.Compare(a.Age, b.Age) })
	fmt.Println(plusVieux)
	// Output:
	// 3 Dave
	// false
	// {Dave 45}
}

func Example_modifier() {
	s := []int{1, 2, 3, 4, 5}
	s = slices.Insert(s, 1, 10, 11)
	fmt.Println(s)
	s = slices.Delete(s, 0, 2)
	fmt.Println(s)
	s = slices.DeleteFunc(s, func(n int) bool { return n%2 == 0 })
	fmt.Println(s)
	fmt.Println(slices.Compact([]int{1, 1, 2, 2, 2, 3, 1}))
	// Output:
	// [1 10 11 2 3 4 5]
	// [11 2 3 4 5]
	// [11 3 5]
	// [1 2 3 1]
}

func Example_chunk() {
	for lot := range slices.Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3) {
		fmt.Print(lot)
	}
	fmt.Println()
	// Output: [1 2 3][4 5 6][7]
}

func Example_maps() {
	stock := map[string]int{"vis": 120, "écrou": 80, "clou": 0}
	copie := maps.Clone(stock)
	maps.DeleteFunc(copie, func(_ string, q int) bool { return q == 0 })
	fmt.Println(len(stock), len(copie))
	fmt.Println(slices.Sorted(maps.Keys(copie)))
	fmt.Println(maps.Equal(stock, copie))
	// Output:
	// 3 2
	// [vis écrou]
	// false
}

func Example_collect() {
	ages := maps.Collect(func(yield func(string, int) bool) {
		for _, p := range equipe {
			if !yield(p.Nom, p.Age) {
				return
			}
		}
	})
	fmt.Println(ages["Carol"])
	// Output: 31
}

// Filtre ne garde que les éléments qui satisfont garde.
func Filtre[V any](seq iter.Seq[V], garde func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if garde(v) && !yield(v) {
				return
			}
		}
	}
}

// Transforme applique f à chaque élément.
func Transforme[V, W any](seq iter.Seq[V], f func(V) W) iter.Seq[W] {
	return func(yield func(W) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func Example_pipelineIterateurs() {
	nombres := slices.Values([]int{1, 2, 3, 4, 5, 6})
	pairs := Filtre(nombres, func(n int) bool { return n%2 == 0 })
	carres := Transforme(pairs, func(n int) int { return n * n })
	fmt.Println(slices.Collect(carres))
	// Output: [4 16 36]
}

func Example_pull() {
	suivant, stop := iter.Pull(slices.Values([]string{"a", "b"}))
	defer stop()
	for {
		v, ok := suivant()
		if !ok {
			break
		}
		fmt.Print(v)
	}
	fmt.Println()
	// Output: ab
}

func Example_cmpOr() {
	config := map[string]string{"hote": ""}
	hote := cmp.Or(config["hote"], "localhost")
	port := cmp.Or(config["port"], "8080")
	fmt.Println(hote + ":" + port)
	// Output: localhost:8080
}
