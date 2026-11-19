package cli

import (
	"bytes"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/web"
)

// apiDeTest démarre la vraie API, sur une base SQLite en mémoire : les tests
// couvrent la ligne de commande, le client généré et le serveur ensemble.
func apiDeTest(t *testing.T) string {
	t.Helper()
	s, err := store.Ouvre(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	h, err := web.Nouveau(s, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

// lance exécute taches-cli avec les arguments donnés, et renvoie sa sortie.
func lance(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var sortie bytes.Buffer
	racine := NouvelleRacine(Options{Sortie: &sortie, Erreurs: &sortie})
	racine.SetArgs(args)
	err := racine.ExecuteContext(t.Context())
	return sortie.String(), err
}

func TestScenario(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // aucun fichier de configuration personnel
	url := apiDeTest(t)

	etapes := []struct {
		args     []string
		contient string
	}{
		{[]string{"ajoute", "Relire", "le", "chapitre"}, "Tâche 1 créée : Relire le chapitre"},
		{[]string{"ajoute", "Publier"}, "Tâche 2 créée : Publier"},
		{[]string{"fait", "1"}, "Tâche 1 faite : Relire le chapitre"},
		{[]string{"liste"}, "1   [x]    Relire le chapitre"},
		{[]string{"liste", "--a-faire"}, "2   [ ]    Publier"},
		{[]string{"ls", "-f", "json"}, `"titre": "Publier"`},
		{[]string{"supprime", "2"}, "Tâche 2 supprimée"},
	}
	for _, e := range etapes {
		sortie, err := lance(t, append(e.args, "--serveur", url)...)
		if err != nil {
			t.Fatalf("%v : %v", e.args, err)
		}
		if !strings.Contains(sortie, e.contient) {
			t.Errorf("%v : %q absent de\n%s", e.args, e.contient, sortie)
		}
	}
}

func TestErreurs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	url := apiDeTest(t)

	cas := []struct {
		args    []string
		message string
	}{
		{[]string{"fait", "42"}, "tâche introuvable (HTTP 404)"},
		{[]string{"fait", "abc"}, `identifiant invalide : "abc"`},
		{[]string{"ajoute", " "}, "titre : obligatoire (HTTP 422)"},
		{[]string{"liste", "--faites", "--a-faire"}, "none of the others can be"},
		{[]string{"liste", "--format", "xml"}, `format inconnu : "xml"`},
		{[]string{"supprime"}, "accepts 1 arg(s), received 0"},
	}
	for _, c := range cas {
		_, err := lance(t, append(c.args, "--serveur", url)...)
		if err == nil || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%v : erreur %v, attendu %q", c.args, err, c.message)
		}
	}
}

// TestPrioriteConfiguration vérifie l'ordre de priorité de viper : option >
// variable d'environnement > fichier de configuration > valeur par défaut.
func TestPrioriteConfiguration(t *testing.T) {
	url := apiDeTest(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// 1. Le fichier de configuration désigne la bonne API.
	if err := os.MkdirAll(filepath.Join(dir, "taches"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "serveur: " + url + "\nformat: json\n"
	if err := os.WriteFile(filepath.Join(dir, "taches", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if sortie, err := lance(t, "liste"); err != nil || strings.TrimSpace(sortie) != "[]" {
		t.Fatalf("depuis le fichier : %q, %v", sortie, err)
	}

	// 2. La variable d'environnement l'emporte sur le fichier.
	t.Setenv("TACHES_FORMAT", "table")
	if sortie, _ := lance(t, "liste"); !strings.HasPrefix(sortie, "ID") {
		t.Errorf("la variable d'environnement devrait l'emporter : %q", sortie)
	}

	// 3. L'option l'emporte sur la variable d'environnement.
	if sortie, _ := lance(t, "liste", "--format", "json"); strings.TrimSpace(sortie) != "[]" {
		t.Errorf("l'option devrait l'emporter : %q", sortie)
	}

	// 4. Un serveur injoignable donne une erreur claire.
	t.Setenv("TACHES_SERVEUR", "http://127.0.0.1:1")
	if _, err := lance(t, "liste"); err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("erreur = %v", err)
	}
}

func TestCompletion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	url := apiDeTest(t)
	lance(t, "ajoute", "Écrire", "--serveur", url)

	// Le shell appelle la commande cachée __complete pendant la saisie.
	sortie, err := lance(t, "__complete", "fait", "--serveur", url, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sortie, "1\tÉcrire") || !strings.Contains(sortie, ":4") {
		t.Errorf("complétion inattendue :\n%s", sortie)
	}
}

func TestAide(t *testing.T) {
	sortie, err := lance(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, attendu := range []string{"ajoute", "fait", "liste", "supprime", "completion", "--serveur"} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("%q absent de l'aide", attendu)
		}
	}
}
