package messagerie

import (
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
)

// PublieurNATS publie sur NATS JetStream : les messages sont conservés par le
// serveur jusqu'à leur acquittement, et survivent au redémarrage des
// consommateurs.
func PublieurNATS(url string, log watermill.LoggerAdapter) (message.Publisher, error) {
	return wnats.NewPublisher(wnats.PublisherConfig{
		URL:       url,
		Marshaler: &wnats.NATSMarshaler{}, // métadonnées en en-têtes NATS
		JetStream: wnats.JetStreamConfig{
			AutoProvision: true, // crée le flux s'il n'existe pas
			// L'identifiant du message sert à JetStream pour écarter les
			// doublons publiés dans sa fenêtre de déduplication.
			TrackMsgID: true,
		},
	}, log)
}

// AbonneNATS crée, pour chaque gestionnaire, un consommateur durable
// JetStream : le serveur retient où en est chaque gestionnaire, et plusieurs
// instances du service se partagent les messages (groupe de consommateurs).
func AbonneNATS(url string, log watermill.LoggerAdapter) Abonne {
	return func(gestionnaire string) (message.Subscriber, error) {
		// « stock.reserve » → « stock-reserve » : le point est interdit dans
		// les noms de consommateurs.
		nom := strings.ReplaceAll(gestionnaire, ".", "-")
		return wnats.NewSubscriber(wnats.SubscriberConfig{
			URL:              url,
			QueueGroupPrefix: nom,
			SubscribersCount: 1,
			AckWaitTimeout:   30 * time.Second, // sans acquittement, le message est relivré
			CloseTimeout:     10 * time.Second,
			Unmarshaler:      &wnats.NATSMarshaler{},
			JetStream: wnats.JetStreamConfig{
				AutoProvision: true,
				DurablePrefix: nom,
			},
		}, log)
	}
}
