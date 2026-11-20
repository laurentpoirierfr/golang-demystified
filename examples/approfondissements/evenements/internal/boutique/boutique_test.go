package boutique_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/boutique"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/memoire"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/internal/messagerie"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/evenements/pkg/evenement"
)

// banc assemble la boutique sur le transport en mémoire de Watermill
// (GoChannel) : les mêmes gestionnaires, le même routeur et les mêmes
// middlewares qu'avec NATS, sans courtier.
type banc struct {
	service *boutique.Service
	depot   *memoire.Depot
	poison  <-chan *message.Message
}

type options struct {
	notifieur boutique.Notifieur
	// decore enveloppe le publieur : pour simuler un courtier qui livre
	// certains messages deux fois.
	decore func(message.Publisher) message.Publisher
}

func monte(t *testing.T, o options) *banc {
	t.Helper()
	log := slog.New(slog.DiscardHandler)
	wlog := watermill.NewSlogLogger(log)
	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 100}, wlog)

	var publieur message.Publisher = pubsub
	if o.decore != nil {
		publieur = o.decore(pubsub)
	}
	bus, err := messagerie.NouveauBus(publieur, wlog)
	if err != nil {
		t.Fatal(err)
	}
	depot := memoire.Nouveau(bus, map[string]int{"clavier": 5, "souris": 10})
	var n atomic.Int64
	service := boutique.Nouveau(depot, o.notifieur, func() string { return fmt.Sprintf("c%d", n.Add(1)) })

	routeur, err := messagerie.NouveauRouteur(pubsub, log)
	if err != nil {
		t.Fatal(err)
	}
	// Avec GoChannel, chaque abonnement reçoit tous les messages du sujet.
	abonne := func(string) (message.Subscriber, error) { return pubsub, nil }
	if err := messagerie.AjouteGestionnaires(routeur, abonne, wlog, service.Gestionnaires()...); err != nil {
		t.Fatal(err)
	}
	poison, err := pubsub.Subscribe(t.Context(), messagerie.SujetPoison)
	if err != nil {
		t.Fatal(err)
	}

	ctx, arrete := context.WithCancel(context.Background())
	fini := make(chan error, 1)
	go func() { fini <- routeur.Run(ctx) }()
	t.Cleanup(func() {
		arrete()
		if err := <-fini; err != nil {
			t.Errorf("routeur : %v", err)
		}
	})
	<-routeur.Running()
	return &banc{service: service, depot: depot, poison: poison}
}

// attend que la condition soit vraie : le traitement est asynchrone.
func attend(t *testing.T, description string, condition func() bool) {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(limite) {
			t.Fatalf("délai dépassé : %s", description)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (b *banc) statut(t *testing.T, id string) boutique.Statut {
	t.Helper()
	c, err := b.service.Commande(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return c.Statut
}

func (b *banc) stock(t *testing.T, ref string) int {
	t.Helper()
	n, _ := b.depot.Stock(t.Context(), ref)
	return n
}

// notifications enregistre les notifications, comme un fournisseur qui
// ignore les doublons d'après leur clé.
type notifications struct {
	mu      sync.Mutex
	appels  int
	envoyes map[string]string // clé → message
	echecs  int               // nombre d'échecs à simuler avant de réussir
}

func (n *notifications) Notifie(_ context.Context, cle, client, msg string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.appels++
	if n.echecs != 0 {
		if n.echecs > 0 {
			n.echecs--
		}
		return errors.New("service de courriel indisponible")
	}
	if n.envoyes == nil {
		n.envoyes = map[string]string{}
	}
	n.envoyes[cle] = client + " : " + msg
	return nil
}

func (n *notifications) liste() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	var l []string
	for _, m := range n.envoyes {
		l = append(l, m)
	}
	return l
}

func TestCommandeConfirmee(t *testing.T) {
	notifs := &notifications{}
	b := monte(t, options{notifieur: notifs})

	id, err := b.service.Passe(t.Context(), "alice", []evenement.Article{{Ref: "clavier", Quantite: 2}, {Ref: "souris", Quantite: 1}})
	if err != nil {
		t.Fatal(err)
	}
	// Juste après, la commande est en attente : la suite est asynchrone.
	attend(t, "confirmation", func() bool { return b.statut(t, id) == boutique.Confirmee })
	attend(t, "notification", func() bool { return len(notifs.liste()) == 1 })

	if b.stock(t, "clavier") != 3 || b.stock(t, "souris") != 9 {
		t.Errorf("stock : clavier %d, souris %d", b.stock(t, "clavier"), b.stock(t, "souris"))
	}
	if got := notifs.liste()[0]; got != "alice : Votre commande c1 est confirmée." {
		t.Errorf("notification : %q", got)
	}
}

func TestCompensation(t *testing.T) {
	notifs := &notifications{}
	b := monte(t, options{notifieur: notifs})

	// La souris est disponible, mais pas les 6 claviers : rien n'est réservé.
	id, err := b.service.Passe(t.Context(), "bob", []evenement.Article{{Ref: "souris", Quantite: 1}, {Ref: "clavier", Quantite: 6}})
	if err != nil {
		t.Fatal(err)
	}
	attend(t, "annulation", func() bool { return b.statut(t, id) == boutique.Annulee })
	c, _ := b.service.Commande(t.Context(), id)
	if c.Raison != "stock insuffisant : clavier" {
		t.Errorf("raison : %q", c.Raison)
	}
	if b.stock(t, "souris") != 10 {
		t.Errorf("la souris n'aurait pas dû être réservée : stock %d", b.stock(t, "souris"))
	}
	attend(t, "notification", func() bool { return len(notifs.liste()) == 1 })
}

// doublon publie chaque message deux fois, avec le même identifiant : c'est
// ce que fait un courtier « au moins une fois » après une coupure réseau.
type doublon struct{ message.Publisher }

func (d doublon) Publish(sujet string, msgs ...*message.Message) error {
	for _, m := range msgs {
		if err := d.Publisher.Publish(sujet, m, m.Copy()); err != nil {
			return err
		}
	}
	return nil
}

func TestIdempotence(t *testing.T) {
	notifs := &notifications{}
	b := monte(t, options{notifieur: notifs, decore: func(p message.Publisher) message.Publisher { return doublon{p} }})

	id, err := b.service.Passe(t.Context(), "alice", []evenement.Article{{Ref: "clavier", Quantite: 2}})
	if err != nil {
		t.Fatal(err)
	}
	attend(t, "confirmation", func() bool { return b.statut(t, id) == boutique.Confirmee })
	attend(t, "notification", func() bool { return len(notifs.liste()) == 1 })
	time.Sleep(200 * time.Millisecond) // laisse aux doublons le temps d'arriver

	// Chaque message est arrivé deux fois, mais le stock n'est réservé qu'une fois.
	if b.stock(t, "clavier") != 3 {
		t.Errorf("stock du clavier : %d, attendu 3", b.stock(t, "clavier"))
	}
	notifs.mu.Lock()
	defer notifs.mu.Unlock()
	if notifs.appels < 2 || len(notifs.envoyes) != 1 {
		t.Errorf("notifications : %d appels, %d envoyées", notifs.appels, len(notifs.envoyes))
	}
}

func TestNouvellesTentatives(t *testing.T) {
	notifs := &notifications{echecs: 2} // deux pannes, puis le service revient
	b := monte(t, options{notifieur: notifs})

	if _, err := b.service.Passe(t.Context(), "alice", []evenement.Article{{Ref: "souris", Quantite: 1}}); err != nil {
		t.Fatal(err)
	}
	attend(t, "notification", func() bool { return len(notifs.liste()) == 1 })
	notifs.mu.Lock()
	defer notifs.mu.Unlock()
	if notifs.appels != 3 {
		t.Errorf("%d appels, attendu 3", notifs.appels)
	}
}

func TestFilePoison(t *testing.T) {
	notifs := &notifications{echecs: -1} // toujours en panne
	b := monte(t, options{notifieur: notifs})

	id, err := b.service.Passe(t.Context(), "alice", []evenement.Article{{Ref: "souris", Quantite: 1}})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-b.poison:
		msg.Ack()
		if raison := msg.Metadata.Get(middleware.ReasonForPoisonedKey); raison != "service de courriel indisponible" {
			t.Errorf("raison : %q", raison)
		}
		if nom := msg.Metadata.Get("name"); nom != "StockReserve" {
			t.Errorf("événement : %q", nom)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("aucun message dans la file poison")
	}
	// L'échec de la notification ne bloque pas le reste de la saga.
	attend(t, "confirmation", func() bool { return b.statut(t, id) == boutique.Confirmee })
	notifs.mu.Lock()
	defer notifs.mu.Unlock()
	if notifs.appels != 4 { // une tentative, puis trois nouvelles
		t.Errorf("%d appels, attendu 4", notifs.appels)
	}
}

func TestValidation(t *testing.T) {
	b := monte(t, options{notifieur: &notifications{}})
	for _, articles := range [][]evenement.Article{nil, {{Ref: "", Quantite: 1}}, {{Ref: "souris", Quantite: 0}}} {
		if _, err := b.service.Passe(t.Context(), "alice", articles); err == nil {
			t.Errorf("%v : commande acceptée", articles)
		}
	}
}
