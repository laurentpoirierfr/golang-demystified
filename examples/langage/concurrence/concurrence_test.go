// Exemples du chapitre « Goroutines et channels ».
package concurrence

import (
	"fmt"
	"sync"
	"time"
)

func Example_waitGroup() {
	var wg sync.WaitGroup
	resultats := make([]int, 5)
	for i := range 5 {
		wg.Go(func() { // Go 1.25
			resultats[i] = i * i
		})
	}
	wg.Wait()
	fmt.Println(resultats)
	// Output: [0 1 4 9 16]
}

func Example_channel() {
	ch := make(chan string)
	go func() {
		ch <- "ping"
	}()
	msg := <-ch
	fmt.Println(msg)
	// Output: ping
}

func producteur(n int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for i := range n {
			ch <- i
		}
	}()
	return ch
}

func Example_rangeChannel() {
	somme := 0
	for v := range producteur(5) {
		somme += v
	}
	fmt.Println(somme)
	// Output: 10
}

func Example_buffer() {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2 // ne bloque pas : le buffer a deux places
	fmt.Println(len(ch), cap(ch))
	fmt.Println(<-ch, <-ch)
	// Output:
	// 2 2
	// 1 2
}

func Example_select() {
	resultat := make(chan string, 1) // buffer : la goroutine ne reste pas bloquée
	go func() {
		time.Sleep(time.Second) // un calcul trop long
		resultat <- "terminé"
	}()

	select {
	case r := <-resultat:
		fmt.Println(r)
	case <-time.After(10 * time.Millisecond):
		fmt.Println("délai dépassé")
	}
	// Output: délai dépassé
}

type Compteur struct {
	mu sync.Mutex
	n  map[string]int
}

func (c *Compteur) Incremente(cle string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[cle]++
}

func Example_mutex() {
	c := Compteur{n: make(map[string]int)}
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() { c.Incremente("visites") })
	}
	wg.Wait()
	fmt.Println(c.n["visites"])
	// Output: 1000
}
