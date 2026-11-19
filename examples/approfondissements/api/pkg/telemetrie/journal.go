package telemetrie

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Journal enveloppe un slog.Handler et ajoute à chaque message les
// identifiants de la trace et du span en cours : depuis un message du
// journal, on retrouve la trace complète de la requête, et inversement.
type Journal struct {
	slog.Handler
}

// Handle ajoute trace_id et span_id si le contexte contient un span.
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
