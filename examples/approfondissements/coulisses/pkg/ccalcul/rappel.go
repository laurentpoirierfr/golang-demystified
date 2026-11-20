//go:build cgo

package ccalcul

// Les déclarations (#include) sont permises ici ; les définitions, non.

// #include <stdint.h>
import "C"

import "runtime/cgo"

// appelleGo est appelée par le code C. Un fichier qui exporte des fonctions
// (//export) ne peut pas définir de fonctions C dans son préambule : elles
// sont dans ccalcul.c.
//
//export appelleGo
func appelleGo(poignee C.uintptr_t, valeur C.int64_t) {
	f := cgo.Handle(poignee).Value().(func(int64))
	f(int64(valeur))
}
