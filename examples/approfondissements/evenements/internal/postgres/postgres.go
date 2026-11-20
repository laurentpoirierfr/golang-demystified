// Package postgres implémente le dépôt de la boutique avec PostgreSQL, et les
// deux motifs qui rendent une architecture événementielle fiable :
//
//   - l'outbox : les événements sont écrits dans une table, dans la même
//     transaction que les données. Un transitaire (forwarder) les envoie
//     ensuite au courtier. Plus de « double écriture » : si la transaction
//     échoue, aucun événement ne part ; si elle réussit, l'événement partira,
//     même si le courtier est indisponible à ce moment-là ;
//   - l'inbox : la clé de chaque message traité est enregistrée, dans la même
//     transaction que ses effets. Un message relivré n'est pas traité deux fois.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wsql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/jackc/pgx/v5/stdlib" // pilote « pgx »

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/boutique"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/messagerie"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

// SujetOutbox est le sujet, donc la table, de l'outbox.
const SujetOutbox = "outbox"

const schema = `
CREATE TABLE IF NOT EXISTS commandes (
	id       TEXT PRIMARY KEY,
	client   TEXT NOT NULL,
	articles JSONB NOT NULL,
	statut   TEXT NOT NULL,
	raison   TEXT NOT NULL DEFAULT '',
	creation TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS stock (
	ref      TEXT PRIMARY KEY,
	quantite INTEGER NOT NULL CHECK (quantite >= 0)
);
-- L'inbox : un message, identifié par sa clé, n'est traité qu'une fois.
CREATE TABLE IF NOT EXISTS messages_traites (
	cle TEXT PRIMARY KEY,
	le  TIMESTAMPTZ NOT NULL DEFAULT now()
);`

// Depot est le dépôt PostgreSQL.
type Depot struct {
	db  *sql.DB
	log watermill.LoggerAdapter
}

// Ouvre se connecte, crée les tables, et prépare la table de l'outbox.
func Ouvre(ctx context.Context, dsn string, log watermill.LoggerAdapter) (*Depot, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}
	d := &Depot{db: db, log: log}
	// La table de l'outbox doit exister avant la première publication, qui a
	// lieu dans une transaction (où Watermill refuse de créer des tables).
	abonne, err := d.AbonneOutbox()
	if err != nil {
		db.Close()
		return nil, err
	}
	defer abonne.Close()
	if err := abonne.SubscribeInitialize(SujetOutbox); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// Close ferme la connexion.
func (d *Depot) Close() error { return d.db.Close() }

// AbonneOutbox lit la table de l'outbox, pour le transitaire.
func (d *Depot) AbonneOutbox() (*wsql.Subscriber, error) {
	return wsql.NewSubscriber(wsql.BeginnerFromStdSQL(d.db), wsql.SubscriberConfig{
		ConsumerGroup:    "transitaire",
		SchemaAdapter:    wsql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   wsql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
		PollInterval:     100 * time.Millisecond,
	}, d.log)
}

// Transitaire envoie les événements de l'outbox vers le courtier.
func (d *Depot) Transitaire(courtier message.Publisher) (*forwarder.Forwarder, error) {
	abonne, err := d.AbonneOutbox()
	if err != nil {
		return nil, err
	}
	return forwarder.NewForwarder(abonne, courtier, d.log, forwarder.Config{ForwarderTopic: SujetOutbox})
}

// Approvisionne crée un article avec son stock initial, s'il n'existe pas encore :
// un redémarrage du service ne doit pas remettre le stock à zéro.
func (d *Depot) Approvisionne(ctx context.Context, ref string, quantite int) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO stock (ref, quantite) VALUES ($1, $2)
		ON CONFLICT (ref) DO NOTHING`, ref, quantite)
	return err
}

// Execute exécute f dans une transaction.
func (d *Depot) Execute(ctx context.Context, cle string, f func(context.Context, boutique.Unite) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if cle != "" {
		// Si deux livraisons du même message arrivent en même temps, la
		// seconde attend la fin de la première sur la clé primaire, puis ne
		// fait rien.
		res, err := tx.ExecContext(ctx, `INSERT INTO messages_traites (cle) VALUES ($1) ON CONFLICT DO NOTHING`, cle)
		if err != nil {
			return errors.Join(err, tx.Rollback())
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			return errors.Join(err, tx.Rollback()) // déjà traité
		}
	}

	// Les événements sont publiés dans la table de l'outbox, au sein de la
	// transaction, enveloppés avec leur sujet de destination.
	dansTx, err := wsql.NewPublisher(wsql.TxFromStdSQL(tx), wsql.PublisherConfig{
		SchemaAdapter: wsql.DefaultPostgreSQLSchema{},
	}, d.log)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}
	bus, err := messagerie.NouveauBus(forwarder.NewPublisher(dansTx, forwarder.PublisherConfig{ForwarderTopic: SujetOutbox}), d.log)
	if err != nil {
		return errors.Join(err, tx.Rollback())
	}

	if err := f(ctx, &unite{tx: tx, bus: bus}); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// Commande renvoie une commande.
func (d *Depot) Commande(ctx context.Context, id string) (boutique.Commande, error) {
	var c boutique.Commande
	var articles []byte
	err := d.db.QueryRowContext(ctx, `SELECT id, client, articles, statut, raison FROM commandes WHERE id = $1`, id).
		Scan(&c.ID, &c.Client, &articles, &c.Statut, &c.Raison)
	if errors.Is(err, sql.ErrNoRows) {
		return c, boutique.ErrIntrouvable
	}
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(articles, &c.Articles)
}

// Stock renvoie le stock d'un article.
func (d *Depot) Stock(ctx context.Context, ref string) (int, error) {
	var n int
	err := d.db.QueryRowContext(ctx, `SELECT quantite FROM stock WHERE ref = $1`, ref).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

type unite struct {
	tx  *sql.Tx
	bus interface {
		Publish(ctx context.Context, evt any) error
	}
}

func (u *unite) CreeCommande(ctx context.Context, c boutique.Commande) error {
	articles, err := json.Marshal(c.Articles)
	if err != nil {
		return err
	}
	_, err = u.tx.ExecContext(ctx, `INSERT INTO commandes (id, client, articles, statut) VALUES ($1, $2, $3, $4)`,
		c.ID, c.Client, articles, c.Statut)
	return err
}

func (u *unite) ChangeStatut(ctx context.Context, id string, s boutique.Statut, raison string) error {
	res, err := u.tx.ExecContext(ctx, `UPDATE commandes SET statut = $2, raison = $3 WHERE id = $1`, id, s, raison)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return errors.Join(err, boutique.ErrIntrouvable)
	}
	return nil
}

// Reserve décrémente le stock de chaque article. Un point de sauvegarde
// (SAVEPOINT) permet d'annuler les premières réservations si un article
// manque, sans annuler toute la transaction : l'événement StockInsuffisant
// doit, lui, être publié.
func (u *unite) Reserve(ctx context.Context, articles []evenement.Article) (string, error) {
	if _, err := u.tx.ExecContext(ctx, `SAVEPOINT reservation`); err != nil {
		return "", err
	}
	for _, a := range articles {
		// La condition « quantite >= $2 » rend la vérification et la
		// décrémentation atomiques : pas de survente entre deux commandes
		// concurrentes.
		res, err := u.tx.ExecContext(ctx,
			`UPDATE stock SET quantite = quantite - $2 WHERE ref = $1 AND quantite >= $2`, a.Ref, a.Quantite)
		if err != nil {
			return "", err
		}
		if n, err := res.RowsAffected(); err != nil {
			return "", err
		} else if n == 0 {
			_, err := u.tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT reservation`)
			return a.Ref, err
		}
	}
	_, err := u.tx.ExecContext(ctx, `RELEASE SAVEPOINT reservation`)
	return "", err
}

func (u *unite) Publie(ctx context.Context, evt any) error { return u.bus.Publish(ctx, evt) }

// EvenementsOutbox compte les événements écrits dans l'outbox (pour les
// tests et la supervision).
func (d *Depot) EvenementsOutbox(ctx context.Context) (int, error) {
	var n int
	err := d.db.QueryRowContext(ctx, `SELECT count(*) FROM "watermill_`+SujetOutbox+`"`).Scan(&n)
	return n, err
}
