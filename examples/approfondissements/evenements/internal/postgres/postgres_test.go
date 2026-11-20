package postgres_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/boutique"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/postgres"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

func ouvre(t *testing.T) (*postgres.Depot, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := t.Context()
	conteneur, err := tcpostgres.Run(ctx, "docker.io/postgres:17-alpine",
		tcpostgres.WithDatabase("boutique"),
		tcpostgres.WithUsername("gopher"),
		tcpostgres.WithPassword("secret"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	d, err := postgres.Ouvre(ctx, dsn, watermill.NewSlogLogger(slog.New(slog.DiscardHandler)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d, dsn
}

// enAttente compte les événements de l'outbox.
func enAttente(t *testing.T, d *postgres.Depot) int {
	t.Helper()
	n, err := d.EvenementsOutbox(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPostgresOutbox(t *testing.T) {
	d, _ := ouvre(t)
	ctx := t.Context()
	commande := boutique.Commande{ID: "c1", Client: "alice", Statut: boutique.EnAttente,
		Articles: []evenement.Article{{Ref: "clavier", Quantite: 1}}}

	// La transaction échoue après la publication : ni commande, ni événement.
	echec := errors.New("panne au milieu de la transaction")
	err := d.Execute(ctx, "", func(ctx context.Context, u boutique.Unite) error {
		if err := u.CreeCommande(ctx, commande); err != nil {
			return err
		}
		if err := u.Publie(ctx, &evenement.CommandePassee{CommandeID: "c1"}); err != nil {
			return err
		}
		return echec
	})
	if !errors.Is(err, echec) {
		t.Fatalf("err = %v", err)
	}
	if _, err := d.Commande(ctx, "c1"); !errors.Is(err, boutique.ErrIntrouvable) {
		t.Errorf("la commande ne devrait pas exister : %v", err)
	}
	if n := enAttente(t, d); n != 0 {
		t.Errorf("%d événements dans l'outbox, attendu 0", n)
	}

	// La transaction réussit : la commande et l'événement, ensemble.
	err = d.Execute(ctx, "", func(ctx context.Context, u boutique.Unite) error {
		if err := u.CreeCommande(ctx, commande); err != nil {
			return err
		}
		return u.Publie(ctx, &evenement.CommandePassee{CommandeID: "c1"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := enAttente(t, d); n != 1 {
		t.Errorf("%d événements dans l'outbox, attendu 1", n)
	}
}

func TestPostgresInbox(t *testing.T) {
	d, _ := ouvre(t)
	ctx := t.Context()
	if err := d.Approvisionne(ctx, "souris", 10); err != nil {
		t.Fatal(err)
	}
	reserve := func(ctx context.Context, u boutique.Unite) error {
		_, err := u.Reserve(ctx, []evenement.Article{{Ref: "souris", Quantite: 3}})
		return err
	}
	// Le même message, livré trois fois.
	for range 3 {
		if err := d.Execute(ctx, "stock.reserve/message-1", reserve); err != nil {
			t.Fatal(err)
		}
	}
	if n, _ := d.Stock(ctx, "souris"); n != 7 {
		t.Errorf("stock = %d, attendu 7 : le message a été traité plusieurs fois", n)
	}
}

func TestPostgresReservationToutOuRien(t *testing.T) {
	d, _ := ouvre(t)
	ctx := t.Context()
	d.Approvisionne(ctx, "souris", 10)
	d.Approvisionne(ctx, "clavier", 1)

	var manquant string
	err := d.Execute(ctx, "", func(ctx context.Context, u boutique.Unite) error {
		var err error
		manquant, err = u.Reserve(ctx, []evenement.Article{{Ref: "souris", Quantite: 2}, {Ref: "clavier", Quantite: 5}})
		return err
	})
	if err != nil || manquant != "clavier" {
		t.Fatalf("manquant = %q, err = %v", manquant, err)
	}
	// La souris, réservée avant que le clavier ne manque, a été rendue.
	if n, _ := d.Stock(ctx, "souris"); n != 10 {
		t.Errorf("stock de la souris = %d, attendu 10", n)
	}
}
