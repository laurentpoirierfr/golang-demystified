// Package intergiciel illustre le motif du middleware : une fonction qui
// reçoit un http.Handler et en renvoie un autre, qui l'enveloppe. Ce motif
// n'a besoin d'aucun framework : il découle de l'interface http.Handler.
package intergiciel

import (
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// Intergiciel enveloppe un handler.
type Intergiciel func(http.Handler) http.Handler

// Chaine applique les intergiciels à h. Le premier de la liste est le plus
// extérieur : il voit la requête en premier, et la réponse en dernier.
//
//	Chaine(h, Recupere(log), Journal(log))  ≡  Recupere(log)(Journal(log)(h))
func Chaine(h http.Handler, intergiciels ...Intergiciel) http.Handler {
	for _, m := range slices.Backward(intergiciels) {
		h = m(h)
	}
	return h
}

// Journal note chaque requête : méthode, chemin, statut, durée.
func Journal(log *slog.Logger) Intergiciel {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debut := time.Now()
			enr := &enregistreur{ResponseWriter: w, statut: http.StatusOK}
			suivant.ServeHTTP(enr, r)
			log.InfoContext(r.Context(), "requête", "methode", r.Method, "chemin", r.URL.Path,
				"statut", enr.statut, "duree", time.Since(debut))
		})
	}
}

// Recupere transforme une panique en réponse 500, au lieu de couper la
// connexion sans réponse.
func Recupere(log *slog.Logger) Intergiciel {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.ErrorContext(r.Context(), "panique", "valeur", v, "chemin", r.URL.Path)
					http.Error(w, "erreur interne", http.StatusInternalServerError)
				}
			}()
			suivant.ServeHTTP(w, r)
		})
	}
}

// enregistreur retient le statut écrit par le handler. Il intègre
// http.ResponseWriter : toutes les autres méthodes sont celles d'origine.
type enregistreur struct {
	http.ResponseWriter
	statut int
}

func (e *enregistreur) WriteHeader(statut int) {
	e.statut = statut
	e.ResponseWriter.WriteHeader(statut)
}

// Unwrap permet à http.ResponseController d'atteindre le ResponseWriter
// d'origine (Flush, délais), à travers l'enveloppe.
func (e *enregistreur) Unwrap() http.ResponseWriter { return e.ResponseWriter }
