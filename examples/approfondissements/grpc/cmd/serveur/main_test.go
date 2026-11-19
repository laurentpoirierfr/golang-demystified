package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/internal/service"
	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/grpc/pkg/rpc/taches/v1/tachesv1connect"
)

// demarre lance le vrai programme sur un port libre et renvoie son adresse.
func demarre(t *testing.T, env map[string]string) string {
	t.Helper()
	env["TACHES_ADRESSE"] = "127.0.0.1:0"
	ctx, arrete := context.WithCancel(context.Background())
	pret := make(chan string, 1)
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, func(k string) string { return env[k] }, io.Discard, pret) }()
	t.Cleanup(func() {
		arrete()
		if err := <-fini; err != nil {
			t.Errorf("arrêt : %v", err)
		}
	})
	select {
	case adresse := <-pret:
		return adresse
	case err := <-fini:
		t.Fatalf("démarrage : %v", err)
		return ""
	}
}

// h2c parle HTTP/2 en clair, comme l'exige gRPC sans TLS.
func h2c() *http.Client {
	protocoles := new(http.Protocols)
	protocoles.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: &http.Transport{Protocols: protocoles}}
}

// Le même client généré, sur les trois protocoles : seule l'option du
// transport change.
func TestTroisProtocoles(t *testing.T) {
	adresse := "http://" + demarre(t, map[string]string{})
	protocoles := map[string][]connecthttp.Option{
		"connect": nil,
		"grpc":    {connecthttp.WithGRPC()},
		"grpcweb": {connecthttp.WithGRPCWeb()},
	}
	for nom, options := range protocoles {
		t.Run(nom, func(t *testing.T) {
			c := tachesv1connect.NewTacheServiceClient(connect.NewClient(
				connecthttp.NewTransport(h2c(), adresse, options...)))

			cree, err := c.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "via " + nom})
			if err != nil {
				t.Fatal(err)
			}
			trouve, err := c.Trouve(t.Context(), &tachesv1.TrouveRequest{Id: cree.GetTache().GetId()})
			if err != nil {
				t.Fatal(err)
			}
			if trouve.GetTache().GetTitre() != "via "+nom {
				t.Errorf("titre = %q", trouve.GetTache().GetTitre())
			}

			_, err = c.Trouve(t.Context(), &tachesv1.TrouveRequest{Id: 999})
			if connect.CodeOf(err) != connect.CodeNotFound {
				t.Errorf("code = %v, attendu not_found", connect.CodeOf(err))
			}

			// Flux serveur : la première tâche reçue est la tâche n° 1.
			ctx, arrete := context.WithCancel(t.Context())
			defer arrete()
			flux, err := c.Suit(ctx, &tachesv1.SuitRequest{})
			if err != nil {
				t.Fatal(err)
			}
			defer flux.Close()
			rep, err := flux.Receive()
			if err != nil {
				t.Fatal(err)
			}
			if rep.GetTache().GetId() != 1 {
				t.Errorf("première tâche du flux : %v", rep.GetTache())
			}
		})
	}
}

// Le protocole Connect, c'est du HTTP ordinaire : un POST de JSON suffit, et
// une méthode sans effet de bord s'appelle même en GET.
func TestHTTPOrdinaire(t *testing.T) {
	adresse := "http://" + demarre(t, map[string]string{})

	rep, err := http.Post(adresse+"/taches.v1.TacheService/Cree", "application/json",
		strings.NewReader(`{"titre": "Sans client généré"}`))
	if err != nil {
		t.Fatal(err)
	}
	var cree struct {
		Tache struct {
			ID       string `json:"id"` // int64 est encodé en chaîne en JSON Protobuf
			Titre    string `json:"titre"`
			Creation string `json:"creation"` // horodatage RFC 3339
		} `json:"tache"`
	}
	err = json.NewDecoder(rep.Body).Decode(&cree)
	rep.Body.Close()
	if err != nil || rep.StatusCode != 200 || cree.Tache.ID != "1" || cree.Tache.Titre != "Sans client généré" {
		t.Fatalf("POST : statut %d, %+v, %v", rep.StatusCode, cree, err)
	}

	requete := url.Values{"encoding": {"json"}, "message": {`{"id": 1}`}}
	rep, err = http.Get(adresse + "/taches.v1.TacheService/Trouve?" + requete.Encode())
	if err != nil {
		t.Fatal(err)
	}
	corps, _ := io.ReadAll(rep.Body)
	rep.Body.Close()
	if rep.StatusCode != 200 || !strings.Contains(string(corps), `"titre":"Sans client généré"`) {
		t.Errorf("GET : statut %d, %s", rep.StatusCode, corps)
	}

	// Une erreur Connect est un JSON lisible, avec un statut HTTP cohérent.
	rep, err = http.Post(adresse+"/taches.v1.TacheService/Trouve", "application/json", strings.NewReader(`{"id": 42}`))
	if err != nil {
		t.Fatal(err)
	}
	corps, _ = io.ReadAll(rep.Body)
	rep.Body.Close()
	if rep.StatusCode != 404 || string(corps) != `{"code":"not_found","message":"tâche 42 introuvable"}` {
		t.Errorf("erreur : statut %d, %s", rep.StatusCode, corps)
	}
}

// Un client gRPC officiel (grpc-go) appelle le serveur Connect : le protocole
// gRPC est respecté à la lettre. Invoke appelle une méthode par son nom, avec
// les messages générés par protoc-gen-go, sans code généré pour grpc-go.
func TestClientGRPCGo(t *testing.T) {
	adresse := demarre(t, map[string]string{"TACHES_JETON": "secret"})
	conn, err := grpc.NewClient(adresse, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var rep tachesv1.CreeResponse
	err = conn.Invoke(t.Context(), tachesv1connect.TacheServiceCreeProcedure,
		&tachesv1.CreeRequest{Titre: "via grpc-go"}, &rep)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("sans jeton : %v", err)
	}

	ctx := metadata.AppendToOutgoingContext(t.Context(), "authorization", "Bearer secret")
	if err := conn.Invoke(ctx, tachesv1connect.TacheServiceCreeProcedure,
		&tachesv1.CreeRequest{Titre: "via grpc-go"}, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.GetTache().GetTitre() != "via grpc-go" {
		t.Errorf("réponse : %v", rep.GetTache())
	}
}

func TestAuthentification(t *testing.T) {
	adresse := "http://" + demarre(t, map[string]string{"TACHES_JETON": "secret"})
	transport := connecthttp.NewTransport(h2c(), adresse)

	anonyme := tachesv1connect.NewTacheServiceClient(connect.NewClient(transport))
	if _, err := anonyme.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "x"}); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("sans jeton : %v", err)
	}
	authentifie := tachesv1connect.NewTacheServiceClient(connect.NewClient(transport, service.Jeton("secret")))
	if _, err := authentifie.Cree(t.Context(), &tachesv1.CreeRequest{Titre: "x"}); err != nil {
		t.Errorf("avec jeton : %v", err)
	}
}
