package inscription_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/idiomatique/internal/inscription"
)

// Des doublures écrites à la main : quelques lignes, sans bibliothèque de
// « mocks ». Elles satisfont les interfaces implicitement.
type depotMemoire struct {
	membres map[string]inscription.Membre
	panne   error
}

func (d *depotMemoire) Existe(_ context.Context, c string) (bool, error) {
	_, ok := d.membres[c]
	return ok, d.panne
}

func (d *depotMemoire) Enregistre(_ context.Context, m inscription.Membre) error {
	if d.panne != nil {
		return d.panne
	}
	d.membres[m.Courriel] = m
	return nil
}

type courrierEspion struct {
	envoyes []string
	panne   error
}

func (c *courrierEspion) Envoie(_ context.Context, a, sujet, _ string) error {
	c.envoyes = append(c.envoyes, a+" : "+sujet)
	return c.panne
}

var lundi = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

func service() (*inscription.Service, *depotMemoire, *courrierEspion) {
	d := &depotMemoire{membres: map[string]inscription.Membre{}}
	c := &courrierEspion{}
	return inscription.Nouveau(d, c, func() time.Time { return lundi }), d, c
}

func TestInscrit(t *testing.T) {
	s, d, c := service()
	m, err := s.Inscrit(t.Context(), "Alice <ALICE@exemple.org>", " Alice ")
	if err != nil {
		t.Fatal(err)
	}
	if m.Courriel != "alice@exemple.org" || m.Nom != "Alice" || !m.Le.Equal(lundi) {
		t.Errorf("membre : %+v", m)
	}
	if len(d.membres) != 1 || len(c.envoyes) != 1 || c.envoyes[0] != "alice@exemple.org : Bienvenue" {
		t.Errorf("dépôt %v, courriers %v", d.membres, c.envoyes)
	}

	// Une seconde inscription est refusée, et repérable avec errors.Is.
	if _, err := s.Inscrit(t.Context(), "alice@exemple.org", "Alice"); !errors.Is(err, inscription.ErrDejaInscrit) {
		t.Errorf("err = %v", err)
	}
}

func TestPannes(t *testing.T) {
	panne := errors.New("base indisponible")
	s, d, c := service()
	d.panne = panne
	// L'erreur d'origine reste repérable à travers l'enveloppe (%w).
	if _, err := s.Inscrit(t.Context(), "bob@exemple.org", "Bob"); !errors.Is(err, panne) {
		t.Errorf("err = %v", err)
	}
	if len(c.envoyes) != 0 {
		t.Error("aucun courriel ne doit partir si l'inscription échoue")
	}

	s, d, c = service()
	c.panne = errors.New("serveur de courrier injoignable")
	m, err := s.Inscrit(t.Context(), "bob@exemple.org", "Bob")
	if err == nil || m.Courriel == "" || len(d.membres) != 1 {
		t.Errorf("le membre doit être inscrit malgré le courrier : %+v, %v", m, err)
	}
}

func TestEntreesInvalides(t *testing.T) {
	s, _, _ := service()
	for _, cas := range [][2]string{{"pas-un-courriel", "Bob"}, {"bob@exemple.org", "  "}} {
		if _, err := s.Inscrit(t.Context(), cas[0], cas[1]); err == nil {
			t.Errorf("%q accepté", cas)
		}
	}
}
