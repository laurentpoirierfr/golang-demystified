//go:build wasip1

// Command boucle est un greffon malveillant, pour les tests : son analyse ne
// se termine jamais, et il tente d'épuiser la mémoire.
package main

import "unsafe"

var garde [][]byte

//go:wasmexport alloue
func alloue(taille uint32) uint32 {
	b := make([]byte, max(taille, 1))
	garde = append(garde, b)
	return uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
}

//go:wasmexport libere
func libere(uint32) {}

//go:wasmexport analyse
func analyse(_, n uint32) uint64 {
	if n == 1 { // « 1 » : boucle infinie
		for {
		}
	}
	for { // sinon : demande toujours plus de mémoire
		garde = append(garde, make([]byte, 1<<20))
	}
}

func main() {}
