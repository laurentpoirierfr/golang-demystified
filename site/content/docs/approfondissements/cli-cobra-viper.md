---
title: "Ligne de commande : cobra et viper"
weight: 33
---

Go excelle pour les outils en ligne de commande : un exécutable unique,
statique, rapide à démarrer, compilable pour toutes les plateformes. Docker,
Kubernetes (`kubectl`), Hugo, GitHub (`gh`) ou Terraform sont écrits en Go,
et la plupart utilisent deux bibliothèques de Steve Francia :

- **[cobra](https://cobra.dev)** structure une application en **commandes et
  sous-commandes** (`git commit`, `kubectl get pods`), avec leurs options, la
  validation des arguments, l'aide et la complétion automatique du shell ;
- **[viper](https://github.com/spf13/viper)** gère la **configuration** :
  fichiers (YAML, TOML, JSON…), variables d'environnement, options de la ligne
  de commande et valeurs par défaut, avec un ordre de priorité clair.

Ce chapitre construit `taches-cli`, le client en ligne de commande de l'API du
chapitre « Une API web complète ». Le code est dans
`examples/approfondissements/api` : un second exécutable dans `cmd/`, la
logique dans `internal/cli`, et le client HTTP **généré depuis le contrat
OpenAPI** dans `pkg/client`.

```
$ taches-cli ajoute Relire le chapitre
Tâche 1 créée : Relire le chapitre
$ taches-cli fait 1
Tâche 1 faite : Relire le chapitre
$ taches-cli liste
ID  FAITE  TITRE
1   [x]    Relire le chapitre
```

> [!NOTE]
> Pour un outil simple, à une seule commande, le package `flag` de la
> bibliothèque standard suffit (voir le chapitre « Tour d'horizon »). cobra se
> justifie dès qu'il y a plusieurs sous-commandes.

## Le client HTTP, généré depuis le contrat

Le contrat OpenAPI de l'API décrit déjà toutes les opérations : oapi-codegen
génère aussi le **client**. Une seconde configuration le place dans
`pkg/client`, car c'est précisément du code destiné à être importé par
d'autres projets :

```yaml
# api/oapi-codegen-client.yaml
package: client
output: ../pkg/client/client.gen.go
generate:
  models: true
  client: true
```

```go
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
//go:generate go tool oapi-codegen -config oapi-codegen-client.yaml openapi.yaml
```

Le client expose une méthode par opération, dont la réponse est décodée selon
le statut HTTP :

```go
c, err := client.NewClientWithResponses("http://localhost:8080",
	client.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))

r, err := c.CreeTacheWithResponse(ctx, client.NouvelleTache{Titre: "Relire"})
// r.JSON201 *Tache, r.JSON400 *Erreur, r.JSON422 *Erreur…
```

Le serveur et le client sont générés depuis le **même** contrat : ils ne
peuvent pas diverger.

## cobra : les commandes

### La commande racine

Une application cobra est un arbre de `*cobra.Command`. La racine porte le
nom du programme, sa description et les **options globales**, héritées par
toutes les sous-commandes (`PersistentFlags`) :

```go
func NouvelleRacine(o Options) *cobra.Command {
	a := &app{v: viper.New(), sortie: o.Sortie}

	racine := &cobra.Command{
		Use:               "taches-cli",
		Short:             "Gère les tâches de l'API des tâches",
		Long:              `taches-cli est le client en ligne de commande de l'API des tâches. …`,
		Version:           Version,
		SilenceUsage:      true, // pas d'aide affichée pour une erreur d'exécution
		SilenceErrors:     true, // main affiche l'erreur lui-même
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return a.configure(cmd) },
	}
	racine.SetOut(o.Sortie)
	racine.SetErr(o.Erreurs)

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
```

Quelques choix méritent d'être expliqués :

- la racine est construite par une **fonction**, et non stockée dans une
  variable globale initialisée par `init()` comme dans de nombreux exemples.
  Chaque test crée ainsi sa propre racine, sans état partagé ;
- les sorties passent par `SetOut` et `SetErr`, que les tests remplacent par un
  tampon ;
- `SilenceUsage` évite d'afficher toute l'aide quand une commande échoue pour
  une raison qui n'a rien à voir avec son usage (un serveur injoignable, par
  exemple) ;
- `Version` ajoute automatiquement l'option `--version`. La valeur est fixée à
  la compilation : `go build -ldflags "-X …/internal/cli.Version=1.2.0"`.

### Les sous-commandes

Chaque sous-commande déclare son usage, la validation de ses arguments et sa
fonction `RunE`, qui renvoie une erreur plutôt que de quitter le programme :

```go
func (a *app) commandeFait() *cobra.Command {
	var annule bool
	cmd := &cobra.Command{
		Use:               "fait <id>",
		Short:             "Marque une tâche comme faite",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: a.completeIdentifiants,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := identifiant(args[0])
			if err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			r, err := c.ModifieTacheWithResponse(cmd.Context(), id, client.ModificationTache{Faite: !annule})
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tâche %d faite : %s\n", id, r.JSON200.Titre)
			return nil
		},
	}
	cmd.Flags().BoolVar(&annule, "annule", false, "remet la tâche à faire")
	return cmd
}
```

| Outil de cobra                          | Rôle                                               |
|-----------------------------------------|----------------------------------------------------|
| `Args: cobra.ExactArgs(1)`, `MinimumNArgs`, `NoArgs`… | valide le nombre d'arguments          |
| `Aliases: []string{"ls"}`               | noms alternatifs (`taches-cli ls`)                 |
| `Example`                               | exemples affichés dans l'aide                      |
| `cmd.Flags()` / `PersistentFlags()`     | options locales / héritées par les sous-commandes  |
| `MarkFlagRequired`, `MarkFlagsMutuallyExclusive` | contraintes entre options                 |
| `PersistentPreRunE`                     | préparation commune (configuration, connexion)     |
| `cmd.Context()`                         | le contexte passé à `ExecuteContext`               |

Le programme principal se résume à créer le contexte, exécuter la racine et
afficher l'erreur éventuelle :

```go
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	racine := cli.NouvelleRacine(cli.Options{Sortie: os.Stdout, Erreurs: os.Stderr})
	if err := racine.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}
```

### L'aide et la complétion

cobra génère l'aide de chaque commande (`taches-cli --help`,
`taches-cli fait --help`) et une commande `completion` qui produit le script
de complétion pour bash, zsh, fish et PowerShell :

```bash
taches-cli completion fish > ~/.config/fish/completions/taches-cli.fish
source <(taches-cli completion bash)
```

La complétion des commandes et des options est automatique. Pour les
arguments, `ValidArgsFunction` propose des valeurs dynamiques : ici, les
identifiants des tâches, avec leur titre en description, obtenus auprès de
l'API.

```go
func (a *app) completeIdentifiants(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, err := a.client()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	r, err := c.ListeTachesWithResponse(cmd.Context())
	if err != nil || r.JSON200 == nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var propositions []string
	for _, t := range *r.JSON200 {
		propositions = append(propositions, fmt.Sprintf("%d\t%s", t.Id, t.Titre))
	}
	return propositions, cobra.ShellCompDirectiveNoFileComp
}
```

```
$ taches-cli fait <TAB>
1  -- Relire le chapitre
2  -- Publier
```

> [!TIP]
> Les textes générés par cobra (« Available Commands », « Flags »…) sont en
> anglais. `SetUsageTemplate` et `SetHelpTemplate` permettent de les
> remplacer par des gabarits `text/template` traduits.

## viper : la configuration

### L'ordre de priorité

Un outil en ligne de commande lit sa configuration à plusieurs endroits. viper
les fusionne, par ordre de priorité décroissante :

1. les **options** de la ligne de commande (`--serveur …`) ;
2. les **variables d'environnement** (`TACHES_SERVEUR=…`) ;
3. le **fichier de configuration** (`~/.config/taches/config.yaml`) ;
4. les **valeurs par défaut**, ici celles des options.

```go
func (a *app) configure(cmd *cobra.Command) error {
	v := a.v
	// Les options déclarées sur la ligne de commande sont prioritaires.
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	// TACHES_SERVEUR, TACHES_DELAI, TACHES_FORMAT…
	v.SetEnvPrefix("TACHES")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

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
	return nil
}
```

Le fichier de configuration est facultatif : son absence n'est pas une erreur,
mais un fichier illisible en est une. `os.UserConfigDir` renvoie le répertoire
de configuration propre à chaque système : `~/.config` sous Linux,
`~/Library/Application Support` sous macOS, `%AppData%` sous Windows.

```yaml
# ~/.config/taches/config.yaml
serveur: https://taches.example.com
format: json
delai: 5s
```

Les commandes lisent ensuite les valeurs avec `v.GetString("serveur")`,
`v.GetDuration("delai")`, etc., sans se soucier de leur provenance.

### viper sans état global

La documentation de viper utilise souvent ses fonctions globales
(`viper.GetString`…), qui agissent sur une instance partagée par tout le
programme. `viper.New()` crée une instance indépendante : chaque racine a la
sienne, et les tests ne se perturbent pas les uns les autres.

> [!WARNING]
> viper apporte de nombreuses dépendances (formats de fichiers, surveillance
> des modifications, systèmes de configuration distants). Pour un service qui
> ne lit que quelques variables d'environnement, une fonction comme celle du
> chapitre « Une API web complète », avec `os.Getenv` et `cmp.Or`, est plus
> simple. viper se justifie pour un outil utilisé par des humains, qui
> attendent un fichier de configuration et des options.

## Tester la ligne de commande

Comme la racine est construite par une fonction et écrit dans un `io.Writer`,
une commande se teste comme une fonction ordinaire. Les tests démarrent la
**vraie API**, avec une base SQLite en mémoire : ils couvrent ensemble la ligne
de commande, le client généré et le serveur.

```go
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
	url := apiDeTest(t)                      // la vraie API, sur httptest
	etapes := []struct {
		args     []string
		contient string
	}{
		{[]string{"ajoute", "Relire", "le", "chapitre"}, "Tâche 1 créée : Relire le chapitre"},
		{[]string{"fait", "1"}, "Tâche 1 faite : Relire le chapitre"},
		{[]string{"liste"}, "1   [x]    Relire le chapitre"},
		{[]string{"ls", "-f", "json"}, `"titre": "Relire le chapitre"`},
		…
	}
	for _, e := range etapes {
		sortie, err := lance(t, append(e.args, "--serveur", url)...)
		…
	}
}
```

Les erreurs se testent de la même façon, qu'elles viennent de cobra
(arguments manquants, options incompatibles), de la commande ou de l'API :

```go
{[]string{"fait", "42"}, "tâche introuvable (HTTP 404)"},
{[]string{"ajoute", " "}, "titre : obligatoire (HTTP 422)"},
{[]string{"liste", "--faites", "--a-faire"}, "none of the others can be"},
{[]string{"supprime"}, "accepts 1 arg(s), received 0"},
```

Un test vérifie l'ordre de priorité de viper, en écrivant un fichier de
configuration dans un répertoire temporaire puis en le contredisant par une
variable d'environnement (`t.Setenv`) et par une option. Enfin, la complétion
se teste avec la commande cachée `__complete`, celle que le shell appelle
pendant la saisie :

```go
sortie, err := lance(t, "__complete", "fait", "--serveur", url, "")
// 1	Écrire
// :4
```

## Distribuer l'outil

Un outil Go se distribue sous la forme d'un exécutable par plateforme :

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/taches-cli-linux-amd64 ./cmd/taches-cli
GOOS=darwin  GOARCH=arm64 go build -o dist/taches-cli-darwin-arm64 ./cmd/taches-cli
GOOS=windows GOARCH=amd64 go build -o dist/taches-cli.exe ./cmd/taches-cli
```

Pour un projet publié, [GoReleaser](https://goreleaser.com) automatise ces
compilations, les archives, les sommes de contrôle, les paquets (Homebrew,
deb, rpm) et la publication sur GitHub à chaque tag. Les utilisateurs qui ont
Go peuvent aussi installer l'outil directement :

```bash
go install github.com/utilisateur/projet/cmd/taches-cli@latest
```
