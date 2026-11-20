// Command course contient une course de données : deux goroutines modifient
// le même compteur sans synchronisation. Le compilateur Go l'accepte ; le
// détecteur de courses la signale à l'exécution :
//
//	go run -race ./course
//
// La même erreur, écrite en Rust (../../rust/course), ne compile pas.
package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for range 100_000 {
				compteur++ // lecture et écriture concurrentes, sans verrou
			}
		})
	}
	wg.Wait()
	fmt.Println(compteur) // rarement 200 000
}
