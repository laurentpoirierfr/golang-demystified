package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/adaptateurs/memoire"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/application"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/internal/domaine"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/horloge"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/identifiant"
)

// notifieurEspion enregistre les notifications au lieu de les envoyer.
type notifieurEspion struct {
	messages []string
	echec    error
}

func (n *notifieurEspion) Notifie(_ context.Context, m domaine.Membre, msg string) error {
	if n.echec != nil {
		return n.echec
	}
	n.messages = append(n.messages, m.Nom+" : "+msg)
	return nil
}

type banc struct {
	service   *application.Service
	horloge   *horloge.Manuelle
	notifieur *notifieurEspion
	depot     *memoire.Depot
}

func nouveauBanc() banc {
	depot := memoire.Nouveau(
		[]domaine.Livre{
			{ISBN: "go-1", Titre: "Le langage Go"},
			{ISBN: "go-2", Titre: "Concurrence en Go"},
			{ISBN: "go-3", Titre: "Go et le Web"},
			{ISBN: "go-4", Titre: "Génériques"},
		},
		[]domaine.Membre{{ID: "alice", Nom: "Alice"}, {ID: "bob", Nom: "Bob"}},
	)
	b := banc{
		horloge:   horloge.NouvelleManuelle(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)),
		notifieur: &notifieurEspion{},
		depot:     depot,
	}
	b.service = application.NouveauService(application.Dependances{
		Catalogue: depot, Emprunts: depot, Transactions: depot,
		Notifieur: b.notifieur, Horloge: b.horloge, Identifiants: &identifiant.Sequence{Prefixe: "e"},
		Journal: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return b
}

func TestEmprunteEtRend(t *testing.T) {
	b := nouveauBanc()
	ctx := t.Context()

	e, err := b.service.Emprunte(ctx, "alice", "go-1")
	if err != nil {
		t.Fatal(err)
	}
	if want := b.horloge.Maintenant().Add(21 * 24 * time.Hour); !e.Echeance.Equal(want) {
		t.Errorf("échéance = %v, attendu %v", e.Echeance, want)
	}

	if _, err := b.service.Emprunte(ctx, "bob", "go-1"); !errors.Is(err, domaine.ErrLivreIndisponible) {
		t.Errorf("second emprunt : %v, attendu ErrLivreIndisponible", err)
	}

	b.horloge.Avance(10 * 24 * time.Hour)
	penalite, err := b.service.Rend(ctx, e.ID)
	if err != nil || penalite != 0 {
		t.Fatalf("Rend = %d, %v", penalite, err)
	}
	if _, err := b.service.Emprunte(ctx, "bob", "go-1"); err != nil {
		t.Errorf("le livre rendu devrait être disponible : %v", err)
	}
}

func TestQuota(t *testing.T) {
	b := nouveauBanc()
	for _, isbn := range []string{"go-1", "go-2", "go-3"} {
		if _, err := b.service.Emprunte(t.Context(), "alice", isbn); err != nil {
			t.Fatal(err)
		}
	}
	_, err := b.service.Emprunte(t.Context(), "alice", "go-4")
	if !errors.Is(err, domaine.ErrQuotaAtteint) {
		t.Errorf("erreur = %v, attendu ErrQuotaAtteint", err)
	}
	if err.Error() != "emprunt de go-4 par alice : 3 emprunts au plus" {
		t.Errorf("message = %q", err)
	}
}

func TestRetardPenaliteEtSuspension(t *testing.T) {
	b := nouveauBanc()
	ctx := t.Context()
	e, _ := b.service.Emprunte(ctx, "alice", "go-1")

	b.horloge.Avance(21*24*time.Hour + 2*24*time.Hour + time.Hour) // 2 jours et 1 heure de retard
	penalite, err := b.service.Rend(ctx, e.ID)
	if err != nil || penalite != 60 {
		t.Fatalf("Rend = %d, %v ; attendu 60 centimes", penalite, err)
	}
	if _, err := b.service.Emprunte(ctx, "alice", "go-2"); !errors.Is(err, domaine.ErrMembreSuspendu) {
		t.Errorf("erreur = %v, attendu ErrMembreSuspendu", err)
	}
}

func TestRelanceRetards(t *testing.T) {
	b := nouveauBanc()
	ctx := t.Context()
	b.service.Emprunte(ctx, "alice", "go-1")
	b.horloge.Avance(5 * 24 * time.Hour)
	b.service.Emprunte(ctx, "bob", "go-2")

	b.horloge.Avance(17 * 24 * time.Hour) // alice est en retard, bob pas encore
	n, err := b.service.RelanceRetards(ctx)
	if err != nil || n != 1 {
		t.Fatalf("RelanceRetards = %d, %v", n, err)
	}
	want := "Alice : « Le langage Go » devait être rendu le 22/09/2026."
	if len(b.notifieur.messages) != 1 || b.notifieur.messages[0] != want {
		t.Errorf("messages = %q", b.notifieur.messages)
	}

	b.notifieur.echec = errors.New("serveur SMTP indisponible")
	if n, err := b.service.RelanceRetards(ctx); err != nil || n != 0 {
		t.Errorf("avec échec d'envoi : %d, %v ; attendu 0, nil", n, err)
	}
}

func TestIntrouvable(t *testing.T) {
	b := nouveauBanc()
	if _, err := b.service.Emprunte(t.Context(), "inconnu", "go-1"); !errors.Is(err, domaine.ErrIntrouvable) {
		t.Errorf("membre inconnu : %v", err)
	}
	if _, err := b.service.Rend(t.Context(), "e42"); !errors.Is(err, domaine.ErrIntrouvable) {
		t.Errorf("emprunt inconnu : %v", err)
	}
}
