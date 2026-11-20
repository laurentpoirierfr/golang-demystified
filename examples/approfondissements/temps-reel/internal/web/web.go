// Package web expose les salons en temps réel : WebSocket pour discuter,
// Server-Sent Events pour suivre, et une page HTML qui utilise les deux.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/salon"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/pkg/sse"
)

//go:embed statique
var statique embed.FS

// TailleMaxMessage est la taille maximale d'un message WebSocket reçu : au-delà,
// la connexion est fermée (code 1009, « message too big »).
const TailleMaxMessage = 4 << 10

var nomValide = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

// Options configure le serveur.
type Options struct {
	// Origines autorisées pour les WebSockets, en plus de l'hôte du serveur
	// (« app.example.com »). Voir la section sur le détournement de session.
	Origines []string
	// Pouls est l'intervalle des signaux de vie (ping WebSocket, commentaire
	// SSE).
	Pouls time.Duration
}

// Serveur porte les dépendances des handlers.
type Serveur struct {
	hub  *salon.Hub
	log  *slog.Logger
	opts Options

	// arret est annulé à l'arrêt du serveur : les connexions longues
	// (WebSocket, SSE) se ferment alors proprement.
	arret  context.Context
	Arrete context.CancelFunc
}

// Nouveau crée le serveur.
func Nouveau(hub *salon.Hub, log *slog.Logger, opts Options) *Serveur {
	if opts.Pouls == 0 {
		opts.Pouls = 30 * time.Second
	}
	arret, arrete := context.WithCancel(context.Background())
	return &Serveur{hub: hub, log: log, opts: opts, arret: arret, Arrete: arrete}
}

// Routes assemble les routes.
func (s *Serveur) Routes() http.Handler {
	mux := http.NewServeMux()
	page, _ := fs.Sub(statique, "statique")
	mux.Handle("GET /", http.FileServerFS(page))
	mux.HandleFunc("GET /salons/{salon}/ws", s.websocket)
	mux.HandleFunc("GET /salons/{salon}/flux", s.flux)
	mux.HandleFunc("POST /salons/{salon}/messages", s.publie)
	return mux
}

// entrant est un message envoyé par un client WebSocket.
type entrant struct {
	Texte string `json:"texte"`
}

// sortant est un message envoyé à un client WebSocket.
type sortant struct {
	Type    string         `json:"type"` // « message » ou « erreur »
	Message *salon.Message `json:"message,omitempty"`
	Erreur  string         `json:"erreur,omitempty"`
}

func (s *Serveur) websocket(w http.ResponseWriter, r *http.Request) {
	nom, pseudo := r.PathValue("salon"), r.URL.Query().Get("pseudo")
	if !nomValide.MatchString(nom) || !nomValide.MatchString(pseudo) {
		http.Error(w, "salon ou pseudo invalide", http.StatusBadRequest)
		return
	}
	// Accept vérifie l'en-tête Origin : une page d'un autre site ne peut pas
	// ouvrir de WebSocket avec les cookies de l'utilisateur.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: s.opts.Origines})
	if err != nil {
		s.log.WarnContext(r.Context(), "WebSocket refusé", "erreur", err)
		return // Accept a déjà répondu (403 pour une origine refusée)
	}
	defer func() { _ = c.CloseNow() }() // sans effet si Close a déjà fermé la connexion
	c.SetReadLimit(TailleMaxMessage)

	// La connexion se termine quand le client part, quand la lecture échoue,
	// ou quand le serveur s'arrête.
	ctx, termine := context.WithCancel(r.Context())
	defer termine()
	defer context.AfterFunc(s.arret, termine)()

	abonnement, _ := s.hub.Abonne(nom, 0)
	defer s.hub.Desabonne(abonnement)
	s.log.InfoContext(ctx, "WebSocket ouvert", "salon", nom, "pseudo", pseudo)

	// Lecture, dans sa propre goroutine : une connexion WebSocket est
	// bidirectionnelle, et lecture et écriture ne doivent pas s'attendre.
	//
	// Attention : annuler le contexte d'un Read ferme brutalement la
	// connexion. La lecture n'utilise donc pas ctx, annulé à l'arrêt du
	// serveur : la boucle principale doit d'abord envoyer la trame de
	// fermeture « going away ». La lecture se termine avec la connexion.
	go func() {
		defer termine()
		for {
			var e entrant
			if err := wsjson.Read(r.Context(), c, &e); err != nil {
				return // client parti, message trop gros, JSON invalide…
			}
			if _, err := s.hub.Publie(nom, pseudo, e.Texte); err != nil {
				// L'écriture est sûre depuis plusieurs goroutines.
				_ = s.ecrit(ctx, c, sortant{Type: "erreur", Erreur: err.Error()})
			}
		}
	}()

	pouls := time.NewTicker(s.opts.Pouls)
	defer pouls.Stop()
	for {
		select {
		case m, ok := <-abonnement.Messages():
			if !ok {
				c.Close(websocket.StatusTryAgainLater, "client trop lent")
				return
			}
			if err := s.ecrit(ctx, c, sortant{Type: "message", Message: &m}); err != nil {
				return
			}
		case <-pouls.C:
			// Le ping vérifie que le client répond toujours : une connexion
			// coupée sans fermeture (Wi-Fi perdu) serait sinon gardée à vie.
			ping, annule := context.WithTimeout(ctx, s.opts.Pouls/2)
			err := c.Ping(ping)
			annule()
			if err != nil {
				return
			}
		case <-ctx.Done():
			if s.arret.Err() != nil {
				c.Close(websocket.StatusGoingAway, "arrêt du serveur")
			}
			return
		}
	}
}

// ecrit envoie un message, avec un délai : un client qui ne lit plus ne doit
// pas bloquer la goroutine indéfiniment.
func (s *Serveur) ecrit(ctx context.Context, c *websocket.Conn, v sortant) error {
	ctx, annule := context.WithTimeout(ctx, 5*time.Second)
	defer annule()
	return wsjson.Write(ctx, c, v)
}

func (s *Serveur) flux(w http.ResponseWriter, r *http.Request) {
	nom := r.PathValue("salon")
	if !nomValide.MatchString(nom) {
		http.Error(w, "salon invalide", http.StatusBadRequest)
		return
	}
	// À la reconnexion, EventSource renvoie le dernier identifiant reçu.
	depuis, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)

	flux, err := sse.Ouvre(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	abonnement, manques := s.hub.Abonne(nom, depuis)
	defer s.hub.Desabonne(abonnement)

	envoie := func(m salon.Message) error {
		donnees, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return flux.Envoie(sse.Evenement{ID: strconv.FormatInt(m.ID, 10), Type: "message", Donnees: string(donnees)})
	}
	// Le premier événement indique le délai de reconnexion conseillé.
	if err := flux.Envoie(sse.Evenement{Type: "bienvenue", Donnees: nom, Reessai: 2 * time.Second}); err != nil {
		return
	}
	for _, m := range manques {
		if err := envoie(m); err != nil {
			return
		}
	}

	pouls := time.NewTicker(s.opts.Pouls)
	defer pouls.Stop()
	for {
		select {
		case m, ok := <-abonnement.Messages():
			if !ok {
				return // trop lent : le client se reconnectera, et rattrapera
			}
			if err := envoie(m); err != nil {
				return
			}
		case <-pouls.C:
			if err := flux.Commentaire("pouls"); err != nil {
				return // client parti
			}
		case <-r.Context().Done():
			return
		case <-s.arret.Done():
			return
		}
	}
}

func (s *Serveur) publie(w http.ResponseWriter, r *http.Request) {
	nom := r.PathValue("salon")
	var req struct {
		Auteur string `json:"auteur"`
		Texte  string `json:"texte"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, TailleMaxMessage)).Decode(&req); err != nil ||
		!nomValide.MatchString(nom) || !nomValide.MatchString(req.Auteur) {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	m, err := s.hub.Publie(nom, req.Auteur, req.Texte)
	if errors.Is(err, salon.ErrVide) || errors.Is(err, salon.ErrTropLong) {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(m)
}
