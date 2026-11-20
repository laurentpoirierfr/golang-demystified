// Package sse implémente les Server-Sent Events (SSE), un flux d'événements
// du serveur vers le navigateur, sur une simple réponse HTTP. Le format est
// défini par la norme HTML (« text/event-stream ») :
//
//	id: 42
//	event: message
//	data: {"texte":"bonjour"}
//
// Chaque événement se termine par une ligne vide. Le navigateur (EventSource)
// se reconnecte seul en cas de coupure, en renvoyant le dernier identifiant
// reçu dans l'en-tête Last-Event-ID : le serveur peut alors renvoyer ce qui a
// été manqué.
package sse

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Evenement est un événement du flux.
type Evenement struct {
	ID      string        // identifiant, renvoyé par le client à la reconnexion
	Type    string        // type d'événement (« message » par défaut)
	Donnees string        // contenu, sur une ou plusieurs lignes
	Reessai time.Duration // délai de reconnexion conseillé au client
}

// ErrSautDeLigne signale un identifiant ou un type contenant un saut de ligne :
// il permettrait d'injecter des champs ou des événements dans le flux.
var ErrSautDeLigne = errors.New("sse : saut de ligne interdit dans l'identifiant ou le type")

// Flux écrit des événements sur une réponse HTTP.
type Flux struct {
	w  http.ResponseWriter
	rc *http.ResponseController
}

// Ouvre commence le flux : en-têtes, statut 200, puis envoi immédiat, pour
// que le client sache que la connexion est établie.
func Ouvre(w http.ResponseWriter) (*Flux, error) {
	rc := http.NewResponseController(w)
	// Un flux dure longtemps : on lève le délai d'écriture du serveur
	// (http.Server.WriteTimeout) pour cette réponse seulement.
	if err := rc.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return nil, err
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // demande à nginx de ne pas mettre en tampon
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		return nil, fmt.Errorf("sse : la réponse ne peut pas être vidée : %w", err)
	}
	return &Flux{w: w, rc: rc}, nil
}

// Envoie écrit un événement, et le pousse immédiatement vers le client.
func (f *Flux) Envoie(e Evenement) error {
	if strings.ContainsAny(e.ID, "\r\n") || strings.ContainsAny(e.Type, "\r\n") {
		return ErrSautDeLigne
	}
	var b strings.Builder
	if e.ID != "" {
		b.WriteString("id: " + e.ID + "\n")
	}
	if e.Type != "" {
		b.WriteString("event: " + e.Type + "\n")
	}
	if e.Reessai > 0 {
		b.WriteString("retry: " + strconv.FormatInt(e.Reessai.Milliseconds(), 10) + "\n")
	}
	// Une donnée sur plusieurs lignes devient plusieurs lignes « data: » :
	// un saut de ligne ne peut pas terminer l'événement prématurément.
	donnees := strings.ReplaceAll(e.Donnees, "\r\n", "\n")
	for ligne := range strings.SplitSeq(donnees, "\n") {
		b.WriteString("data: " + ligne + "\n")
	}
	b.WriteString("\n")
	return f.ecrit(b.String())
}

// Commentaire écrit une ligne ignorée par le client. Envoyée régulièrement,
// elle maintient la connexion ouverte à travers les proxys qui coupent les
// connexions inactives, et révèle au serveur un client parti.
func (f *Flux) Commentaire(texte string) error {
	return f.ecrit(": " + strings.ReplaceAll(texte, "\n", " ") + "\n\n")
}

func (f *Flux) ecrit(s string) error {
	if _, err := f.w.Write([]byte(s)); err != nil {
		return err
	}
	return f.rc.Flush()
}
