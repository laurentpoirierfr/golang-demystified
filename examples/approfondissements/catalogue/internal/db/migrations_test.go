package db

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var nomMigration = regexp.MustCompile(`^(\d{6})_[a-z0-9_]+\.(up|down)\.sql$`)

// TestFichiersDeMigration vérifie, sans base de données, que chaque migration
// a un nom conforme et une migration inverse.
func TestFichiersDeMigration(t *testing.T) {
	fichiers, err := fs.Glob(migrations, "migrations/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(fichiers) == 0 {
		t.Fatal("aucune migration embarquée")
	}
	ups, downs := map[string]bool{}, map[string]bool{}
	for _, f := range fichiers {
		nom := strings.TrimPrefix(f, "migrations/")
		m := nomMigration.FindStringSubmatch(nom)
		if m == nil {
			t.Errorf("nom de migration non conforme : %s", nom)
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(nom, ".up.sql"), ".down.sql")
		if m[2] == "up" {
			ups[base] = true
		} else {
			downs[base] = true
		}
	}
	for base := range ups {
		if !downs[base] {
			t.Errorf("%s : migration « down » manquante", base)
		}
	}
	for base := range downs {
		if !ups[base] {
			t.Errorf("%s : migration « up » manquante", base)
		}
	}
}
