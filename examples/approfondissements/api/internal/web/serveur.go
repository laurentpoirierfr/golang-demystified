// Package web expose les tâches à travers une API HTTP JSON.
package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/httpkit"
)

// Store est ce dont l'API a besoin pour stocker les tâches. L'interface est
// définie ici, par celui qui l'utilise : n'importe quel stockage convient, y
// compris un double de test.
type Store interface {
	Liste(ctx context.Context) ([]tache.Tache, error)
	Trouve(ctx context.Context, id int64) (tache.Tache, error)
	Cree(ctx context.Context, titre string) (tache.Tache, error)
	MarqueFaite(ctx context.Context, id int64, faite bool) (tache.Tache, error)
	Supprime(ctx context.Context, id int64) error
	Ping(ctx context.Context) error
}

// Serveur porte les dépendances des handlers.
type Serveur struct {
	store Store
	log   *slog.Logger
}

// Nouveau crée le serveur.
func Nouveau(store Store, log *slog.Logger) *Serveur {
	return &Serveur{store: store, log: log}
}

// Routes renvoie le handler complet, middlewares compris.
func (s *Serveur) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /taches", s.liste)
	mux.HandleFunc("POST /taches", s.cree)
	mux.HandleFunc("GET /taches/{id}", s.detail)
	mux.HandleFunc("PATCH /taches/{id}", s.modifie)
	mux.HandleFunc("DELETE /taches/{id}", s.supprime)
	mux.HandleFunc("GET /sante", s.sante)

	// Le premier middleware de la liste est le plus extérieur.
	return httpkit.Chaine(mux,
		httpkit.Identifiant(),
		httpkit.Journal(s.log),
		httpkit.Recuperation(s.log),
	)
}

func (s *Serveur) liste(w http.ResponseWriter, r *http.Request) {
	taches, err := s.store.Liste(r.Context())
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, taches)
}

func (s *Serveur) cree(w http.ResponseWriter, r *http.Request) {
	var entree struct {
		Titre string `json:"titre"`
	}
	if err := httpkit.LitJSON(w, r, &entree); err != nil {
		s.erreur(w, r, err)
		return
	}
	titre, err := tache.ValideTitre(entree.Titre)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	t, err := s.store.Cree(r.Context(), titre)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/taches/%d", t.ID))
	httpkit.EcritJSON(w, http.StatusCreated, t)
}

func (s *Serveur) detail(w http.ResponseWriter, r *http.Request) {
	id, err := identifiant(r)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	t, err := s.store.Trouve(r.Context(), id)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, t)
}

func (s *Serveur) modifie(w http.ResponseWriter, r *http.Request) {
	id, err := identifiant(r)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	var entree struct {
		Faite *bool `json:"faite"` // pointeur : distingue « absent » de false
	}
	if err := httpkit.LitJSON(w, r, &entree); err != nil {
		s.erreur(w, r, err)
		return
	}
	if entree.Faite == nil {
		s.erreur(w, r, &tache.ErrValidation{Champ: "faite", Raison: "obligatoire"})
		return
	}
	t, err := s.store.MarqueFaite(r.Context(), id, *entree.Faite)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, t)
}

func (s *Serveur) supprime(w http.ResponseWriter, r *http.Request) {
	id, err := identifiant(r)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	if err := s.store.Supprime(r.Context(), id); err != nil {
		s.erreur(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Serveur) sante(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		s.log.ErrorContext(r.Context(), "base indisponible", "erreur", err)
		httpkit.EcritJSON(w, http.StatusServiceUnavailable, map[string]string{"statut": "indisponible"})
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, map[string]string{"statut": "ok"})
}

func identifiant(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, &httpkit.ErrRequete{Raison: "identifiant invalide"}
	}
	return id, nil
}

// erreur traduit une erreur en réponse HTTP. Les erreurs inattendues sont
// journalisées, mais leur détail n'est jamais envoyé au client.
func (s *Serveur) erreur(w http.ResponseWriter, r *http.Request, err error) {
	statut, message := http.StatusInternalServerError, "erreur interne"
	if v, ok := errors.AsType[*tache.ErrValidation](err); ok {
		statut, message = http.StatusUnprocessableEntity, v.Error()
	} else if e, ok := errors.AsType[*httpkit.ErrRequete](err); ok {
		statut, message = http.StatusBadRequest, e.Error()
	} else if errors.Is(err, tache.ErrIntrouvable) {
		statut, message = http.StatusNotFound, err.Error()
	} else {
		s.log.ErrorContext(r.Context(), "erreur interne", "erreur", err)
	}
	httpkit.EcritJSON(w, statut, map[string]string{"erreur": message})
}
