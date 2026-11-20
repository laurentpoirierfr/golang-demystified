// Package salon diffuse les messages des salons de discussion à leurs
// abonnés. Il ignore tout du transport : un abonné peut être une connexion
// WebSocket, un flux SSE ou un test.
package salon

import (
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Message est un message d'un salon.
type Message struct {
	ID     int64     `json:"id"` // croissant : sert de Last-Event-ID
	Salon  string    `json:"salon"`
	Auteur string    `json:"auteur"`
	Texte  string    `json:"texte"`
	Le     time.Time `json:"le"`
}

const (
	// TailleHistorique est le nombre de messages conservés, pour les clients
	// qui se reconnectent après une coupure.
	TailleHistorique = 100
	// TailleTampon est le retard qu'un abonné peut prendre avant d'être
	// déconnecté.
	TailleTampon = 32
	// LongueurMax est la longueur maximale d'un message, en caractères.
	LongueurMax = 500
)

// Erreurs de validation.
var (
	ErrVide     = errors.New("message vide")
	ErrTropLong = errors.New("message trop long")
)

// Hub relie les auteurs et les abonnés.
type Hub struct {
	Maintenant func() time.Time

	mu         sync.Mutex
	dernierID  int64
	historique []Message // les TailleHistorique derniers messages, tous salons
	abonnes    map[*Abonnement]struct{}
}

// Nouveau crée un hub.
func Nouveau() *Hub {
	return &Hub{Maintenant: time.Now, abonnes: map[*Abonnement]struct{}{}}
}

// Abonnement reçoit les messages d'un salon.
type Abonnement struct {
	salon    string
	messages chan Message
}

// Messages renvoie le canal des messages. Il est fermé quand l'abonnement se
// termine : désabonnement, ou abonné trop lent.
func (a *Abonnement) Messages() <-chan Message { return a.messages }

// Abonne inscrit un abonné au salon, et renvoie les messages publiés après
// depuis (0 : aucun), qu'il a manqués. L'historique et l'inscription sont
// pris sous le même verrou : aucun message ne peut être manqué ni reçu deux
// fois.
func (h *Hub) Abonne(salon string, depuis int64) (*Abonnement, []Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var manques []Message
	if depuis > 0 {
		for _, m := range h.historique {
			if m.Salon == salon && m.ID > depuis {
				manques = append(manques, m)
			}
		}
	}
	a := &Abonnement{salon: salon, messages: make(chan Message, TailleTampon)}
	h.abonnes[a] = struct{}{}
	return a, manques
}

// Desabonne termine l'abonnement. Il peut être appelé plusieurs fois.
func (h *Hub) Desabonne(a *Abonnement) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.retire(a)
}

func (h *Hub) retire(a *Abonnement) {
	if _, ok := h.abonnes[a]; ok {
		delete(h.abonnes, a)
		close(a.messages)
	}
}

// Publie valide et diffuse un message.
func (h *Hub) Publie(salon, auteur, texte string) (Message, error) {
	texte = strings.TrimSpace(texte)
	switch {
	case texte == "":
		return Message{}, ErrVide
	case utf8.RuneCountInString(texte) > LongueurMax:
		return Message{}, ErrTropLong
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dernierID++
	m := Message{ID: h.dernierID, Salon: salon, Auteur: auteur, Texte: texte, Le: h.Maintenant().UTC()}
	h.historique = append(h.historique, m)
	if len(h.historique) > TailleHistorique {
		h.historique = h.historique[1:]
	}
	for a := range h.abonnes {
		if a.salon != salon {
			continue
		}
		select {
		case a.messages <- m:
		default:
			// Un abonné trop lent ne doit ralentir ni les autres, ni
			// l'auteur : il est déconnecté, et se reconnectera.
			h.retire(a)
		}
	}
	return m, nil
}

// Abonnes renvoie le nombre d'abonnés (pour la supervision et les tests).
func (h *Hub) Abonnes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.abonnes)
}
