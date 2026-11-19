// Package web est un adaptateur primaire (« driving ») : il traduit des
// requêtes HTTP en appels aux cas d'usage, et leurs résultats en réponses.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/domaine"
)

// CasDUsage est ce que l'adaptateur attend de l'application.
type CasDUsage interface {
	Emprunte(ctx context.Context, membre, isbn string) (domaine.Emprunt, error)
	Rend(ctx context.Context, emprunt string) (int, error)
}

// Routes expose les cas d'usage en HTTP.
func Routes(app CasDUsage) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /emprunts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Membre string `json:"membre"`
			ISBN   string `json:"isbn"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			repond(w, http.StatusBadRequest, map[string]string{"erreur": "JSON invalide"})
			return
		}
		e, err := app.Emprunte(r.Context(), req.Membre, req.ISBN)
		if err != nil {
			erreur(w, err)
			return
		}
		repond(w, http.StatusCreated, map[string]any{
			"id": e.ID, "isbn": e.ISBN, "membre": e.Membre, "echeance": e.Echeance,
		})
	})

	mux.HandleFunc("POST /emprunts/{id}/retour", func(w http.ResponseWriter, r *http.Request) {
		penalite, err := app.Rend(r.Context(), r.PathValue("id"))
		if err != nil {
			erreur(w, err)
			return
		}
		repond(w, http.StatusOK, map[string]int{"penalite_centimes": penalite})
	})
	return mux
}

// erreur traduit les erreurs du domaine en statuts HTTP. C'est le seul
// endroit qui connaît à la fois le métier et le protocole.
func erreur(w http.ResponseWriter, err error) {
	statut := http.StatusInternalServerError
	switch {
	case errors.Is(err, domaine.ErrIntrouvable):
		statut = http.StatusNotFound
	case errors.Is(err, domaine.ErrLivreIndisponible),
		errors.Is(err, domaine.ErrQuotaAtteint),
		errors.Is(err, domaine.ErrMembreSuspendu),
		errors.Is(err, domaine.ErrDejaRendu):
		statut = http.StatusConflict
	}
	message := err.Error()
	if statut == http.StatusInternalServerError {
		message = "erreur interne"
	}
	repond(w, statut, map[string]string{"erreur": message})
}

func repond(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	json.NewEncoder(w).Encode(v)
}
