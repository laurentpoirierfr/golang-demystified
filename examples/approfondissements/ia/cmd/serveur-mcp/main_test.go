package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestStdio lance le vrai programme, comme le ferait Claude Code, et lui
// parle par son entrée et sa sortie standard.
func TestStdio(t *testing.T) {
	if testing.Short() {
		t.Skip("compile et lance le programme")
	}
	binaire := filepath.Join(t.TempDir(), "serveur-mcp")
	if out, err := exec.Command("go", "build", "-o", binaire, ".").CombinedOutput(); err != nil {
		t.Fatalf("compilation : %v\n%s", err, out)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	commande := exec.Command(binaire)
	commande.Stderr = os.Stderr // le journal du serveur, hors du protocole
	session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: commande}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	if session.InitializeResult().ServerInfo.Name != "taches" {
		t.Errorf("serveur : %+v", session.InitializeResult().ServerInfo)
	}
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "ajouter_tache", Arguments: map[string]any{"titre": "Brancher Claude Code"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if txt := res.Content[0].(*mcp.TextContent).Text; !strings.Contains(txt, "tâche 1 ajoutée") {
		t.Errorf("résultat : %q", txt)
	}
}

// TestHTTP sert le transport Streamable HTTP, et deux clients successifs
// partagent la même liste de tâches.
func TestHTTP(t *testing.T) {
	ecoute, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	adresse := ecoute.Addr().String()
	ecoute.Close() // run écoute lui-même sur cette adresse
	ctx, arrete := context.WithCancel(t.Context())
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, adresse) }()
	t.Cleanup(func() {
		arrete()
		if err := <-fini; err != nil {
			t.Errorf("arrêt : %v", err)
		}
	})

	appelle := func(nom string, args map[string]any) string {
		t.Helper()
		var session *mcp.ClientSession
		var err error
		for range 50 { // le serveur démarre
			client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
			session, err = client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: "http://" + adresse}, nil)
			if err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: nom, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return res.Content[0].(*mcp.TextContent).Text
	}
	appelle("ajouter_tache", map[string]any{"titre": "Par HTTP"})
	if r := appelle("lister_taches", map[string]any{}); r != "1. Par HTTP (à faire)\n" {
		t.Errorf("liste : %q", r)
	}
}
