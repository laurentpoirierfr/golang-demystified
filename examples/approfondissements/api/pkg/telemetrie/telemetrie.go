// Package telemetrie prépare l'observabilité d'un service avec OpenTelemetry :
// traces (exportées en OTLP, vers Jaeger, Tempo ou un collecteur), métriques
// (exposées au format Prometheus) et journaux corrélés aux traces. Il ne
// dépend d'aucun métier : il est placé dans pkg/ pour être réutilisé.
package telemetrie

import (
	"context"
	"errors"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Options configure la télémétrie.
type Options struct {
	Service string // nom du service, visible dans Jaeger et Prometheus
	Version string

	// Exportateur reçoit les traces. S'il est nil, les traces sont créées
	// (pour la corrélation des journaux) mais ne sont envoyées nulle part.
	Exportateur sdktrace.SpanExporter

	// Synchrone exporte chaque trace dès sa fin, au lieu de les regrouper
	// par lots : utile dans les tests, à éviter en production.
	Synchrone bool
}

// Telemetrie regroupe les fournisseurs de traces et de métriques.
type Telemetrie struct {
	Traces    trace.TracerProvider
	Metriques metric.MeterProvider

	// Prometheus sert les métriques au format Prometheus, sur /metrics.
	Prometheus http.Handler

	arrets []func(context.Context) error
}

// Demarre crée les fournisseurs. Arrete doit être appelée à la fin du
// programme, pour envoyer les dernières traces.
func Demarre(o Options) (*Telemetrie, error) {
	// La ressource décrit le service : chaque trace et chaque métrique en porte
	// les attributs.
	res := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName(o.Service),
		semconv.ServiceVersion(o.Version),
	)

	// Traces.
	optsTraces := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if o.Exportateur != nil {
		if o.Synchrone {
			optsTraces = append(optsTraces, sdktrace.WithSyncer(o.Exportateur))
		} else {
			optsTraces = append(optsTraces, sdktrace.WithBatcher(o.Exportateur))
		}
	}
	traces := sdktrace.NewTracerProvider(optsTraces...)

	// Métriques : un registre Prometheus propre au service, plutôt que le
	// registre global, pour ne pas mélanger les métriques de plusieurs tests.
	registre := prometheus.NewRegistry()
	exportateur, err := otelprom.New(otelprom.WithRegisterer(registre))
	if err != nil {
		return nil, err
	}
	metriques := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(exportateur))

	// Propagation du contexte de trace entre services : en-têtes W3C
	// « traceparent » et « baggage ».
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	return &Telemetrie{
		Traces:     traces,
		Metriques:  metriques,
		Prometheus: promhttp.HandlerFor(registre, promhttp.HandlerOpts{}),
		arrets:     []func(context.Context) error{traces.Shutdown, metriques.Shutdown},
	}, nil
}

// Arrete envoie les données en attente et libère les ressources.
func (t *Telemetrie) Arrete(ctx context.Context) error {
	var errs []error
	for _, arret := range t.arrets {
		errs = append(errs, arret(ctx))
	}
	return errors.Join(errs...)
}

// Handler expose les métriques sur /metrics et instrumente api avec otelhttp :
// un span et des métriques pour chaque requête.
//
// Il n'utilise pas de http.ServeMux : un routeur placé devant otelhttp lui
// imposerait sa propre route (« / ») pour nommer les spans et étiqueter les
// métriques. La vraie route est fournie, après le routage de l'API, par le
// middleware Route.
func (t *Telemetrie) Handler(nom string, api http.Handler) http.Handler {
	instrumente := otelhttp.NewHandler(api, nom,
		otelhttp.WithTracerProvider(t.Traces),
		otelhttp.WithMeterProvider(t.Metriques),
		otelhttp.WithSpanNameFormatter(NomDeSpan))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/metrics" {
			t.Prometheus.ServeHTTP(w, r)
			return
		}
		instrumente.ServeHTTP(w, r)
	})
}
