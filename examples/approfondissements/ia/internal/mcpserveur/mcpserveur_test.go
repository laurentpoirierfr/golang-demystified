package mcpserveur_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/mcpserveur"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// connecte relie un client MCP au serveur en mémoire, sans réseau ni
// processus : le SDK fournit une paire de transports reliés.
func connecte(t *testing.T) (*mcp.ClientSession, *outils.Taches) {
	t.Helper()
	taches := &outils.Taches{}
	boite, err := outils.Boite(taches)
	if err != nil {
		t.Fatal(err)
	}
	coteServeur, coteClient := mcp.NewInMemoryTransports()
	serveur := mcpserveur.Nouveau(boite, "test")
	ss, err := serveur.Connect(t.Context(), coteServeur, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(t.Context(), coteClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, taches
}

func appelle(t *testing.T, cs *mcp.ClientSession, nom string, args any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: nom, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if txt, ok := c.(*mcp.TextContent); ok {
			b.WriteString(txt.Text)
		}
	}
	return b.String(), res.IsError
}

func TestListeDesOutils(t *testing.T) {
	cs, _ := connecte(t)
	var noms []string
	for outil, err := range cs.Tools(t.Context(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		noms = append(noms, outil.Name)
		if outil.Name == "ajouter_tache" {
			schema, _ := json.Marshal(outil.InputSchema)
			if !strings.Contains(string(schema), `"required":["titre"]`) {
				t.Errorf("schéma : %s", schema)
			}
		}
	}
	if strings.Join(noms, ",") != "ajouter_tache,lister_taches,terminer_tache" {
		t.Errorf("outils : %v", noms)
	}
}

func TestAppels(t *testing.T) {
	cs, taches := connecte(t)
	if r, erreur := appelle(t, cs, "ajouter_tache", map[string]any{"titre": "Préparer la démo MCP"}); erreur || r != "tâche 1 ajoutée : Préparer la démo MCP" {
		t.Errorf("ajout : %q %v", r, erreur)
	}
	if r, _ := appelle(t, cs, "terminer_tache", map[string]any{"id": 1}); !strings.Contains(r, "terminée") {
		t.Errorf("fin : %q", r)
	}
	if !taches.Liste(true)[0].Faite {
		t.Error("la tâche devrait être faite")
	}
	// Une erreur d'outil revient comme un résultat marqué IsError.
	if r, erreur := appelle(t, cs, "terminer_tache", map[string]any{"id": 42}); !erreur || !strings.Contains(r, "introuvable") {
		t.Errorf("erreur : %q %v", r, erreur)
	}
}
