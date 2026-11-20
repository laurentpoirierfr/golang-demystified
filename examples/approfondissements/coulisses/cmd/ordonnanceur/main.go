// Command ordonnanceur montre deux comportements de l'ordonnanceur de Go.
//
//	ordonnanceur          GOMAXPROCS, et ce qui le détermine
//	ordonnanceur -boucle  une boucle infinie ne bloque pas les autres goroutines
//
// Avec -boucle, le programme n'a qu'un seul fil d'exécution (GOMAXPROCS=1),
// et une goroutine calcule sans jamais s'arrêter. Depuis Go 1.14, le runtime
// l'interrompt (préemption asynchrone) pour laisser travailler les autres :
// main reprend la main. Avec GODEBUG=asyncpreemptoff=1, il ne le peut plus,
// et le programme reste bloqué.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

// compteur est lu par main pendant que calcule l'incrémente : il est donc
// atomique, sans quoi ce serait une course de données.
var compteur atomic.Uint64

//go:noinline
func calcule() {
	for { // aucune fonction appelée, aucun point d'arrêt naturel
		compteur.Add(1)
	}
}

func main() {
	boucle := flag.Bool("boucle", false, "démontrer la préemption")
	flag.Parse()

	if !*boucle {
		fmt.Printf("processeurs logiques (NumCPU) : %d\n", runtime.NumCPU())
		fmt.Printf("GOMAXPROCS                    : %d\n", runtime.GOMAXPROCS(0))
		fmt.Printf("variable GOMAXPROCS           : %q\n", os.Getenv("GOMAXPROCS"))
		return
	}

	runtime.GOMAXPROCS(1)
	go calcule()
	debut := time.Now()
	time.Sleep(50 * time.Millisecond) // main cède le seul fil à calcule
	fmt.Printf("main a repris après %v ; la boucle en était à %d tours\n",
		time.Since(debut).Round(time.Millisecond), compteur.Load())
}
