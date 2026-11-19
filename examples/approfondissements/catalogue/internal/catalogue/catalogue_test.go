package catalogue

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/catalogue/internal/db"
)

// basePostgres démarre un PostgreSQL jetable, et renvoie sa chaîne de
// connexion. Le test est ignoré avec -short, ou sans moteur de conteneurs.
func basePostgres(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("test d'intégration ignoré en mode -short")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	conteneur, err := tcpostgres.Run(t.Context(), "docker.io/postgres:17-alpine",
		tcpostgres.WithDatabase("catalogue"),
		tcpostgres.WithUsername("gopher"),
		tcpostgres.WithPassword("secret"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, conteneur)
	if err != nil {
		t.Fatal(err)
	}
	return conteneur.MustConnectionString(t.Context(), "sslmode=disable")
}

func TestPostgresCatalogue(t *testing.T) {
	dsn := basePostgres(t)
	ctx := t.Context()

	version, err := db.Migre(dsn)
	if err != nil || version != 2 {
		t.Fatalf("Migre = %d, %v ; attendu la version 2", version, err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	c := Nouveau(pool)

	livres := []NouveauLivre{
		{ISBN: "978-0-13-419044-0", Titre: "The Go Programming Language", Auteur: "Alan Donovan", Annee: new(int32(2015)), Exemplaires: 2},
		{ISBN: "978-1-49-195003-3", Titre: "Concurrency in Go", Auteur: "Katherine Cox-Buday", Annee: new(int32(2017)), Exemplaires: 1},
		{ISBN: "978-0-00-000000-1", Titre: "Go en pratique", Auteur: "Alan Donovan"}, // année inconnue
	}
	for _, l := range livres {
		if _, err := c.Ajoute(ctx, l); err != nil {
			t.Fatalf("Ajoute(%s) : %v", l.Titre, err)
		}
	}

	t.Run("ISBN unique, sans effet partiel", func(t *testing.T) {
		_, err := c.Ajoute(ctx, NouveauLivre{ISBN: livres[0].ISBN, Titre: "Doublon", Auteur: "Nouvel auteur"})
		if !errors.Is(err, ErrISBNDejaUtilise) {
			t.Fatalf("erreur = %v, attendu ErrISBNDejaUtilise", err)
		}
		// La transaction a été annulée : « Nouvel auteur » n'existe pas.
		stats, _ := c.Requetes().StatistiquesParAuteur(ctx)
		if len(stats) != 2 {
			t.Errorf("auteurs = %+v, attendu 2 (la transaction aurait dû être annulée)", stats)
		}
	})

	t.Run("jointure", func(t *testing.T) {
		lignes, err := c.Requetes().LivresAvecAuteur(ctx)
		if err != nil || len(lignes) != 3 {
			t.Fatalf("LivresAvecAuteur = %d, %v", len(lignes), err)
		}
		if lignes[0].Auteur != "Alan Donovan" || lignes[0].Livre.Annee != nil {
			t.Errorf("première ligne inattendue : %+v", lignes[0])
		}
	})

	t.Run("recherche avec paramètre facultatif", func(t *testing.T) {
		tous, _ := c.Recherche(ctx, "go", nil)
		recents, _ := c.Recherche(ctx, "go", new(int32(2016)))
		if len(tous) != 3 || len(recents) != 1 || recents[0].Titre != "Concurrency in Go" {
			t.Errorf("recherche : %d puis %d résultat(s)", len(tous), len(recents))
		}
	})

	t.Run("stock", func(t *testing.T) {
		isbn := livres[1].ISBN // un seul exemplaire
		if err := c.Emprunte(ctx, isbn); err != nil {
			t.Fatal(err)
		}
		if err := c.Emprunte(ctx, isbn); !errors.Is(err, ErrPlusDExemplaire) {
			t.Errorf("second emprunt : %v, attendu ErrPlusDExemplaire", err)
		}
		if err := c.Emprunte(ctx, "inconnu"); !errors.Is(err, ErrIntrouvable) {
			t.Errorf("livre inconnu : %v, attendu ErrIntrouvable", err)
		}
	})
}

// TestPostgresMigrationsReversibles applique toutes les migrations, les annule
// une à une, puis les réapplique : chaque « down » doit défaire son « up ».
func TestPostgresMigrationsReversibles(t *testing.T) {
	dsn := basePostgres(t)
	m, err := db.NouveauMigrateur(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		t.Fatalf("up : %v", err)
	}
	if err := m.Down(); err != nil {
		t.Fatalf("down : %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up après down : %v", err)
	}
	if v, sale, _ := m.Version(); v != 2 || sale {
		t.Errorf("version = %d (sale : %v), attendu 2", v, sale)
	}
}
