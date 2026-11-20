// Package divers contient les solutions des exercices du chapitre « Tour
// d'horizon ».
package divers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// Exercice 1 ★ : extraire avec une expression régulière.

// emailRe est compilée une seule fois, au démarrage du programme.
var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

// Emails renvoie les adresses trouvées dans un texte.
func Emails(texte string) []string {
	return emailRe.FindAllString(texte, -1)
}

// Exercice 2 ★★ : totaliser un CSV.

// TotalParCategorie lit un CSV « produit,categorie,prix », avec une ligne
// d'en-tête, et additionne les prix par catégorie.
func TotalParCategorie(r io.Reader) (map[string]float64, error) {
	lecteur := csv.NewReader(r)
	lecteur.FieldsPerRecord = 3
	if _, err := lecteur.Read(); err != nil { // l'en-tête
		return nil, fmt.Errorf("en-tête : %w", err)
	}
	totaux := map[string]float64{}
	for {
		ligne, err := lecteur.Read()
		if errors.Is(err, io.EOF) {
			return totaux, nil
		}
		if err != nil {
			return nil, err
		}
		prix, err := strconv.ParseFloat(ligne[2], 64)
		if err != nil {
			l, _ := lecteur.FieldPos(2)
			return nil, fmt.Errorf("ligne %d : prix %q invalide", l, ligne[2])
		}
		totaux[ligne[1]] += prix
	}
}

// Exercice 3 ★★★ : signer un message.

// Signe renvoie la signature HMAC-SHA256 d'un message, en hexadécimal.
func Signe(cle []byte, message string) string {
	mac := hmac.New(sha256.New, cle)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verifie indique si la signature est celle du message. hmac.Equal compare
// en temps constant.
func Verifie(cle []byte, message, signature string) bool {
	recue, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, cle)
	mac.Write([]byte(message))
	return hmac.Equal(recue, mac.Sum(nil))
}
