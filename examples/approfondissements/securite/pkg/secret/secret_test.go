package secret_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/secret"
)

func Example() {
	type Config struct {
		Base       string
		MotDePasse secret.Secret
	}
	cfg := Config{Base: "taches", MotDePasse: secret.Nouveau("hunter2")}

	fmt.Printf("%v\n", cfg)
	fmt.Printf("%+v\n", cfg)
	fmt.Println(fmt.Errorf("connexion refusée avec %s", cfg.MotDePasse))
	j, _ := json.Marshal(cfg)
	fmt.Println(string(j))
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	log.Info("connexion", "mot_de_passe", cfg.MotDePasse)

	fmt.Println(cfg.MotDePasse.Valeur()) // révélé, explicitement
	// Output:
	// {taches [masqué]}
	// {Base:taches MotDePasse:[masqué]}
	// connexion refusée avec [masqué]
	// {"Base":"taches","MotDePasse":"[masqué]"}
	// level=INFO msg=connexion mot_de_passe=[masqué]
	// hunter2
}

func TestGoString(t *testing.T) {
	if got := fmt.Sprintf("%#v", secret.Nouveau("hunter2")); got != "[masqué]" {
		t.Errorf("%%#v = %q", got)
	}
}

func TestEgal(t *testing.T) {
	s := secret.Nouveau("hunter2")
	if !s.Egal("hunter2") || s.Egal("hunter3") || s.Egal("") {
		t.Error("comparaison incorrecte")
	}
}

func TestLit(t *testing.T) {
	fichier := filepath.Join(t.TempDir(), "mdp")
	if err := os.WriteFile(fichier, []byte("depuis-un-fichier\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"MDP": "depuis-env", "CLE_FILE": fichier, "ABSENT_FILE": "/nexiste/pas"}
	getenv := func(k string) string { return env[k] }

	if s, _ := secret.Lit(getenv, "MDP"); s.Valeur() != "depuis-env" {
		t.Errorf("MDP = %q", s.Valeur())
	}
	if s, _ := secret.Lit(getenv, "CLE"); s.Valeur() != "depuis-un-fichier" {
		t.Errorf("CLE = %q", s.Valeur())
	}
	if _, err := secret.Lit(getenv, "ABSENT"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ABSENT : %v", err)
	}
}
