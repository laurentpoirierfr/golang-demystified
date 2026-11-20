package service

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"connectrpc.com/connect/v2"
)

// Journal journalise chaque appel : procédure, protocole, code et durée. Un
// intercepteur de Connect v2 entoure l'appel entier, flux compris.
func Journal(log *slog.Logger) connect.ServerInterceptor {
	return func(suivant connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, flux connect.ServerStream) error {
			debut := time.Now()
			err := suivant(ctx, spec, flux)
			info, _ := connect.CallInfoForServerContext(ctx)
			niveau := slog.LevelInfo
			if err != nil {
				niveau = slog.LevelWarn
			}
			attrs := []any{"procedure", spec.Procedure}
			if info.Protocol != "" { // vide pour un appel en mémoire (connectinprocess)
				attrs = append(attrs, "protocole", info.Protocol)
			}
			attrs = append(attrs, "code", codeTexte(err), "duree", time.Since(debut))
			log.Log(ctx, niveau, "appel", attrs...)
			return err
		}
	}
}

// codeTexte renvoie « ok » ou le nom du code d'erreur (« not_found »).
func codeTexte(err error) string {
	if err == nil {
		return "ok"
	}
	return connect.CodeOf(err).String()
}

// Recuperation transforme une panique en erreur INTERNAL, sans arrêter le
// serveur ni divulguer le message de la panique au client.
func Recuperation(log *slog.Logger) connect.ServerInterceptor {
	return func(suivant connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, flux connect.ServerStream) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.ErrorContext(ctx, "panique", "procedure", spec.Procedure, "valeur", r)
					err = connect.NewError(connect.CodeInternal, "erreur interne")
				}
			}()
			return suivant(ctx, spec, flux)
		}
	}
}

// Authentification refuse les appels sans l'en-tête « Authorization: Bearer
// <jeton> ». Le contrôle a lieu avant toute lecture du message : un client
// non authentifié ne coûte presque rien au serveur.
func Authentification(jeton string) connect.ServerInterceptor {
	attendu := []byte("Bearer " + jeton)
	return func(suivant connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, flux connect.ServerStream) error {
			info, _ := connect.CallInfoForServerContext(ctx)
			recu := []byte(info.RequestHeader().Get("Authorization"))
			// Comparaison en temps constant : la durée ne révèle rien du jeton.
			if subtle.ConstantTimeCompare(recu, attendu) != 1 {
				return connect.NewError(connect.CodeUnauthenticated, "jeton absent ou invalide")
			}
			return suivant(ctx, spec, flux)
		}
	}
}

// Jeton ajoute l'en-tête d'authentification à chaque appel d'un client.
func Jeton(jeton string) connect.ClientInterceptor {
	return func(suivant connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			// L'appelant a peut-être déjà attaché un CallInfo pour lire les
			// en-têtes de la réponse : on le réutilise plutôt que de le masquer.
			info, ok := connect.CallInfoForClientContext(ctx)
			if !ok {
				ctx, info = connect.NewClientContext(ctx)
			}
			info.RequestHeader().Set("Authorization", "Bearer "+jeton)
			return suivant(ctx, spec)
		}
	}
}
