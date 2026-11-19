package telemetrie

import (
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Route nomme le span d'après la route du http.ServeMux (« /taches/{id} »)
// et ajoute cette route aux métriques de otelhttp.
//
// otelhttp enveloppe le routeur : quand il crée le span, la route n'est pas
// encore connue. Ce middleware doit donc être placé après le routage, là où
// r.Pattern est renseigné. Il dépose la route dans les étiquettes de
// otelhttp, où NomDeSpan la retrouve pour nommer le span.
func Route(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Pattern != "" {
			// Le motif peut commencer par la méthode : « POST /taches ».
			route := r.Pattern
			if _, chemin, ok := strings.Cut(route, " "); ok {
				route = chemin
			}
			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + route)
			span.SetAttributes(semconv.HTTPRoute(route))
			if etiquettes, ok := otelhttp.LabelerFromContext(r.Context()); ok {
				etiquettes.Add(semconv.HTTPRoute(route))
			}
		}
		suivant.ServeHTTP(w, r)
	})
}

// NomDeSpan nomme le span d'une requête « MÉTHODE route »
// (« GET /taches/{id} ») si la route est connue, et « MÉTHODE » sinon, comme
// le recommandent les conventions d'OpenTelemetry. Nommer les spans d'après la
// route, et non d'après le chemin (« /taches/42 »), évite une infinité de
// noms différents. À passer à otelhttp.WithSpanNameFormatter.
func NomDeSpan(_ string, r *http.Request) string {
	if etiquettes, ok := otelhttp.LabelerFromContext(r.Context()); ok {
		for _, a := range etiquettes.Get() {
			if a.Key == semconv.HTTPRouteKey {
				return r.Method + " " + a.Value.AsString()
			}
		}
	}
	return r.Method
}
