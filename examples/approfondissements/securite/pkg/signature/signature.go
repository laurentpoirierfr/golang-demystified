// Package signature montre les signatures post-quantiques ML-DSA (FIPS 204),
// disponibles dans la bibliothèque standard depuis Go 1.27 (crypto/mldsa).
package signature

import (
	"crypto/mldsa"
	"crypto/rand"
)

// contexte distingue les signatures de cette application de toute autre
// signature faite avec la même clé : une signature de facture ne peut pas
// être rejouée comme signature de contrat.
var contexte = &mldsa.Options{Context: "golang-demystified/factures/v1"}

// Signe signe le message avec la clé privée.
func Signe(cle *mldsa.PrivateKey, message []byte) ([]byte, error) {
	return cle.Sign(rand.Reader, message, contexte)
}

// Verifie vérifie la signature du message.
func Verifie(cle *mldsa.PublicKey, message, sig []byte) error {
	return mldsa.Verify(cle, message, sig, contexte)
}
