// Package auth authentifie les requêtes par un jeton JWT émis par un
// fournisseur d'identité OpenID Connect, et vérifie leurs autorisations.
package auth

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Identite décrit l'appelant authentifié.
type Identite struct {
	Sujet   string   // identifiant stable de l'utilisateur ou du service (« sub »)
	Portees []string // autorisations accordées (« scope »)
}

// APortee indique si l'appelant a reçu l'autorisation donnée.
func (i Identite) APortee(portee string) bool { return slices.Contains(i.Portees, portee) }

type cleIdentite struct{}

// IdentiteDe renvoie l'identité de l'appelant, placée dans le contexte par
// Authentifie.
func IdentiteDe(ctx context.Context) (Identite, bool) {
	i, ok := ctx.Value(cleIdentite{}).(Identite)
	return i, ok
}

// Verificateur vérifie un jeton. *oidc.IDTokenVerifier le satisfait.
type Verificateur interface {
	Verify(ctx context.Context, brut string) (*oidc.IDToken, error)
}

// NouveauVerificateur interroge la découverte OpenID Connect de l'émetteur
// (/.well-known/openid-configuration) pour trouver ses clés publiques, et
// renvoie un vérificateur qui exige l'audience donnée. Les clés sont mises en
// cache, et rechargées quand le fournisseur en change (rotation).
func NouveauVerificateur(ctx context.Context, emetteur, audience string) (*oidc.IDTokenVerifier, error) {
	fournisseur, err := oidc.NewProvider(ctx, emetteur)
	if err != nil {
		return nil, err
	}
	return fournisseur.Verifier(&oidc.Config{ClientID: audience}), nil
}

// Authentifie exige un jeton valide dans l'en-tête « Authorization: Bearer ».
// Le vérificateur contrôle la signature, l'émetteur, l'audience et
// l'expiration ; l'identité est ensuite placée dans le contexte.
func Authentifie(v Verificateur, log *slog.Logger) func(http.Handler) http.Handler {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			brut, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || brut == "" {
				refuse(w, `Bearer`)
				return
			}
			jeton, err := v.Verify(r.Context(), brut)
			if err != nil {
				// La raison est journalisée, jamais le jeton ; le client, lui,
				// n'apprend rien qui l'aiderait à forger un jeton.
				log.WarnContext(r.Context(), "jeton refusé", "raison", err.Error())
				refuse(w, `Bearer error="invalid_token"`)
				return
			}
			var revendications struct {
				Scope string `json:"scope"`
			}
			if err := jeton.Claims(&revendications); err != nil {
				refuse(w, `Bearer error="invalid_token"`)
				return
			}
			id := Identite{Sujet: jeton.Subject, Portees: strings.Fields(revendications.Scope)}
			suivant.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), cleIdentite{}, id)))
		})
	}
}

// Exige refuse la requête si l'appelant n'a pas reçu la portée donnée.
// L'authentification (qui êtes-vous ?) répond 401 ; l'autorisation (en
// avez-vous le droit ?) répond 403.
func Exige(portee string, suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IdentiteDe(r.Context())
		if !ok {
			refuse(w, `Bearer`)
			return
		}
		if !id.APortee(portee) {
			w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="`+portee+`"`)
			http.Error(w, "autorisation insuffisante", http.StatusForbidden)
			return
		}
		suivant.ServeHTTP(w, r)
	})
}

// refuse répond 401, avec l'en-tête WWW-Authenticate de la RFC 6750.
func refuse(w http.ResponseWriter, defi string) {
	w.Header().Set("WWW-Authenticate", defi)
	http.Error(w, "authentification requise", http.StatusUnauthorized)
}
