package signature_test

import (
	"crypto/mldsa"
	"fmt"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/signature"
)

func Example() {
	cle, err := mldsa.GenerateKey(mldsa.MLDSA65())
	if err != nil {
		panic(err)
	}
	facture := []byte("Facture n° 42 : 1 250,00 €")
	sig, err := signature.Signe(cle, facture)
	if err != nil {
		panic(err)
	}
	fmt.Println("taille de la signature :", len(sig), "octets")
	fmt.Println("taille de la clé publique :", len(cle.PublicKey().Bytes()), "octets")

	fmt.Println("authentique :", signature.Verifie(cle.PublicKey(), facture, sig) == nil)
	falsifiee := []byte("Facture n° 42 : 9 250,00 €")
	fmt.Println("falsifiée :", signature.Verifie(cle.PublicKey(), falsifiee, sig) == nil)
	// Output:
	// taille de la signature : 3309 octets
	// taille de la clé publique : 1952 octets
	// authentique : true
	// falsifiée : false
}
