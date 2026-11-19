// Package api contient le contrat OpenAPI du service, source de vérité de
// son interface HTTP.
package api

import _ "embed"

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
//go:generate go tool oapi-codegen -config oapi-codegen-client.yaml openapi.yaml

// Contrat est le contrat OpenAPI, tel qu'il est servi aux clients.
//
//go:embed openapi.yaml
var Contrat []byte
