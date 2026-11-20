package web_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/salon"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/pkg/sse"
)

type message struct {
	Type    string         `json:"type"`
	Message *salon.Message `json:"message"`
	Erreur  string         `json:"erreur"`
}

// demarre lance le serveur avec un http.Server configuré comme en production.
func demarre(t *testing.T, opts web.Options, delaiEcriture time.Duration) (string, *web.Serveur) {
	t.Helper()
	s := web.Nouveau(salon.Nouveau(), slog.New(slog.DiscardHandler), opts)
	srv := httptest.NewUnstartedServer(s.Routes())
	srv.Config.WriteTimeout = delaiEcriture
	srv.Config.RegisterOnShutdown(s.Arrete)
	srv.Start()
	t.Cleanup(srv.Close)
	return srv.URL, s
}

func connecte(t *testing.T, base, salon, pseudo string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(base, "http") + "/salons/" + salon + "/ws?pseudo=" + pseudo
	c, _, err := websocket.Dial(t.Context(), url, nil) //nolint:bodyclose // Dial gère le corps de la réponse
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return c
}

func lit(t *testing.T, c *websocket.Conn) message {
	t.Helper()
	ctx, annule := context.WithTimeout(t.Context(), 5*time.Second)
	defer annule()
	var m message
	if err := wsjson.Read(ctx, c, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// ouvreFlux ouvre un flux SSE et renvoie un itérateur sur ses événements.
func ouvreFlux(t *testing.T, base, salon string, depuis int64) func() sse.Evenement {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), "GET", base+"/salons/"+salon+"/flux", nil)
	if depuis > 0 {
		req.Header.Set("Last-Event-ID", strconv.FormatInt(depuis, 10))
	}
	rep, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rep.Body.Close() })
	if rep.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Content-Type : %q", rep.Header.Get("Content-Type"))
	}
	evenements := make(chan sse.Evenement)
	go func() {
		defer close(evenements)
		for e, err := range sse.Lit(rep.Body) {
			if err != nil {
				return
			}
			evenements <- e
		}
	}()
	return func() sse.Evenement {
		t.Helper()
		select {
		case e, ok := <-evenements:
			if !ok {
				t.Fatal("flux terminé")
			}
			return e
		case <-time.After(5 * time.Second):
			t.Fatal("aucun événement")
			return sse.Evenement{}
		}
	}
}

func TestDiscussion(t *testing.T) {
	base, _ := demarre(t, web.Options{}, 0)
	alice := connecte(t, base, "golang", "alice")
	bob := connecte(t, base, "golang", "bob")
	flux := ouvreFlux(t, base, "golang", 0)
	if e := flux(); e.Type != "bienvenue" || e.Reessai != 2*time.Second {
		t.Errorf("premier événement : %+v", e)
	}

	if err := wsjson.Write(t.Context(), alice, map[string]string{"texte": "Bonjour <b>Bob</b>"}); err != nil {
		t.Fatal(err)
	}
	// Les deux clients WebSocket et le flux SSE reçoivent le message.
	for _, c := range []*websocket.Conn{alice, bob} {
		if m := lit(t, c); m.Type != "message" || m.Message.Auteur != "alice" || m.Message.Texte != "Bonjour <b>Bob</b>" {
			t.Errorf("WebSocket : %+v", m)
		}
	}
	if e := flux(); e.Type != "message" || e.ID != "1" || !strings.Contains(e.Donnees, `"auteur":"alice"`) {
		t.Errorf("SSE : %+v", e)
	}

	// Un message invalide : une erreur, pour l'auteur seulement.
	wsjson.Write(t.Context(), bob, map[string]string{"texte": "   "})
	if m := lit(t, bob); m.Type != "erreur" || m.Erreur != "message vide" {
		t.Errorf("erreur : %+v", m)
	}
}

func TestRepriseSSE(t *testing.T) {
	base, _ := demarre(t, web.Options{}, 0)
	for _, texte := range []string{"un", "deux", "trois"} {
		rep, err := http.Post(base+"/salons/golang/messages", "application/json",
			strings.NewReader(`{"auteur": "alice", "texte": "`+texte+`"}`))
		if err != nil || rep.StatusCode != http.StatusCreated {
			t.Fatalf("publication : %v %v", rep, err)
		}
		rep.Body.Close()
	}
	// Le client a reçu le message 1, puis a perdu la connexion : à la
	// reconnexion, Last-Event-ID lui fait rattraper 2 et 3, et rien d'autre.
	flux := ouvreFlux(t, base, "golang", 1)
	flux() // bienvenue
	for _, attendu := range []string{"2", "3"} {
		if e := flux(); e.ID != attendu {
			t.Errorf("ID = %q, attendu %q", e.ID, attendu)
		}
	}
}

func TestOrigineRefusee(t *testing.T) {
	base, _ := demarre(t, web.Options{Origines: []string{"app.example.com"}}, 0)
	url := "ws" + strings.TrimPrefix(base, "http") + "/salons/golang/ws?pseudo=mallory"
	for origine, accepte := range map[string]bool{
		"https://pirate.example":  false,
		"https://app.example.com": true,
	} {
		c, rep, err := websocket.Dial(t.Context(), url, &websocket.DialOptions{ //nolint:bodyclose // Dial gère le corps de la réponse
			HTTPHeader: http.Header{"Origin": {origine}},
		})
		if accepte {
			if err != nil {
				t.Errorf("%s : refusée : %v", origine, err)
				continue
			}
			c.CloseNow()
		} else if err == nil || rep.StatusCode != http.StatusForbidden {
			t.Errorf("%s : acceptée (%v)", origine, err)
		}
	}
}

func TestMessageTropGros(t *testing.T) {
	base, _ := demarre(t, web.Options{}, 0)
	c := connecte(t, base, "golang", "alice")
	gros := strings.Repeat("a", web.TailleMaxMessage+1)
	c.Write(t.Context(), websocket.MessageText, []byte(`{"texte":"`+gros+`"}`))
	_, _, err := c.Read(t.Context())
	if websocket.CloseStatus(err) != websocket.StatusMessageTooBig {
		t.Errorf("fermeture : %v", err)
	}
}

func TestPouls(t *testing.T) {
	base, _ := demarre(t, web.Options{Pouls: 50 * time.Millisecond}, 0)
	req, _ := http.NewRequestWithContext(t.Context(), "GET", base+"/salons/golang/flux", nil)
	rep, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Body.Close()
	tampon := make([]byte, 4096)
	var lu strings.Builder
	limite := time.Now().Add(2 * time.Second)
	for !strings.Contains(lu.String(), ": pouls") && time.Now().Before(limite) {
		n, err := rep.Body.Read(tampon)
		lu.Write(tampon[:n])
		if err == io.EOF {
			break
		}
	}
	if !strings.Contains(lu.String(), ": pouls\n\n") {
		t.Errorf("pas de signal de vie : %q", lu.String())
	}
}

// Le serveur a un WriteTimeout court : les connexions longues doivent
// survivre au-delà.
func TestDelaiEcriture(t *testing.T) {
	const delai = 300 * time.Millisecond
	base, _ := demarre(t, web.Options{}, delai)
	ws := connecte(t, base, "golang", "alice")
	flux := ouvreFlux(t, base, "golang", 0)
	flux() // bienvenue

	time.Sleep(3 * delai)

	if err := wsjson.Write(t.Context(), ws, map[string]string{"texte": "toujours là ?"}); err != nil {
		t.Fatalf("écriture WebSocket : %v", err)
	}
	if m := lit(t, ws); m.Type != "message" {
		t.Errorf("WebSocket après le délai : %+v", m)
	}
	if e := flux(); e.Type != "message" {
		t.Errorf("SSE après le délai : %+v", e)
	}
}
