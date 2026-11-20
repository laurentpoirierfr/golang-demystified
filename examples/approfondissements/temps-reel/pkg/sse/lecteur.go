package sse

import (
	"bufio"
	"io"
	"iter"
	"strconv"
	"strings"
	"time"
)

// Lit décode un flux d'événements, comme le fait EventSource dans un
// navigateur : pour un client Go, un outil ou un test. L'itérateur s'arrête à
// la fin du flux ou à la première erreur de lecture.
//
//	for e, err := range sse.Lit(reponse.Body) { … }
func Lit(r io.Reader) iter.Seq2[Evenement, error] {
	return func(yield func(Evenement, error) bool) {
		lignes := bufio.NewScanner(r)
		var e Evenement
		var donnees []string
		aDesDonnees := false
		for lignes.Scan() {
			ligne := strings.TrimSuffix(lignes.Text(), "\r")
			if ligne == "" { // fin de l'événement
				if aDesDonnees {
					e.Donnees = strings.Join(donnees, "\n")
					if !yield(e, nil) {
						return
					}
				}
				// L'identifiant est conservé d'un événement à l'autre ; le
				// reste repart de zéro.
				e = Evenement{ID: e.ID}
				donnees, aDesDonnees = nil, false
				continue
			}
			if strings.HasPrefix(ligne, ":") {
				continue // commentaire
			}
			champ, valeur, _ := strings.Cut(ligne, ":")
			valeur = strings.TrimPrefix(valeur, " ")
			switch champ {
			case "id":
				e.ID = valeur
			case "event":
				e.Type = valeur
			case "data":
				donnees = append(donnees, valeur)
				aDesDonnees = true
			case "retry":
				if ms, err := strconv.Atoi(valeur); err == nil {
					e.Reessai = time.Duration(ms) * time.Millisecond
				}
			}
		}
		if err := lignes.Err(); err != nil {
			yield(Evenement{}, err)
		}
	}
}
