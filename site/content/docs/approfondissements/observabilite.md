---
title: "V.9 Observabilité"
weight: 90
---

# Chapitre V.9 — Observabilité : traces, métriques et journaux

Un service en production doit pouvoir répondre à trois questions : **que
s'est-il passé ?** (les journaux), **combien et à quelle vitesse ?** (les
métriques), et **où le temps a-t-il été passé ?** (les traces). Ce sont les
trois signaux de l'**observabilité**.

[OpenTelemetry](https://opentelemetry.io) est le standard ouvert qui les
unifie : une API et un SDK pour produire les signaux, un protocole (OTLP)
pour les transporter, et des conventions de nommage communes. Il est
compatible avec la plupart des outils : [Jaeger](https://www.jaegertracing.io)
et Tempo pour les traces, [Prometheus](https://prometheus.io) pour les
métriques, Grafana pour tout afficher, et les offres des fournisseurs de cloud.

Ce chapitre rend observable l'API des tâches du chapitre « Une API web
complète ». Le code est dans [`examples/approfondissements/api`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/api),
dans le package réutilisable `pkg/telemetrie`.

## Les trois signaux, reliés

Le résultat final, observé sur l'API réelle, montre comment les signaux se
complètent. Une requête `PATCH /taches/1` produit une ligne de journal :

```json
{"level":"INFO","msg":"requête","methode":"PATCH","chemin":"/taches/1","statut":200,
 "trace_id":"bb546a740ef2c88e4a857d6525e51429","span_id":"32310684a51f17b0"}
```

Son `trace_id` mène, dans Jaeger, à la trace complète de la requête, avec le
détail des opérations de stockage :

```
bb546a74  PATCH /taches/{id}
bb546a74  store.MarqueFaite     ← enfant de PATCH /taches/{id}
bb546a74  store.Trouve          ← enfant de store.MarqueFaite
```

Et les métriques, exposées au format Prometheus, comptent les requêtes par
route et par statut :

```
http_server_request_duration_seconds_count{http_request_method="PATCH",http_response_status_code="200",http_route="/taches/{id}",…} 1
http_server_request_duration_seconds_count{http_request_method="POST",http_response_status_code="201",http_route="/taches",…} 1
taches_creees_total{…} 1
```

## Mettre en place OpenTelemetry

Les modules d'OpenTelemetry pour Go sont :

| Module | Rôle |
|--------|------|
| `go.opentelemetry.io/otel` | L'API : traceurs, compteurs, propagation |
| `go.opentelemetry.io/otel/sdk` et `sdk/metric` | L'implémentation : échantillonnage, regroupement, export |
| `…/exporters/otlp/otlptrace/otlptracehttp` | Export des traces en OTLP sur HTTP |
| `…/exporters/prometheus` | Exposition des métriques au format Prometheus |
| `go.opentelemetry.io/contrib/…/otelhttp` | Instrumentation automatique de `net/http` |

Une bibliothèque n'utilise que l'**API** : sans SDK configuré, ses appels ne
font rien et ne coûtent presque rien. C'est le programme principal qui choisit
le SDK et les exportateurs. Le package `pkg/telemetrie` regroupe cette
configuration :

```go
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
		optsTraces = append(optsTraces, sdktrace.WithBatcher(o.Exportateur))
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
		…
	}, nil
}
```

Les fournisseurs sont passés explicitement aux composants, plutôt que d'être
enregistrés comme fournisseurs globaux : chaque test peut ainsi utiliser les
siens.

### Exporter les traces

Dans le programme principal, les traces sont exportées en OTLP si un
collecteur est configuré :

```go
options := telemetrie.Options{Service: "api-taches", Version: "1.0.0"}
if getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
	// L'exportateur lit lui-même les variables standard d'OpenTelemetry :
	// OTEL_EXPORTER_OTLP_ENDPOINT (adresse de base, complétée par /v1/traces),
	// OTEL_EXPORTER_OTLP_HEADERS (authentification), etc.
	exportateur, err := otlptracehttp.New(ctx)
	if err != nil {
		return fmt.Errorf("traces : %w", err)
	}
	options.Exportateur = exportateur
}
tel, err := telemetrie.Demarre(options)
if err != nil {
	return err
}
defer func() {
	// Envoie les dernières traces avant de quitter.
	arret, annule := context.WithTimeout(context.Background(), 5*time.Second)
	defer annule()
	if err := tel.Arrete(arret); err != nil {
		log.Warn("arrêt de la télémétrie", "erreur", err)
	}
}()
```

> [!WARNING]
> `OTEL_EXPORTER_OTLP_ENDPOINT` est une adresse **de base**
> (`http://collecteur:4318`), que le SDK complète par `/v1/traces`. L'option
> `otlptracehttp.WithEndpointURL`, elle, attend l'adresse **complète**. Lui
> passer la variable provoque des erreurs `404 Not Found` à chaque envoi :
> c'est l'erreur qu'a révélée le premier essai de ce chapitre contre un vrai
> Jaeger. Laisser l'exportateur lire ses variables d'environnement évite le
> piège, et prend aussi en charge l'authentification
> (`OTEL_EXPORTER_OTLP_HEADERS`) sans code supplémentaire.

Les traces sont **regroupées par lots** (`WithBatcher`) et envoyées en arrière-plan :
la requête n'attend jamais l'exportation. C'est pourquoi l'arrêt du programme
doit appeler `Arrete`, pour envoyer le dernier lot.

## Les traces

### L'instrumentation automatique : otelhttp

`otelhttp` enveloppe un `http.Handler` : chaque requête crée un **span**
racine, avec la méthode, la route, le statut et la durée, et alimente les
métriques HTTP standard. Il lit aussi l'en-tête `traceparent` : si la requête
vient d'un autre service instrumenté, son span rejoint la trace de l'appelant.

```go
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
```

### Nommer les spans d'après la route

Un span doit être nommé d'après la **route** (`GET /taches/{id}`), et non
d'après le chemin (`GET /taches/42`) : sinon, chaque identifiant crée un nom
différent, et les outils ne peuvent plus regrouper les requêtes.

La difficulté est que `otelhttp` enveloppe le routeur : quand il crée le span,
la route n'est pas encore connue. Un middleware placé **après** le routage,
là où `r.Pattern` est renseigné (Go 1.22), la transmet au span et aux
métriques :

```go
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
```

Dans l'API, il est ajouté aux middlewares appliqués par le code généré
d'oapi-codegen, qui s'exécutent après le routage :

```go
HandlerWithOptions(s, StdHTTPServerOptions{
	BaseRouter:  mux,
	Middlewares: []MiddlewareFunc{valide, telemetrie.Route},
	…
})
```

> [!WARNING]
> Ne placez pas de `http.ServeMux` **devant** `otelhttp`, par exemple pour
> servir `/metrics` à côté de l'API. `otelhttp` lirait la route de ce routeur
> (« / ») et l'imposerait au nom des spans et aux étiquettes des métriques :
> toutes les requêtes auraient la même route. Le premier essai de ce chapitre
> avait commis cette erreur, que seules les métriques observées sur le service
> réel ont révélée. D'où le simple `HandlerFunc` de `Telemetrie.Handler`, et un
> test qui vérifie désormais l'étiquette `http_route="/taches"`.

### L'instrumentation manuelle : les spans du stockage

Les opérations de stockage créent leurs propres spans. Comme le contexte de
la requête leur est transmis, ils deviennent automatiquement des **enfants**
du span de la requête :

```go
func (s *Store) debut(ctx context.Context, operation string) (context.Context, trace.Span) {
	return s.traceur.Start(ctx, "store."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", s.d.systeme),
			attribute.String("db.operation.name", operation),
		))
}

// fin termine le span, en y enregistrant l'erreur éventuelle. Une tâche
// introuvable est un résultat normal, pas une erreur technique.
func fin(span trace.Span, err error) {
	if err != nil && !errors.Is(err, tache.ErrIntrouvable) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func (s *Store) Cree(ctx context.Context, titre string) (_ tache.Tache, err error) {
	ctx, span := s.debut(ctx, "Cree")
	defer func() { fin(span, err) }()
	…
}
```

Trois détails comptent :

- le **contexte renvoyé** par `Start` doit être utilisé pour la suite : c'est
  lui qui porte le span, et qui rend enfants les spans créés ensuite
  (`store.Trouve` appelé par `store.MarqueFaite`) ;
- le résultat **nommé** `err` permet à la fonction différée de connaître
  l'erreur renvoyée ;
- seules les erreurs **techniques** marquent le span en erreur : une tâche
  introuvable est une réponse normale, qui ne doit pas déclencher d'alerte.

Par défaut, le stockage utilise un traceur sans effet
(`noop.NewTracerProvider()`) : le code fonctionne à l'identique quand
l'observabilité n'est pas activée, notamment dans ses propres tests.

> [!TIP]
> Pour les bases de données, les bibliothèques `otelsql` ou l'instrumentation
> intégrée de pgx (`otelpgx`) créent ces spans automatiquement pour chaque
> requête SQL. L'instrumentation manuelle reste utile pour les étapes métier :
> un calcul, un appel à un service tiers, une file de messages.

## Les métriques

`otelhttp` produit déjà les métriques HTTP standard (durée, taille des
requêtes et des réponses). Les métriques **métier** se déclarent avec un
`Meter` :

```go
func (s *Serveur) Instrumente(mp metric.MeterProvider) error {
	mesures := mp.Meter("github.com/…/api/internal/web")
	creees, err := mesures.Int64Counter("taches.creees",
		metric.WithDescription("Nombre de tâches créées"),
		metric.WithUnit("{tache}"))
	if err != nil {
		return err
	}
	s.creees = creees
	return nil
}

// Dans CreeTache, après l'enregistrement :
s.creees.Add(r.Context(), 1)
```

| Instrument | Usage | Exemple |
|------------|-------|---------|
| `Int64Counter` | ce qui ne fait qu'augmenter | tâches créées, erreurs |
| `Int64UpDownCounter` | ce qui monte et descend | connexions ouvertes |
| `Float64Histogram` | une distribution de valeurs | durées, tailles |
| `Int64ObservableGauge` | une valeur lue périodiquement | taille d'une file |

L'exportateur Prometheus traduit les noms selon les conventions de
Prometheus : le compteur `taches.creees` devient `taches_creees_total`, et
la durée `http.server.request.duration` devient
`http_server_request_duration_seconds`, un histogramme dont on tire les
percentiles.

> [!CAUTION]
> Chaque combinaison de valeurs d'étiquettes crée une série temporelle. Une
> étiquette à valeurs illimitées (un identifiant d'utilisateur, un chemin
> complet) fait exploser la mémoire de Prometheus. C'est une raison de plus
> d'utiliser la route, et non le chemin.

## Les journaux corrélés

Un journal qui ne porte pas l'identifiant de trace oblige à rapprocher les
signaux à la main, par l'heure. Un `slog.Handler` qui enveloppe le handler
JSON ajoute `trace_id` et `span_id` à chaque message émis avec un contexte :

```go
type Journal struct {
	slog.Handler
}

func (j Journal) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return j.Handler.Handle(ctx, r)
}

// WithAttrs et WithGroup doivent renvoyer un Journal, sinon les journaux
// dérivés (logger.With…) perdraient la corrélation.
func (j Journal) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Journal{j.Handler.WithAttrs(attrs)}
}

func (j Journal) WithGroup(nom string) slog.Handler {
	return Journal{j.Handler.WithGroup(nom)}
}
```

```go
log := slog.New(telemetrie.Journal{Handler: slog.NewJSONHandler(sortie, nil)})
```

Pour que la corrélation fonctionne, les messages doivent être émis avec le
contexte de la requête : `log.InfoContext(r.Context(), …)` plutôt que
`log.Info(…)`. C'est ce que fait le middleware de journalisation de
`pkg/httpkit` depuis le chapitre « Une API web complète ».

## Tester l'observabilité

Le SDK fournit un **exportateur en mémoire**, qui conserve les spans au lieu de
les envoyer. Un test vérifie ensemble les trois signaux sur une vraie requête :

```go
traces := tracetest.NewInMemoryExporter()
tel, err := telemetrie.Demarre(telemetrie.Options{
	Service: "api-taches", Version: "test", Exportateur: traces, Synchrone: true,
})
…
h.ServeHTTP(rec, httptest.NewRequest("POST", "/taches", …))

// 1. Traces : un span pour la requête, et un span enfant pour le stockage.
serveur, stockage := noms["POST /taches"], noms["store.Cree"]
if stockage.Parent.SpanID() != serveur.SpanContext.SpanID() {
	t.Error("le span du stockage devrait être l'enfant du span de la requête")
}

// 2. Journaux : la ligne de la requête porte l'identifiant de la trace.
if ligne["trace_id"] != serveur.SpanContext.TraceID().String() { … }

// 3. Métriques : métrique métier et métriques HTTP, étiquetées par la route.
for _, attendu := range []string{
	"taches_creees_total{",
	"http_server_request_duration_seconds_bucket{",
	`http_route="/taches"`,
	`service_name="api-taches"`,
} { … }
```

L'option `Synchrone` exporte chaque span dès sa fin : le test n'a pas à
attendre l'envoi d'un lot.

## Essayer avec Jaeger

Jaeger reçoit les traces en OTLP et les affiche dans son interface web :

```bash
docker run -d --name jaeger -p 16686:16686 -p 4318:4318 jaegertracing/jaeger:2.22.0

OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./cmd/taches
```

Après quelques requêtes, l'interface `http://localhost:16686` affiche les
traces du service `api-taches`. Prometheus, lui, collecte les métriques en
interrogeant périodiquement `/metrics` :

```yaml
# prometheus.yml
scrape_configs:
  - job_name: api-taches
    static_configs:
      - targets: ["localhost:8080"]
```

En production, on place souvent un **collecteur OpenTelemetry** entre les
services et les outils : il reçoit tous les signaux en OTLP, les filtre,
les échantillonne, et les redistribue vers les outils choisis. Changer d'outil
ne demande alors aucune modification des services.

## En résumé

| Signal | Question | Outil dans ce chapitre | Clé de corrélation |
|--------|----------|------------------------|--------------------|
| Journaux | que s'est-il passé ? | `slog` + `telemetrie.Journal` | `trace_id` |
| Traces | où le temps a-t-il été passé ? | `otelhttp`, spans manuels, OTLP, Jaeger | `trace_id` |
| Métriques | combien, à quelle vitesse ? | `otelhttp`, compteurs métier, Prometheus | `service_name`, `http_route` |

Et deux leçons tirées de la mise au point de ce chapitre : vérifier
l'observabilité **contre un vrai outil** (un exportateur mal configuré échoue
silencieusement pour le service), et vérifier les **étiquettes** des métriques,
pas seulement leur présence.
