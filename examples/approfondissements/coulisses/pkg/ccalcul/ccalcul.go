//go:build cgo

// Package ccalcul appelle du code C depuis Go avec cgo, et du code Go depuis C.
// Sans cgo (CGO_ENABLED=0, compilation croisée), le fichier pur.go fournit
// les mêmes fonctions en Go : le package se compile partout.
package ccalcul

// #include "ccalcul.h"
import "C"

import (
	"runtime/cgo"
	"unsafe"
)

// AvecC indique si le code C est utilisé.
const AvecC = true

// Somme additionne les éléments, en C. Passer un pointeur vers de la
// mémoire Go est permis le temps de l'appel, si cette mémoire ne contient
// pas elle-même de pointeurs Go : c'est le cas d'un []int64.
func Somme(t []int64) int64 {
	if len(t) == 0 {
		return 0
	}
	return int64(C.somme((*C.int64_t)(unsafe.Pointer(&t[0])), C.size_t(len(t))))
}

// Parcourt appelle f pour chaque élément, depuis une boucle écrite en C. Une
// fonction Go ne peut pas être passée à C ; on passe une poignée
// (cgo.Handle), un simple entier que C rend à Go, qui retrouve la fonction.
func Parcourt(t []int64, f func(int64)) {
	if len(t) == 0 {
		return
	}
	poignee := cgo.NewHandle(f)
	defer poignee.Delete() // sinon, la fonction ne serait jamais libérée
	C.parcourt((*C.int64_t)(unsafe.Pointer(&t[0])), C.size_t(len(t)), C.uintptr_t(poignee))
}
