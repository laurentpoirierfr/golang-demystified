package auth_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/auth"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/oidctest"
)

const audience = "api-notes"

func TestJetons(t *testing.T) {
	emetteur := oidctest.Nouveau(t)
	v, err := auth.NouveauVerificateur(t.Context(), emetteur.URL, audience)
	if err != nil {
		t.Fatal(err)
	}
	var journal strings.Builder
	h := auth.Authentifie(v, slog.New(slog.NewTextHandler(&journal, nil)))(
		auth.Exige("notes:lire", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, _ := auth.IdentiteDe(r.Context())
			fmt.Fprint(w, id.Sujet)
		})))

	autreCle, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tests := []struct {
		nom     string
		entete  string
		statut  int
		defi    string // en-tête WWW-Authenticate attendu
		journal string // raison journalisée attendue
	}{
		{"valide", "Bearer " + emetteur.Jeton(t, "alice", audience, "notes:lire notes:ecrire"), 200, "", ""},
		{"sans jeton", "", 401, `Bearer`, ""},
		{"autre schéma", "Basic YWxpY2U6c2VjcmV0", 401, `Bearer`, ""},
		{"jeton illisible", "Bearer abc.def.ghi", 401, `Bearer error="invalid_token"`, "malformed"},
		{"expiré", "Bearer " + emetteur.Signe(t, oidctest.Revendications{
			Sujet: "alice", Audience: audience, Portees: "notes:lire", Expire: time.Now().Add(-time.Minute),
		}, nil), 401, `Bearer error="invalid_token"`, "expired"},
		{"autre audience", "Bearer " + emetteur.Jeton(t, "alice", "autre-api", "notes:lire"),
			401, `Bearer error="invalid_token"`, "audience"},
		{"autre émetteur", "Bearer " + emetteur.Signe(t, oidctest.Revendications{
			Emetteur: "https://pirate.example", Sujet: "alice", Audience: audience, Portees: "notes:lire",
		}, nil), 401, `Bearer error="invalid_token"`, "different provider"},
		{"signé par une autre clé", "Bearer " + emetteur.Signe(t, oidctest.Revendications{
			Sujet: "alice", Audience: audience, Portees: "notes:lire",
		}, autreCle), 401, `Bearer error="invalid_token"`, "signature"},
		{"algorithme none", "Bearer " + sansSignature("alice"), 401, `Bearer error="invalid_token"`, ""},
		{"HMAC avec la clé publique", "Bearer " + hmacClePublique(t, emetteur.URL), 401, `Bearer error="invalid_token"`, "unexpected signature algorithm"},
		{"portée absente", "Bearer " + emetteur.Jeton(t, "bob", audience, "notes:ecrire"),
			403, `Bearer error="insufficient_scope", scope="notes:lire"`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			journal.Reset()
			req := httptest.NewRequest("GET", "/", nil)
			if tt.entete != "" {
				req.Header.Set("Authorization", tt.entete)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.statut {
				t.Fatalf("statut %d, attendu %d (%s)", rec.Code, tt.statut, journal.String())
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != tt.defi {
				t.Errorf("WWW-Authenticate = %q, attendu %q", got, tt.defi)
			}
			if !strings.Contains(journal.String(), tt.journal) {
				t.Errorf("journal = %q, devrait contenir %q", journal.String(), tt.journal)
			}
			if tt.entete != "" && strings.Contains(journal.String(), strings.TrimPrefix(tt.entete, "Bearer ")) {
				t.Error("le jeton ne doit jamais apparaître dans le journal")
			}
			if tt.statut == 200 && rec.Body.String() != "alice" {
				t.Errorf("sujet = %q", rec.Body.String())
			}
		})
	}
}

// sansSignature fabrique un jeton « alg: none », la plus vieille attaque
// contre les JWT : un vérificateur naïf l'accepterait sans signature.
func sansSignature(sujet string) string {
	b64 := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	return b64(`{"alg":"none","typ":"JWT"}`) + "." + b64(`{"sub":"`+sujet+`","aud":"`+audience+`"}`) + "."
}

func TestEmetteurInjoignable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	if _, err := auth.NouveauVerificateur(t.Context(), srv.URL, audience); err == nil {
		t.Error("un émetteur injoignable doit empêcher le démarrage")
	}
}

// hmacClePublique fabrique un jeton signé en HS256 avec, pour secret, la clé
// publique du fournisseur (que tout le monde peut télécharger). Un
// vérificateur qui accepterait l'algorithme annoncé par le jeton vérifierait
// cette signature avec… la clé publique, et l'accepterait.
func hmacClePublique(t *testing.T, emetteur string) string {
	t.Helper()
	rep, err := http.Get(emetteur + "/jwks")
	if err != nil {
		t.Fatal(err)
	}
	clePublique, _ := io.ReadAll(rep.Body)
	rep.Body.Close()
	signataire, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: clePublique}, nil)
	if err != nil {
		t.Fatal(err)
	}
	brut, err := jwt.Signed(signataire).Claims(map[string]any{
		"iss": emetteur, "sub": "alice", "aud": audience, "scope": "notes:lire",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return brut
}
