// Package oidctest simule un fournisseur d'identité OpenID Connect (Keycloak,
// Auth0, Entra ID, Google…) pour les tests : il publie sa configuration et
// ses clés publiques, et signe des jetons à la demande.
package oidctest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Emetteur est un faux fournisseur d'identité.
type Emetteur struct {
	URL string // l'émetteur (« iss ») des jetons
	cle *ecdsa.PrivateKey
	kid string
}

// Nouveau démarre un fournisseur sur un port libre, arrêté à la fin du test.
func Nouveau(t testing.TB) *Emetteur {
	t.Helper()
	cle, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	e := &Emetteur{cle: cle, kid: "cle-1"}

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	e.URL = srv.URL

	// La découverte : où trouver les clés, et quels algorithmes sont utilisés.
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                e.URL,
			"jwks_uri":                              e.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"ES256"},
		})
	})
	// Les clés publiques (JWKS), qui permettent de vérifier les signatures.
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &cle.PublicKey, KeyID: e.kid, Algorithm: string(jose.ES256), Use: "sig"},
		}})
	})
	return e
}

// Revendications d'un jeton. Les champs vides sont omis.
type Revendications struct {
	Emetteur string    `json:"iss,omitempty"`
	Sujet    string    `json:"sub,omitempty"`
	Audience string    `json:"aud,omitempty"`
	Expire   time.Time `json:"-"`
	Portees  string    `json:"scope,omitempty"` // séparées par des espaces
}

// Jeton signe un jeton valable une heure pour l'audience donnée, avec les
// portées (scopes) données.
func (e *Emetteur) Jeton(t testing.TB, sujet, audience, portees string) string {
	return e.Signe(t, Revendications{Sujet: sujet, Audience: audience, Portees: portees}, e.cle)
}

// Signe signe des revendications arbitraires avec la clé donnée (celle de
// l'émetteur si elle est nil) : les tests peuvent ainsi fabriquer des jetons
// expirés, d'un autre émetteur, ou signés par une clé inconnue.
func (e *Emetteur) Signe(t testing.TB, r Revendications, cle *ecdsa.PrivateKey) string {
	t.Helper()
	if cle == nil {
		cle = e.cle
	}
	if r.Emetteur == "" {
		r.Emetteur = e.URL
	}
	if r.Expire.IsZero() {
		r.Expire = time.Now().Add(time.Hour)
	}
	signataire, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: cle},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", e.kid))
	if err != nil {
		t.Fatal(err)
	}
	brut, err := jwt.Signed(signataire).
		Claims(r).
		Claims(map[string]any{"exp": r.Expire.Unix(), "iat": time.Now().Unix()}).
		Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return brut
}
