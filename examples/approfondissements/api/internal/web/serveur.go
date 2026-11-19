// Package web expose les tâches à travers une API HTTP JSON. Son interface est
// définie par le contrat OpenAPI api/openapi.yaml : le routage, les types et
// l'interface ServerInterface sont générés par oapi-codegen (openapi.gen.go).
package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/api"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/httpkit"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/telemetrie"
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

	// creees compte les tâches créées : une métrique métier, sans effet tant
	// que Instrumente n'est pas appelée.
	creees metric.Int64Counter
}

// Instrumente active les métriques métier du serveur.
func (s *Serveur) Instrumente(mp metric.MeterProvider) error {
	mesures := mp.Meter("github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/web")
	creees, err := mesures.Int64Counter("taches.creees",
		metric.WithDescription("Nombre de tâches créées"),
		metric.WithUnit("{tache}"))
	if err != nil {
		return err
	}
	s.creees = creees
	return nil
}

// Le compilateur vérifie que Serveur implémente toutes les opérations du
// contrat : une opération ajoutée au contrat sans être codée ne compile pas.
var _ ServerInterface = (*Serveur)(nil)

// Nouveau crée le serveur.
func Nouveau(store Store, log *slog.Logger) *Serveur {
	creees, _ := noop.NewMeterProvider().Meter("").Int64Counter("") // ne peut pas échouer
	return &Serveur{store: store, log: log, creees: creees}
}

// Routes renvoie le handler complet : les opérations du contrat, validées
// par rapport à lui, le contrat lui-même et sa documentation interactive.
func (s *Serveur) Routes() (http.Handler, error) {
	contrat, err := GetSpec() // le contrat embarqué par oapi-codegen
	if err != nil {
		return nil, fmt.Errorf("contrat OpenAPI : %w", err)
	}
	valide := middleware.OapiRequestValidatorWithOptions(contrat, &middleware.Options{
		DoNotValidateServers: true, // le service peut être joint par n'importe quelle adresse
		ErrorHandler: func(w http.ResponseWriter, message string, statut int) {
			httpkit.EcritJSON(w, statut, Erreur{Erreur: message})
		},
	})

	mux := http.NewServeMux()
	HandlerWithOptions(s, StdHTTPServerOptions{
		BaseRouter: mux,
		// Après le routage : la route est enregistrée pour les traces et les
		// métriques, puis la requête est validée par le contrat.
		Middlewares: []MiddlewareFunc{valide, telemetrie.Route},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			s.erreur(w, r, &httpkit.ErrRequete{Raison: "paramètre invalide : " + err.Error()})
		},
	})
	mux.Handle("GET /openapi.yaml", httpkit.Fichier("application/yaml", api.Contrat))
	mux.Handle("GET /docs", httpkit.SwaggerUI("API des tâches", "/openapi.yaml"))

	// Le premier middleware de la liste est le plus extérieur.
	return httpkit.Chaine(mux,
		httpkit.Identifiant(),
		httpkit.Journal(s.log),
		httpkit.Recuperation(s.log),
	), nil
}

// ListeTaches implémente GET /taches.
func (s *Serveur) ListeTaches(w http.ResponseWriter, r *http.Request) {
	taches, err := s.store.Liste(r.Context())
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, taches)
}

// CreeTache implémente POST /taches.
func (s *Serveur) CreeTache(w http.ResponseWriter, r *http.Request) {
	var entree NouvelleTache // type généré depuis le contrat
	if err := httpkit.LitJSON(w, r, &entree); err != nil {
		s.erreur(w, r, err)
		return
	}
	titre, err := tache.ValideTitre(entree.Titre) // règle métier : pas de titre blanc
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	t, err := s.store.Cree(r.Context(), titre)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	s.creees.Add(r.Context(), 1)
	w.Header().Set("Location", fmt.Sprintf("/taches/%d", t.ID))
	httpkit.EcritJSON(w, http.StatusCreated, t)
}

// DetailTache implémente GET /taches/{id}. L'identifiant est déjà converti
// et validé (entier ≥ 1) par le code généré et le middleware de validation.
func (s *Serveur) DetailTache(w http.ResponseWriter, r *http.Request, id int64) {
	t, err := s.store.Trouve(r.Context(), id)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, t)
}

// ModifieTache implémente PATCH /taches/{id}.
func (s *Serveur) ModifieTache(w http.ResponseWriter, r *http.Request, id int64) {
	var entree ModificationTache // « faite » est obligatoire : vérifié par le contrat
	if err := httpkit.LitJSON(w, r, &entree); err != nil {
		s.erreur(w, r, err)
		return
	}
	t, err := s.store.MarqueFaite(r.Context(), id, entree.Faite)
	if err != nil {
		s.erreur(w, r, err)
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, t)
}

// SupprimeTache implémente DELETE /taches/{id}.
func (s *Serveur) SupprimeTache(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.Supprime(r.Context(), id); err != nil {
		s.erreur(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Sante implémente GET /sante.
func (s *Serveur) Sante(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		s.log.ErrorContext(r.Context(), "base indisponible", "erreur", err)
		httpkit.EcritJSON(w, http.StatusServiceUnavailable, Sante{Statut: Indisponible})
		return
	}
	httpkit.EcritJSON(w, http.StatusOK, Sante{Statut: Ok})
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
	httpkit.EcritJSON(w, statut, Erreur{Erreur: message})
}
