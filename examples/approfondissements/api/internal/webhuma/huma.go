// Package webhuma expose la même API des tâches que le package web, mais avec
// Huma et l'approche « code d'abord » : le contrat OpenAPI 3.1 est déduit des
// types Go des entrées et des sorties de chaque opération.
package webhuma

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/tache"
)

// Store est ce dont l'API a besoin pour stocker les tâches.
type Store interface {
	Liste(ctx context.Context) ([]tache.Tache, error)
	Trouve(ctx context.Context, id int64) (tache.Tache, error)
	Cree(ctx context.Context, titre string) (tache.Tache, error)
	MarqueFaite(ctx context.Context, id int64, faite bool) (tache.Tache, error)
	Supprime(ctx context.Context, id int64) error
	Ping(ctx context.Context) error
}

// Tache est la représentation d'une tâche dans l'API. Les balises doc et
// example alimentent le contrat OpenAPI.
type Tache struct {
	ID       int64     `json:"id" doc:"Identifiant de la tâche" example:"1"`
	Titre    string    `json:"titre" doc:"Titre de la tâche" example:"Relire le livre"`
	Faite    bool      `json:"faite" doc:"La tâche est-elle faite ?"`
	Creation time.Time `json:"creation" doc:"Date de création"`
}

func depuis(t tache.Tache) Tache {
	return Tache{ID: t.ID, Titre: t.Titre, Faite: t.Faite, Creation: t.Creation}
}

// Les entrées et sorties des opérations. Huma lit les balises pour savoir
// d'où vient chaque champ (chemin, en-tête, corps) et comment le valider.
type (
	ListeSortie struct {
		Body []Tache
	}

	CreeEntree struct {
		Body struct {
			Titre string `json:"titre" minLength:"1" maxLength:"200" doc:"Titre de la tâche" example:"Relire le livre"`
		}
	}

	TacheSortie struct {
		Location string `header:"Location" doc:"Adresse de la tâche"`
		Body     Tache
	}

	IDEntree struct {
		ID int64 `path:"id" minimum:"1" doc:"Identifiant de la tâche"`
	}

	ModifieEntree struct {
		ID   int64 `path:"id" minimum:"1" doc:"Identifiant de la tâche"`
		Body struct {
			Faite bool `json:"faite" doc:"Nouvel état de la tâche"` // obligatoire : pas d'omitempty
		}
	}

	SanteSortie struct {
		Body struct {
			Statut string `json:"statut" enum:"ok,indisponible"`
		}
	}
)

// Enregistre déclare les opérations auprès de l'API Huma.
func Enregistre(api huma.API, s Store) {
	huma.Register(api, huma.Operation{
		OperationID: "liste-taches", Method: http.MethodGet, Path: "/taches",
		Summary: "Liste les tâches", Tags: []string{"taches"},
	}, func(ctx context.Context, _ *struct{}) (*ListeSortie, error) {
		taches, err := s.Liste(ctx)
		if err != nil {
			return nil, erreur(err)
		}
		sortie := &ListeSortie{Body: make([]Tache, 0, len(taches))}
		for _, t := range taches {
			sortie.Body = append(sortie.Body, depuis(t))
		}
		return sortie, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "cree-tache", Method: http.MethodPost, Path: "/taches",
		Summary: "Crée une tâche", Tags: []string{"taches"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, e *CreeEntree) (*TacheSortie, error) {
		titre, err := tache.ValideTitre(e.Body.Titre) // règle métier : pas de titre blanc
		if err != nil {
			return nil, erreur(err)
		}
		t, err := s.Cree(ctx, titre)
		if err != nil {
			return nil, erreur(err)
		}
		return &TacheSortie{Location: fmt.Sprintf("/taches/%d", t.ID), Body: depuis(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "detail-tache", Method: http.MethodGet, Path: "/taches/{id}",
		Summary: "Détail d'une tâche", Tags: []string{"taches"},
	}, func(ctx context.Context, e *IDEntree) (*TacheSortie, error) {
		t, err := s.Trouve(ctx, e.ID)
		if err != nil {
			return nil, erreur(err)
		}
		return &TacheSortie{Location: fmt.Sprintf("/taches/%d", t.ID), Body: depuis(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "modifie-tache", Method: http.MethodPatch, Path: "/taches/{id}",
		Summary: "Marque une tâche comme faite, ou non", Tags: []string{"taches"},
	}, func(ctx context.Context, e *ModifieEntree) (*TacheSortie, error) {
		t, err := s.MarqueFaite(ctx, e.ID, e.Body.Faite)
		if err != nil {
			return nil, erreur(err)
		}
		return &TacheSortie{Location: fmt.Sprintf("/taches/%d", t.ID), Body: depuis(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "supprime-tache", Method: http.MethodDelete, Path: "/taches/{id}",
		Summary: "Supprime une tâche", Tags: []string{"taches"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, e *IDEntree) (*struct{}, error) {
		return nil, erreur(s.Supprime(ctx, e.ID))
	})

	huma.Register(api, huma.Operation{
		OperationID: "sante", Method: http.MethodGet, Path: "/sante",
		Summary: "Contrôle de santé", Tags: []string{"exploitation"},
	}, func(ctx context.Context, _ *struct{}) (*SanteSortie, error) {
		if err := s.Ping(ctx); err != nil {
			return nil, huma.Error503ServiceUnavailable("base indisponible")
		}
		sortie := &SanteSortie{}
		sortie.Body.Statut = "ok"
		return sortie, nil
	})
}

// erreur traduit les erreurs du domaine en erreurs HTTP de Huma, au format
// « problem+json » de la RFC 9457. Les autres deviennent des erreurs 500.
func erreur(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, tache.ErrIntrouvable):
		return huma.Error404NotFound(err.Error())
	}
	if v, ok := errors.AsType[*tache.ErrValidation](err); ok {
		return huma.Error422UnprocessableEntity(v.Error())
	}
	return err
}

// Routes crée le handler complet. Huma sert aussi le contrat
// (/openapi.json, /openapi.yaml) et la documentation interactive (/docs).
func Routes(s Store) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("API des tâches (Huma)", "1.0.0")
	config.Info.Description = "La même API que celle générée par oapi-codegen, écrite avec Huma."
	api := humago.New(mux, config)
	Enregistre(api, s)
	return mux
}
