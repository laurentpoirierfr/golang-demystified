package main

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestArretAvecConnexionsOuvertes arrête le serveur alors qu'un flux SSE et un
// WebSocket sont ouverts : l'arrêt doit être rapide, et les clients prévenus.
func TestArretAvecConnexionsOuvertes(t *testing.T) {
	ctx, arrete := context.WithCancel(context.Background())
	defer arrete()
	pret := make(chan string, 1)
	fini := make(chan error, 1)
	env := map[string]string{"SALONS_ADRESSE": "127.0.0.1:0"}
	go func() { fini <- run(ctx, func(k string) string { return env[k] }, io.Discard, pret, 0) }()
	adresse := <-pret

	ws, _, err := websocket.Dial(t.Context(), "ws://"+adresse+"/salons/golang/ws?pseudo=alice", nil) //nolint:bodyclose // Dial gère le corps de la réponse
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	rep, err := http.Get("http://" + adresse + "/salons/golang/flux") //nolint:bodyclose // fermé par le defer ci-dessous
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Body.Close()
	fluxFini := make(chan struct{})
	go func() { io.Copy(io.Discard, rep.Body); close(fluxFini) }()

	debut := time.Now()
	arrete()
	if err := <-fini; err != nil {
		t.Fatalf("arrêt : %v", err)
	}
	if duree := time.Since(debut); duree > time.Second {
		t.Errorf("arrêt en %v : les connexions longues ont bloqué Shutdown", duree)
	}

	// Le client WebSocket reçoit une fermeture en règle : « going away ».
	_, _, err = ws.Read(t.Context())
	if websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Errorf("fermeture WebSocket : %v", err)
	}
	select {
	case <-fluxFini:
	case <-time.After(time.Second):
		t.Error("le flux SSE ne s'est pas terminé")
	}
}
