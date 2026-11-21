// Package web expose le service d'inscription en HTTP.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
)

// Config est la configuration du serveur.
type Config struct {
	Adresse string
}

// Inscripteur est ce dont le handler a besoin : une seule méthode.
type Inscripteur interface {
	Inscrit(ctx context.Context, courriel, nom string) (inscription.Membre, error)
}

// Routes renvoie le handler HTTP : POST /membres.
func Routes(s Inscripteur) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /membres", func(w http.ResponseWriter, r *http.Request) {
		var demande struct{ Courriel, Nom string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&demande); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}
		m, err := s.Inscrit(r.Context(), demande.Courriel, demande.Nom)
		switch {
		case errors.Is(err, inscription.ErrDejaInscrit):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case err != nil && m.Courriel == "": // rien n'est inscrit
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"courriel": m.Courriel, "nom": m.Nom})
	})
	return mux
}

// NouveauServeur crée le serveur HTTP, avec des délais raisonnables.
func NouveauServeur(cfg Config, h http.Handler) *http.Server {
	return &http.Server{Addr: cfg.Adresse, Handler: h, ReadHeaderTimeout: 5 * time.Second}
}
