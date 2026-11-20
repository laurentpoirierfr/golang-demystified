// Package mcpserveur expose les outils de tâches par le Model Context Protocol
// (MCP), le protocole ouvert qui relie les assistants (Claude Code, Claude
// Desktop, les éditeurs…) à des outils et des données. Un serveur MCP écrit
// une fois sert tous ces clients.
package mcpserveur

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// Nouveau crée le serveur MCP, avec un outil MCP par outil de la boîte. Le
// schéma de chaque outil est celui de la boîte, déduit des types Go.
func Nouveau(boite []outils.Outil, version string) *mcp.Server {
	serveur := mcp.NewServer(&mcp.Implementation{Name: "taches", Version: version}, nil)
	for _, o := range boite {
		serveur.AddTool(&mcp.Tool{Name: o.Nom, Description: o.Description, InputSchema: o.Schema},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				resultat, err := o.Execute(ctx, req.Params.Arguments)
				if err != nil {
					// Une erreur d'outil est un résultat, marqué comme tel : le
					// modèle la lit et peut se corriger. Une erreur Go
					// signalerait une panne du protocole.
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
					}, nil
				}
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: resultat}}}, nil
			})
	}
	return serveur
}
