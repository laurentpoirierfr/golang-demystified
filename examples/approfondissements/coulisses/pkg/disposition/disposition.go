// Package disposition montre ce que révèle le package unsafe : la place des
// champs d'une structure en mémoire, et des conversions sans copie entre
// []byte et string.
package disposition

import "unsafe"

// Mal rangée : chaque champ est aligné sur sa taille, et le compilateur
// insère des octets de remplissage (padding) entre eux.
type MalRangee struct {
	Actif   bool    // 1 octet, puis 7 de remplissage
	Montant float64 // 8 octets
	Code    bool    // 1 octet, puis 3 de remplissage
	Nombre  int32   // 4 octets
	Visible bool    // 1 octet, puis 7 de remplissage pour aligner la suivante
}

// BienRangee contient les mêmes champs, du plus grand au plus petit.
type BienRangee struct {
	Montant float64
	Nombre  int32
	Actif   bool
	Code    bool
	Visible bool
}

// Champ décrit la place d'un champ.
type Champ struct {
	Nom              string
	Decalage, Taille uintptr
}

// Plan renvoie la place des champs de MalRangee, telle que le compilateur la
// calcule (unsafe.Offsetof, unsafe.Sizeof sont évalués à la compilation).
func Plan() []Champ {
	var m MalRangee
	return []Champ{
		{"Actif", unsafe.Offsetof(m.Actif), unsafe.Sizeof(m.Actif)},
		{"Montant", unsafe.Offsetof(m.Montant), unsafe.Sizeof(m.Montant)},
		{"Code", unsafe.Offsetof(m.Code), unsafe.Sizeof(m.Code)},
		{"Nombre", unsafe.Offsetof(m.Nombre), unsafe.Sizeof(m.Nombre)},
		{"Visible", unsafe.Offsetof(m.Visible), unsafe.Sizeof(m.Visible)},
	}
}

// Chaine convertit des octets en chaîne SANS copie. La chaîne partage la
// mémoire des octets : les modifier ensuite modifierait la chaîne, qui est
// pourtant immuable en Go. À réserver aux octets que plus personne ne
// modifie.
func Chaine(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Octets convertit une chaîne en octets SANS copie. Les octets obtenus ne
// doivent jamais être modifiés : la mémoire d'une chaîne peut être en
// lecture seule, et une écriture ferait planter le programme.
func Octets(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
