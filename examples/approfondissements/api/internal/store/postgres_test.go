package store

import (
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestPostgres exécute la suite de tests sur un vrai PostgreSQL, démarré
// dans un conteneur jetable par Testcontainers.
//
// Le test est ignoré avec -short, ou si aucun moteur de conteneurs (Docker,
// Podman) n'est disponible.
func TestPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := t.Context()
	conteneur, err := tcpostgres.Run(ctx, "docker.io/postgres:17-alpine",
		tcpostgres.WithDatabase("taches"),
		tcpostgres.WithUsername("gopher"),
		tcpostgres.WithPassword("secret"),
		tcpostgres.BasicWaitStrategies(), // attend que la base accepte les connexions
	)
	testcontainers.CleanupContainer(t, conteneur) // supprimé à la fin du test
	if err != nil {
		t.Fatal(err)
	}

	dsn, err := conteneur.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("PostgreSQL :", dsn)

	suite(t, ouvre(t, dsn))
}
