// Package config lit la configuration du service dans l'environnement.
package config

import (
	"cmp"
	"fmt"
	"log/slog"
	"time"
)

// Config regroupe les réglages du service.
type Config struct {
	Adresse       string        // TACHES_ADRESSE, par défaut ":8080"
	Base          string        // TACHES_BASE, par défaut "file:taches.db"
	DelaiArret    time.Duration // TACHES_DELAI_ARRET, par défaut 10s
	NiveauJournal slog.Level    // TACHES_JOURNAL : debug, info, warn, error
}

// Charge lit la configuration avec getenv (os.Getenv en production, une map
// dans les tests).
func Charge(getenv func(string) string) (Config, error) {
	c := Config{
		Adresse: cmp.Or(getenv("TACHES_ADRESSE"), ":8080"),
		Base:    cmp.Or(getenv("TACHES_BASE"), "file:taches.db"),
	}

	delai, err := time.ParseDuration(cmp.Or(getenv("TACHES_DELAI_ARRET"), "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("TACHES_DELAI_ARRET : %w", err)
	}
	c.DelaiArret = delai

	if err := c.NiveauJournal.UnmarshalText([]byte(cmp.Or(getenv("TACHES_JOURNAL"), "info"))); err != nil {
		return Config{}, fmt.Errorf("TACHES_JOURNAL : %w", err)
	}
	return c, nil
}
