package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/oidctest"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/pki"
)

// environnement prépare une autorité, un certificat de serveur, un
// fournisseur d'identité et un répertoire de fichiers.
type environnement struct {
	ac       *pki.AC
	emetteur *oidctest.Emetteur
	env      map[string]string
}

func prepare(t *testing.T) *environnement {
	t.Helper()
	dir := t.TempDir()
	ac, err := pki.NouvelleAC("AC de test")
	if err != nil {
		t.Fatal(err)
	}
	serveur, err := ac.Serveur("localhost", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	ecrit := func(nom string, contenu []byte) string {
		chemin := filepath.Join(dir, nom)
		if err := os.WriteFile(chemin, contenu, 0o600); err != nil {
			t.Fatal(err)
		}
		return chemin
	}
	fichiers := filepath.Join(dir, "fichiers")
	if err := os.Mkdir(fichiers, 0o700); err != nil {
		t.Fatal(err)
	}
	ecrit("fichiers/rapport.txt", []byte("contenu du rapport"))
	ecrit("secret.txt", []byte("hors du répertoire autorisé"))
	// Un lien symbolique qui pointe hors du répertoire : os.Root le refuse.
	if err := os.Symlink(filepath.Join(dir, "secret.txt"), filepath.Join(fichiers, "lien")); err != nil {
		t.Fatal(err)
	}

	e := oidctest.Nouveau(t)
	return &environnement{ac: ac, emetteur: e, env: map[string]string{
		"SECURITE_ADRESSE":  "127.0.0.1:0",
		"OIDC_EMETTEUR":     e.URL,
		"SECURITE_FICHIERS": fichiers,
		"TLS_CERT":          ecrit("serveur.pem", serveur.CertPEM),
		"TLS_CLE":           ecrit("serveur-cle.pem", serveur.ClePEM),
		"CLE_ADMIN_FILE":    ecrit("cle-admin", []byte("cle-tres-secrete\n")),
	}}
}

// demarre lance le vrai programme et renvoie son URL et son journal.
func (e *environnement) demarre(t *testing.T) (string, *strings.Builder) {
	t.Helper()
	ctx, arrete := context.WithCancel(context.Background())
	var journal strings.Builder
	pret := make(chan string, 1)
	fini := make(chan error, 1)
	go func() { fini <- run(ctx, func(k string) string { return e.env[k] }, &journal, pret) }()
	t.Cleanup(func() {
		arrete()
		if err := <-fini; err != nil {
			t.Errorf("arrêt : %v", err)
		}
	})
	select {
	case adresse := <-pret:
		return "https://" + adresse, &journal
	case err := <-fini:
		t.Fatalf("démarrage : %v", err)
		return "", nil
	}
}

// client fait confiance à l'autorité de test, et à elle seule.
func (e *environnement) client(certificats ...tls.Certificate) *http.Client {
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs:      e.ac.Pool(),
		Certificates: certificats,
	}}}
}

func requete(t *testing.T, c *http.Client, methode, url, jeton, corps string, entetes ...string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), methode, url, strings.NewReader(corps))
	if err != nil {
		t.Fatal(err)
	}
	if jeton != "" {
		req.Header.Set("Authorization", "Bearer "+jeton)
	}
	for i := 0; i+1 < len(entetes); i += 2 {
		req.Header.Set(entetes[i], entetes[i+1])
	}
	rep, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer rep.Body.Close()
	b, _ := io.ReadAll(rep.Body)
	return rep.StatusCode, strings.TrimSpace(string(b)), rep.Header
}

func TestNotes(t *testing.T) {
	e := prepare(t)
	base, journal := e.demarre(t)
	c := e.client()
	alice := e.emetteur.Jeton(t, "alice", "api-notes", "notes:lire notes:ecrire")
	bob := e.emetteur.Jeton(t, "bob", "api-notes", "notes:lire")

	statut, corps, entetes := requete(t, c, "POST", base+"/notes", alice, `{"titre": "Ma note", "contenu": "<script>alert(1)</script>"}`)
	if statut != 201 {
		t.Fatalf("création : %d %s", statut, corps)
	}
	// encoding/json échappe le HTML ; les en-têtes de sécurité sont présents.
	if !strings.Contains(corps, `\u003cscript\u003e`) {
		t.Errorf("contenu non échappé : %s", corps)
	}
	if entetes.Get("X-Content-Type-Options") != "nosniff" || entetes.Get("Strict-Transport-Security") == "" {
		t.Errorf("en-têtes de sécurité absents : %v", entetes)
	}

	// Bob peut lire, mais ne voit pas les notes d'Alice, et ne peut pas écrire.
	if statut, corps, _ := requete(t, c, "GET", base+"/notes", bob, ""); statut != 200 || corps != "[]" {
		t.Errorf("notes de bob : %d %s", statut, corps)
	}
	if statut, _, _ := requete(t, c, "POST", base+"/notes", bob, `{"titre": "x"}`); statut != 403 {
		t.Errorf("écriture par bob : %d", statut)
	}

	// La configuration est journalisée, mais la clé d'administration est masquée.
	if !strings.Contains(journal.String(), `"cle_admin":"[masqué]"`) || strings.Contains(journal.String(), "cle-tres-secrete") {
		t.Errorf("journal : %s", journal.String())
	}
}

func TestValidationDesEntrees(t *testing.T) {
	e := prepare(t)
	base, _ := e.demarre(t)
	c := e.client()
	alice := e.emetteur.Jeton(t, "alice", "api-notes", "notes:lire notes:ecrire")

	tests := []struct {
		nom, corps string
		statut     int
	}{
		{"champ inconnu (mass assignment)", `{"titre": "x", "auteur": "bob"}`, 400},
		{"JSON invalide", `{"titre": `, 400},
		{"deux objets", `{"titre": "a"} {"titre": "b"}`, 400},
		{"titre vide", `{"titre": "   "}`, 422},
		{"titre trop long", `{"titre": "` + strings.Repeat("é", 101) + `"}`, 422},
		{"caractère de contrôle", `{"titre": "a\u0000b"}`, 422},
		{"corps trop volumineux", `{"titre": "x", "contenu": "` + strings.Repeat("a", 20<<10) + `"}`, 413},
	}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			if statut, corps, _ := requete(t, c, "POST", base+"/notes", alice, tt.corps); statut != tt.statut {
				t.Errorf("statut %d, attendu %d : %s", statut, tt.statut, corps)
			}
		})
	}
}

func TestFichiersConfines(t *testing.T) {
	e := prepare(t)
	base, journal := e.demarre(t)
	c := e.client()
	alice := e.emetteur.Jeton(t, "alice", "api-notes", "notes:lire")

	statut, corps, entetes := requete(t, c, "GET", base+"/fichiers/rapport.txt", alice, "")
	if statut != 200 || corps != "contenu du rapport" || entetes.Get("Content-Disposition") != "attachment" {
		t.Errorf("fichier autorisé : %d %q %v", statut, corps, entetes)
	}
	// « .. » encodé (%2e%2e%2f) : le routeur ne le nettoie pas, PathValue le
	// décode. C'est os.Root qui protège.
	for _, chemin := range []string{
		"/fichiers/" + url.PathEscape("../secret.txt"),
		"/fichiers/%2e%2e%2fsecret.txt",
		"/fichiers/" + url.PathEscape("/etc/passwd"),
		"/fichiers/lien",
	} {
		if statut, corps, _ := requete(t, c, "GET", base+chemin, alice, ""); statut != 404 || strings.Contains(corps, "hors du répertoire") {
			t.Errorf("%s : %d %q", chemin, statut, corps)
		}
	}
	if !strings.Contains(journal.String(), "fichier refusé") {
		t.Error("les tentatives devraient être journalisées")
	}
}

func TestCSRF(t *testing.T) {
	e := prepare(t)
	base, _ := e.demarre(t)
	alice := e.emetteur.Jeton(t, "alice", "api-notes", "notes:ecrire")
	// Un formulaire d'un autre site, soumis par le navigateur de la victime.
	statut, _, _ := requete(t, e.client(), "POST", base+"/notes", alice, `{"titre": "piège"}`,
		"Sec-Fetch-Site", "cross-site", "Origin", "https://pirate.example")
	if statut != 403 {
		t.Errorf("requête intersite : statut %d, attendu 403", statut)
	}
}

func TestTLS(t *testing.T) {
	e := prepare(t)
	base, _ := e.demarre(t)

	// Un client qui ne connaît pas l'autorité refuse le serveur.
	if err := connecte(http.DefaultClient, base); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("certificat inconnu accepté : %v", err)
	}
	// TLS 1.2 est refusé.
	ancien := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: e.ac.Pool(), MaxVersion: tls.VersionTLS12,
	}}}
	if err := connecte(ancien, base); err == nil {
		t.Error("TLS 1.2 accepté")
	}
	// Le bon client négocie TLS 1.3, avec un échange de clés post-quantique.
	rep, err := e.client().Get(base + "/sante")
	if err != nil {
		t.Fatal(err)
	}
	rep.Body.Close()
	if rep.TLS.Version != tls.VersionTLS13 || rep.TLS.CurveID != tls.X25519MLKEM768 {
		t.Errorf("version %x, échange de clés %v", rep.TLS.Version, rep.TLS.CurveID)
	}
}

func TestMTLS(t *testing.T) {
	e := prepare(t)
	fichierAC := filepath.Join(t.TempDir(), "ac.pem")
	if err := os.WriteFile(fichierAC, e.ac.PEM(), 0o600); err != nil {
		t.Fatal(err)
	}
	e.env["TLS_AC_CLIENTS"] = fichierAC
	base, _ := e.demarre(t)
	jeton := e.emetteur.Jeton(t, "service-facturation", "api-notes", "notes:lire")

	// Sans certificat client, la poignée de main TLS échoue.
	if err := connecte(e.client(), base); err == nil {
		t.Error("client sans certificat accepté")
	}
	// Un certificat d'une autre autorité est refusé.
	autreAC, _ := pki.NouvelleAC("Autre AC")
	intrus, _ := autreAC.Client("intrus")
	certIntrus, _ := intrus.TLS()
	if err := connecte(e.client(certIntrus), base); err == nil {
		t.Error("certificat d'une autre autorité accepté")
	}

	client, _ := e.ac.Client("facturation")
	cert, err := client.TLS()
	if err != nil {
		t.Fatal(err)
	}
	statut, corps, _ := requete(t, e.client(cert), "GET", base+"/moi", jeton, "")
	var moi struct {
		Sujet, Certificat string
	}
	json.Unmarshal([]byte(corps), &moi)
	if statut != 200 || moi.Sujet != "service-facturation" || moi.Certificat != "facturation" {
		t.Errorf("/moi : %d %s", statut, corps)
	}
}

func TestAdministration(t *testing.T) {
	e := prepare(t)
	base, _ := e.demarre(t)
	c := e.client()
	if statut, _, _ := requete(t, c, "GET", base+"/admin/stats", "", "", "X-Cle-Admin", "mauvaise"); statut != 403 {
		t.Errorf("mauvaise clé : %d", statut)
	}
	if statut, corps, _ := requete(t, c, "GET", base+"/admin/stats", "", "", "X-Cle-Admin", "cle-tres-secrete"); statut != 200 || corps != `{"notes":0}` {
		t.Errorf("bonne clé : %d %s", statut, corps)
	}
}

func TestEmetteurObligatoire(t *testing.T) {
	err := run(t.Context(), func(string) string { return "" }, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "OIDC_EMETTEUR") {
		t.Errorf("err = %v", err)
	}
}

// connecte appelle /sante et renvoie l'erreur de connexion éventuelle ; une
// réponse inattendue est refermée.
func connecte(c *http.Client, base string) error {
	rep, err := c.Get(base + "/sante")
	if err != nil {
		return err
	}
	return rep.Body.Close()
}
