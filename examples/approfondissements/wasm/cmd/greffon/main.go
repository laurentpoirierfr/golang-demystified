//go:build wasip1

// Command greffon est un module WebAssembly (WASI) qui analyse un texte. Il
// n'a pas de main utile : compilé en « réacteur » (-buildmode=c-shared), il
// exporte des fonctions que son hôte appelle.
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o greffon.wasm ./cmd/greffon
//	tinygo build -target=wasip1 -buildmode=c-shared -o greffon.wasm ./cmd/greffon
//
// WebAssembly n'échange que des nombres. Un texte passe donc par la mémoire
// du module : l'hôte demande un tampon (alloue), y écrit le texte, appelle
// analyse, lit le résultat à l'adresse renvoyée, puis libère les tampons.
package main

import (
	"encoding/json"
	"unsafe"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

// tampons garde une référence vers chaque tampon prêté à l'hôte : sans elle,
// le ramasse-miettes de Go pourrait réutiliser la mémoire pendant que l'hôte
// y écrit.
var tampons = map[uint32][]byte{}

// alloue réserve un tampon de taille octets (au moins 1) et renvoie son
// adresse dans la mémoire du module.
//
//go:wasmexport alloue
func alloue(taille uint32) uint32 {
	b := make([]byte, max(taille, 1))
	adresse := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
	tampons[adresse] = b
	return adresse
}

// libere rend un tampon au ramasse-miettes.
//
//go:wasmexport libere
func libere(adresse uint32) {
	delete(tampons, adresse)
}

// analyse lit le texte de n octets à l'adresse donnée, et renvoie l'adresse
// et la longueur du résultat JSON, réunies dans un seul entier :
// adresse << 32 | longueur. L'hôte doit libérer ce résultat.
//
//go:wasmexport analyse
func analyse(adresse, n uint32) uint64 {
	tampon, ok := tampons[adresse]
	if !ok || int(n) > len(tampon) {
		return 0 // adresse inconnue : l'hôte reçoit un résultat vide
	}
	resultat, err := json.Marshal(texte.Analyse(string(tampon[:n]), 5))
	if err != nil {
		return 0
	}
	sortie := alloue(uint32(len(resultat)))
	copy(tampons[sortie], resultat)
	return uint64(sortie)<<32 | uint64(len(resultat))
}

// main n'est pas appelée en mode réacteur, mais doit exister.
func main() {}
