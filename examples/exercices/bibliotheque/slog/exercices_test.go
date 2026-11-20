package slog

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"
)

func TestNouveau(t *testing.T) {
	var b bytes.Buffer
	log := Nouveau(&b, slog.LevelInfo, "commandes")
	log.Debug("ignoré")
	log.Info("commande reçue", "id", 42)
	var m map[string]any
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatalf("%v : %s", err, b.String())
	}
	if m["msg"] != "commande reçue" || m["service"] != "commandes" || m["id"] != 42.0 || m["level"] != "INFO" {
		t.Errorf("%v", m)
	}
}

func TestSecret(t *testing.T) {
	var b bytes.Buffer
	log := slog.New(slog.NewTextHandler(&b, nil))
	log.Info("connexion", "utilisateur", "ada", "mot_de_passe", Secret("hunter2"))
	if strings.Contains(b.String(), "hunter2") || !strings.Contains(b.String(), "mot_de_passe=***") {
		t.Errorf("%s", b.String())
	}
}

func TestCompteur(t *testing.T) {
	c := NouveauCompteur(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log := slog.New(c)
	log.Info("un")
	log.With("requete", 7).WithGroup("http").Warn("deux")
	log.Error("trois")
	log.Info("quatre")
	if c.Nombre(slog.LevelInfo) != 2 || c.Nombre(slog.LevelWarn) != 1 || c.Nombre(slog.LevelError) != 1 {
		t.Errorf("info %d, warn %d, error %d", c.Nombre(slog.LevelInfo), c.Nombre(slog.LevelWarn), c.Nombre(slog.LevelError))
	}
}

// slogtest vérifie qu'un handler respecte le contrat de slog : attributs,
// groupes, valeurs vides… Le Compteur, qui délègue tout, doit le passer.
func TestCompteurContrat(t *testing.T) {
	var b bytes.Buffer
	c := NouveauCompteur(slog.NewJSONHandler(&b, nil))
	err := slogtest.TestHandler(c, func() []map[string]any {
		var r []map[string]any
		for _, ligne := range bytes.Split(bytes.TrimSpace(b.Bytes()), []byte("\n")) {
			var m map[string]any
			if err := json.Unmarshal(ligne, &m); err != nil {
				t.Fatal(err)
			}
			r = append(r, m)
		}
		return r
	})
	if err != nil {
		t.Error(err)
	}
}
