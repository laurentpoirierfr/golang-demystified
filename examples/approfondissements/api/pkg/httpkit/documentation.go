package httpkit

import (
	"fmt"
	"html"
	"net/http"
)

// Fichier sert un contenu fixe, par exemple un contrat OpenAPI embarqué.
func Fichier(typeMIME string, contenu []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", typeMIME)
		w.Write(contenu)
	})
}

// VersionSwaggerUI est la version de Swagger UI chargée depuis un CDN.
const VersionSwaggerUI = "5.17.14"

// SwaggerUI sert une page de documentation interactive pour le contrat
// OpenAPI disponible à l'adresse urlContrat. La page charge Swagger UI depuis
// un CDN : aucune dépendance Go n'est nécessaire.
func SwaggerUI(titre, urlContrat string) http.Handler {
	page := fmt.Sprintf(`<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <title>%[1]s</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@%[3]s/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@%[3]s/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({ url: %[2]q, dom_id: "#swagger-ui" });
  </script>
</body>
</html>
`, html.EscapeString(titre), urlContrat, VersionSwaggerUI)
	return Fichier("text/html; charset=utf-8", []byte(page))
}
