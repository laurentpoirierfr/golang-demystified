// Package messagerie assemble Watermill : nommage des sujets, encodage des
// événements, routeur et middlewares. Le transport (mémoire, NATS…) est fourni
// par l'appelant, sous la forme d'un publieur et d'un constructeur d'abonnés.
package messagerie

import (
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// SujetPoison reçoit les messages qu'aucune nouvelle tentative n'a pu
// traiter : ils y attendent une intervention, au lieu de bloquer la file ou
// d'être perdus.
const SujetPoison = "boutique-poison"

// Encodage : les événements voyagent en JSON, nommés d'après leur type
// (« CommandePassee »).
var Encodage = cqrs.JSONMarshaler{GenerateName: cqrs.StructName}

// Sujet renvoie le sujet d'un événement : « boutique-CommandePassee ». Pas de
// point : avec JetStream, le sujet sert aussi de nom de flux, où le point est
// interdit.
func Sujet(nomEvenement string) string { return "boutique-" + nomEvenement }

// NouveauBus crée le bus qui publie les événements sur publieur.
func NouveauBus(publieur message.Publisher, log watermill.LoggerAdapter) (*cqrs.EventBus, error) {
	return cqrs.NewEventBusWithConfig(publieur, cqrs.EventBusConfig{
		GeneratePublishTopic: func(p cqrs.GenerateEventPublishTopicParams) (string, error) {
			return Sujet(p.EventName), nil
		},
		Marshaler: Encodage,
		Logger:    log,
	})
}

// Abonne crée l'abonné d'un gestionnaire. Chaque gestionnaire a son propre
// abonnement : tous reçoivent chaque événement, et chacun le traite à son
// rythme.
type Abonne func(gestionnaire string) (message.Subscriber, error)

// NouveauRouteur crée le routeur, avec les middlewares, dans cet ordre :
//
//   - PoisonQueue : après l'échec de toutes les tentatives, le message part
//     dans SujetPoison, et le traitement continue ;
//   - Retry : nouvelles tentatives, avec un délai croissant ;
//   - Recoverer : une panique devient une erreur, donc une nouvelle tentative.
func NouveauRouteur(poison message.Publisher, log *slog.Logger) (*message.Router, error) {
	wlog := watermill.NewSlogLogger(log)
	routeur, err := message.NewRouter(message.RouterConfig{CloseTimeout: 10 * time.Second}, wlog)
	if err != nil {
		return nil, err
	}
	filePoison, err := middleware.PoisonQueue(poison, SujetPoison)
	if err != nil {
		return nil, err
	}
	routeur.AddMiddleware(
		filePoison,
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: 50 * time.Millisecond,
			Multiplier:      2,
			Logger:          wlog,
		}.Middleware,
		middleware.Recoverer,
	)
	return routeur, nil
}

// AjouteGestionnaires branche les gestionnaires sur le routeur.
func AjouteGestionnaires(routeur *message.Router, abonne Abonne, log watermill.LoggerAdapter, gestionnaires ...cqrs.EventHandler) error {
	processeur, err := cqrs.NewEventProcessorWithConfig(routeur, cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(p cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			return Sujet(p.EventName), nil
		},
		SubscriberConstructor: func(p cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return abonne(p.HandlerName)
		},
		Marshaler: Encodage,
		Logger:    log,
	})
	if err != nil {
		return err
	}
	return processeur.AddHandlers(gestionnaires...)
}
