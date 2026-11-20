//go:build !cgo

// Package ccalcul : version en Go pur, compilée quand cgo n'est pas
// disponible. Les signatures sont celles de ccalcul.go.
package ccalcul

// AvecC indique si le code C est utilisé.
const AvecC = false

// Somme additionne les éléments.
func Somme(t []int64) int64 {
	var s int64
	for _, v := range t {
		s += v
	}
	return s
}

// Parcourt appelle f pour chaque élément.
func Parcourt(t []int64, f func(int64)) {
	for _, v := range t {
		f(v)
	}
}
