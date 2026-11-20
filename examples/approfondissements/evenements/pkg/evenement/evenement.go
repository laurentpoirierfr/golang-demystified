// Package evenement définit les événements échangés par les services de la
// boutique. C'est le contrat entre les services : on peut lui ajouter des
// champs, jamais en retirer ni en changer le sens, car des consommateurs
// écrits pour l'ancienne version liront encore les nouveaux messages.
package evenement

import "time"

// Article est une ligne de commande.
type Article struct {
	Ref      string `json:"ref"`
	Quantite int    `json:"quantite"`
}

// CommandePassee : un client vient de passer une commande. Elle attend la
// réservation du stock.
type CommandePassee struct {
	CommandeID string    `json:"commande_id"`
	Client     string    `json:"client"`
	Articles   []Article `json:"articles"`
	Le         time.Time `json:"le"`
}

// StockReserve : le stock de tous les articles de la commande est réservé.
type StockReserve struct {
	CommandeID string `json:"commande_id"`
}

// StockInsuffisant : un article manque, rien n'a été réservé.
type StockInsuffisant struct {
	CommandeID string `json:"commande_id"`
	Ref        string `json:"ref"` // l'article manquant
}
