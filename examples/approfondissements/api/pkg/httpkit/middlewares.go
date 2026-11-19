// Package httpkit regroupe des briques HTTP génériques, sans lien avec le
// métier du service : middlewares et lecture-écriture de JSON. Il est placé
// dans pkg/ parce qu'il peut être réutilisé tel quel par d'autres projets.
package httpkit

import (
	"context"
	"log/slog"
	"net/http"
	"time"
	"uuid"
)

// Middleware enveloppe un handler pour lui ajouter un comportement.
type Middleware func(http.Handler) http.Handler

// Chaine applique les middlewares : le premier de la liste est le plus
// extérieur, donc le premier à voir la requête.
func Chaine(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type cleRequeteID struct{}

// RequeteID renvoie l'identifiant de la requête en cours.
func RequeteID(ctx context.Context) string {
	id, _ := ctx.Value(cleRequeteID{}).(string)
	return id
}

// Identifiant attribue un identifiant unique à chaque requête, transmis dans
// le contexte et renvoyé au client dans l'en-tête X-Request-ID.
func Identifiant() Middleware {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" || len(id) > 64 {
				id = uuid.NewV7().String()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), cleRequeteID{}, id)
			suivant.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// enregistreur mémorise le statut de la réponse, pour le journal.
type enregistreur struct {
	http.ResponseWriter
	statut int
}

func (e *enregistreur) WriteHeader(statut int) {
	e.statut = statut
	e.ResponseWriter.WriteHeader(statut)
}

// Unwrap permet à http.ResponseController d'atteindre le ResponseWriter
// d'origine (Flush, délais…).
func (e *enregistreur) Unwrap() http.ResponseWriter { return e.ResponseWriter }

// Journal écrit une ligne par requête : méthode, chemin, statut, durée.
func Journal(log *slog.Logger) Middleware {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debut := time.Now()
			rec := &enregistreur{ResponseWriter: w, statut: http.StatusOK}
			suivant.ServeHTTP(rec, r)
			log.InfoContext(r.Context(), "requête",
				slog.String("requete_id", RequeteID(r.Context())),
				slog.String("methode", r.Method),
				slog.String("chemin", r.URL.Path),
				slog.Int("statut", rec.statut),
				slog.Duration("duree", time.Since(debut)),
			)
		})
	}
}

// Recuperation transforme une panique dans un handler en erreur 500, au lieu
// de couper la connexion sans réponse.
func Recuperation(log *slog.Logger) Middleware {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if v == http.ErrAbortHandler { // abandon volontaire : on le propage
						panic(v)
					}
					log.ErrorContext(r.Context(), "panique", "valeur", v,
						"requete_id", RequeteID(r.Context()))
					EcritJSON(w, http.StatusInternalServerError, map[string]string{"erreur": "erreur interne"})
				}
			}()
			suivant.ServeHTTP(w, r)
		})
	}
}
