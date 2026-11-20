package main

import "fmt"

func ExampleCompte() {
	for _, f := range Compte("Le gopher creuse. Le crabe pince. Le gopher dort !", 3) {
		fmt.Println(f.Nombre, f.Mot)
	}
	// Output:
	// 3 le
	// 2 gopher
	// 1 crabe
}
