// Package cli implémente taches-cli, le client en ligne de commande de l'API
// des tâches : les commandes avec cobra, la configuration avec viper, et les
// appels HTTP avec le client généré depuis le contrat OpenAPI (pkg/client).
package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/client"
)

// Version est remplacée à la compilation :
//
//	go build -ldflags "-X .../internal/cli.Version=1.2.0" ./cmd/taches-cli
var Version = "dev"

// Options regroupe les dépendances de la ligne de commande, pour la tester.
type Options struct {
	Sortie  io.Writer
	Erreurs io.Writer
}

// app porte l'état partagé par les commandes : la configuration et le client.
type app struct {
	v      *viper.Viper
	sortie io.Writer
}

// NouvelleRacine construit la commande racine et toutes ses sous-commandes.
// Une nouvelle instance de viper est créée à chaque appel : pas d'état global,
// ce qui rend les tests indépendants les uns des autres.
func NouvelleRacine(o Options) *cobra.Command {
	a := &app{v: viper.New(), sortie: o.Sortie}

	racine := &cobra.Command{
		Use:   "taches-cli",
		Short: "Gère les tâches de l'API des tâches",
		Long: `taches-cli est le client en ligne de commande de l'API des tâches.

La configuration se lit, par ordre de priorité décroissante : dans les options
de la ligne de commande, dans les variables d'environnement TACHES_*, puis dans
le fichier de configuration (par défaut ~/.config/taches/config.yaml).`,
		Version:       Version,
		SilenceUsage:  true, // pas d'aide affichée pour une erreur d'exécution
		SilenceErrors: true, // main affiche l'erreur lui-même
		// Lien avec viper : avant chaque commande, la configuration est lue.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return a.configure(cmd) },
	}
	racine.SetOut(o.Sortie)
	racine.SetErr(o.Erreurs)

	// Options globales, disponibles pour toutes les sous-commandes.
	f := racine.PersistentFlags()
	f.String("config", "", "fichier de configuration (défaut ~/.config/taches/config.yaml)")
	f.String("serveur", "http://localhost:8080", "adresse de l'API")
	f.Duration("delai", 10*time.Second, "délai maximal d'une requête")
	f.StringP("format", "f", "table", "format de sortie : table ou json")

	racine.AddCommand(
		a.commandeListe(),
		a.commandeAjoute(),
		a.commandeFait(),
		a.commandeSupprime(),
	)
	return racine
}

// configure prépare viper : valeurs par défaut des options, variables
// d'environnement et fichier de configuration.
func (a *app) configure(cmd *cobra.Command) error {
	v := a.v
	// 1. Les options analysées par cobra, prioritaires.
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	// 2. Les variables d'environnement : TACHES_SERVEUR, TACHES_DELAI, TACHES_FORMAT…
	v.SetEnvPrefix("TACHES")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// 3. Le fichier de configuration, facultatif.
	if fichier := v.GetString("config"); fichier != "" {
		v.SetConfigFile(fichier) // fichier explicite : il doit exister
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("configuration : %w", err)
		}
		return nil
	}
	if dir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(dir, "taches"))
	}
	v.SetConfigName("config") // config.yaml, config.toml, config.json…
	if err := v.ReadInConfig(); err != nil {
		if _, absent := errors.AsType[viper.ConfigFileNotFoundError](err); !absent {
			return fmt.Errorf("configuration : %w", err)
		}
	}
	// 4. Les valeurs par défaut sont celles des options déclarées sur la racine.
	return nil
}

// client crée le client HTTP de l'API, configuré par viper.
func (a *app) client() (*client.ClientWithResponses, error) {
	return client.NewClientWithResponses(a.v.GetString("serveur"),
		client.WithHTTPClient(&http.Client{Timeout: a.v.GetDuration("delai")}))
}

// erreurAPI traduit une réponse d'erreur de l'API en erreur Go.
func erreurAPI(r *http.Response, corps []byte) error {
	if r.StatusCode < 400 {
		return nil
	}
	var e client.Erreur
	if err := jsonDecode(corps, &e); err == nil && e.Erreur != "" {
		return fmt.Errorf("%s (HTTP %d)", e.Erreur, r.StatusCode)
	}
	return fmt.Errorf("réponse inattendue : %s", r.Status)
}
