---
title: "V.5 Ligne de commande"
weight: 50
---

# Chapitre V.5 — Ligne de commande : cobra et viper

Le chapitre « Une API web complète » a construit un service HTTP. Ce chapitre
construit l'outil qui permet de le piloter depuis un terminal : **`taches-cli`**.

```
$ taches-cli ajoute Relire le chapitre
Tâche 1 créée : Relire le chapitre
$ taches-cli fait 1
Tâche 1 faite : Relire le chapitre
$ taches-cli liste
ID  FAITE  TITRE
1   [x]    Relire le chapitre
```

Go est particulièrement adapté à ce genre d'outil : un exécutable unique,
statique, qui démarre instantanément et se compile pour toutes les
plateformes. [Docker](https://www.docker.com), [Kubernetes](https://kubernetes.io)
(`kubectl`), [Hugo](https://gohugo.io) ou [GitHub CLI](https://cli.github.com)
(`gh`) sont des outils en ligne de commande écrits en Go.

Le code est dans [`examples/approfondissements/api`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/api) : l'exécutable dans
`cmd/taches-cli`, et les commandes dans `internal/cli`.

## Trois questions, trois briques

Toute commande tapée par l'utilisateur, par exemple `taches-cli fait 1`,
pose trois questions au programme. Chacune a sa brique :

| Question                                             | Brique                    | Rôle                                        |
|------------------------------------------------------|---------------------------|---------------------------------------------|
| **Que demande l'utilisateur ?** Quelle commande, quels arguments, quelles options ? | [cobra](https://cobra.dev) | analyse la ligne de commande               |
| **Avec quels réglages ?** Quel serveur, quel délai, quel format d'affichage ? | [viper](https://github.com/spf13/viper) | rassemble la configuration     |
| **Comment parler à l'API ?**                         | le client généré depuis le contrat [OpenAPI](https://www.openapis.org) | envoie la requête HTTP |

Le reste du chapitre suit le trajet d'une commande, dans cet ordre :

```
taches-cli fait 1 --format json
│
├─ 1. cobra      reconnaît la commande « fait », l'argument « 1 », l'option --format
├─ 2. viper      complète les réglages : --format vient de l'option,
│                le serveur de TACHES_SERVEUR ou du fichier de configuration
├─ 3. client     envoie PATCH /taches/1 {"faite": true} au serveur
└─ 4. cobra      affiche le résultat, ou l'erreur
```

Les trois briques sont indépendantes : cobra ne sait rien de la
configuration, viper ne sait rien des commandes, et le client ne sait rien
des deux. Le code de `taches-cli` est la colle qui les relie. Le lien est
fait à deux endroits seulement, qu'il faut repérer :

- la commande racine **demande à viper** de lire la configuration avant chaque
  commande (étape 2) ;
- chaque commande **crée le client** à partir des réglages de viper (étape 3).

## Étape 1 — Que demande l'utilisateur ? cobra

Sans bibliothèque, il faudrait découper `os.Args` à la main pour reconnaître
`fait`, puis `1`, puis `--format json`. Le package `flag` de la bibliothèque
standard sait lire des options, mais pas des **sous-commandes**. cobra
structure l'application en un arbre de commandes :

```
taches-cli                     la racine : options globales
├── liste    (alias ls)        --faites, --a-faire
├── ajoute <titre>…
├── fait <id>                  --annule
├── supprime <id> (alias rm)
└── completion                 ajoutée par cobra
```

### La racine et les options globales

La racine porte le nom du programme et les options **globales**, valables
pour toutes les sous-commandes (`PersistentFlags`). Ce sont précisément les
réglages de l'étape 2 :

```go
func NouvelleRacine(o Options) *cobra.Command {
	a := &app{v: viper.New(), sortie: o.Sortie}

	racine := &cobra.Command{
		Use:           "taches-cli",
		Short:         "Gère les tâches de l'API des tâches",
		Version:       Version,
		SilenceUsage:  true, // pas d'aide affichée pour une erreur d'exécution
		SilenceErrors: true, // main affiche l'erreur lui-même
		// Lien avec l'étape 2 : avant chaque commande, viper lit la configuration.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return a.configure(cmd) },
	}
	racine.SetOut(o.Sortie)
	racine.SetErr(o.Erreurs)

	f := racine.PersistentFlags()
	f.String("config", "", "fichier de configuration (défaut ~/.config/taches/config.yaml)")
	f.String("serveur", "http://localhost:8080", "adresse de l'API")
	f.Duration("delai", 10*time.Second, "délai maximal d'une requête")
	f.StringP("format", "f", "table", "format de sortie : table ou json")

	racine.AddCommand(a.commandeListe(), a.commandeAjoute(), a.commandeFait(), a.commandeSupprime())
	return racine
}
```

Le type `app` est le point de rencontre des trois briques : il porte
l'instance de viper (les réglages) et la sortie, et chaque commande est une de
ses méthodes.

Quelques choix méritent une explication :

- la racine est construite par une **fonction**, et non dans une variable
  globale initialisée par `init()` comme dans beaucoup d'exemples : chaque
  test peut créer sa propre racine, sans état partagé ;
- les sorties passent par `SetOut` et `SetErr`, que les tests remplacent par
  un tampon ;
- `SilenceUsage` évite d'afficher toute l'aide quand l'erreur n'a rien à voir
  avec l'usage, par exemple un serveur injoignable ;
- `Version` ajoute l'option `--version`, dont la valeur est fixée à la
  compilation : `go build -ldflags "-X …/internal/cli.Version=1.2.0"`.

### Une sous-commande

Une sous-commande déclare son usage, la validation de ses arguments, ses
options propres, et sa fonction `RunE`. `RunE` renvoie une erreur au lieu de
quitter le programme, ce qui la rend testable :

```go
func (a *app) commandeListe() *cobra.Command {
	var faites, aFaire bool
	cmd := &cobra.Command{
		Use:     "liste",
		Aliases: []string{"ls"},
		Short:   "Liste les tâches",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client() // étape 3 : le client, configuré par viper
			if err != nil {
				return err
			}
			r, err := c.ListeTachesWithResponse(cmd.Context())
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			taches := []client.Tache{}
			for _, t := range *r.JSON200 {
				if (faites && !t.Faite) || (aFaire && t.Faite) {
					continue
				}
				taches = append(taches, t)
			}
			return a.affiche(taches) // étape 4 : tableau ou JSON, selon viper
		},
	}
	cmd.Flags().BoolVar(&faites, "faites", false, "seulement les tâches faites")
	cmd.Flags().BoolVar(&aFaire, "a-faire", false, "seulement les tâches à faire")
	cmd.MarkFlagsMutuallyExclusive("faites", "a-faire")
	return cmd
}
```

Cette commande montre déjà les trois briques au travail : cobra a validé les
arguments et les options, `a.client()` utilise les réglages de viper, et le
client généré appelle l'API.

| Outil de cobra                                  | Rôle                                              |
|-------------------------------------------------|---------------------------------------------------|
| `Args: cobra.ExactArgs(1)`, `MinimumNArgs`, `NoArgs` | valide le nombre d'arguments                  |
| `Aliases: []string{"ls"}`                       | noms alternatifs                                  |
| `Example`                                       | exemples affichés dans l'aide                     |
| `cmd.Flags()` / `PersistentFlags()`             | options locales / héritées par les sous-commandes |
| `MarkFlagRequired`, `MarkFlagsMutuallyExclusive` | contraintes entre options                        |
| `PersistentPreRunE`                             | préparation commune, avant chaque commande        |
| `cmd.Context()`                                 | le contexte passé à `ExecuteContext`              |

Le programme principal se contente d'exécuter la racine :

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

À ce stade, cobra sait **ce que** l'utilisateur demande. Mais pour l'exécuter,
la commande a besoin de savoir **où** se trouve le serveur. C'est la question
de l'étape 2.

## Étape 2 — Avec quels réglages ? viper

L'adresse du serveur peut venir de plusieurs endroits, et l'utilisateur
s'attend à un comportement précis :

- il la tape une fois dans un **fichier de configuration**, et l'oublie ;
- dans un script ou un conteneur, il la fournit par une **variable
  d'environnement** ;
- ponctuellement, il la change par une **option** : `--serveur …` ;
- sans rien de tout cela, une **valeur par défaut** s'applique.

viper rassemble ces sources, avec un ordre de priorité décroissant : option,
variable d'environnement, fichier, valeur par défaut. Les commandes n'ont plus
qu'à demander `v.GetString("serveur")`, sans se soucier de la provenance de la
valeur.

### Le lien avec cobra

C'est ici que les deux briques se rejoignent. Avant chaque commande, le
`PersistentPreRunE` de la racine appelle `configure`, qui donne à viper les
options analysées par cobra (`BindPFlags`), puis les variables
d'environnement, puis le fichier :

```go
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
	// 4. Les valeurs par défaut sont celles des options déclarées à l'étape 1.
	return nil
}
```

Un même réglage peut donc s'écrire de trois façons, qui aboutissent toutes à
`v.GetString("serveur")` :

```bash
# ~/.config/taches/config.yaml
serveur: https://taches.example.com

# variable d'environnement
TACHES_SERVEUR=https://taches.example.com taches-cli liste

# option
taches-cli liste --serveur https://taches.example.com
```

`os.UserConfigDir` renvoie le répertoire de configuration propre à chaque
système : `~/.config` sous Linux, `~/Library/Application Support` sous macOS,
`%AppData%` sous Windows.

> [!TIP]
> La documentation de viper utilise souvent ses fonctions globales
> (`viper.GetString`…), qui agissent sur une instance partagée par tout le
> programme. `viper.New()` crée une instance indépendante : chaque racine a la
> sienne, et les tests ne se perturbent pas les uns les autres.

La commande sait maintenant **ce que** l'utilisateur demande et **où** se
trouve le serveur. Reste à lui envoyer la requête.

## Étape 3 — Comment parler à l'API ? Le client généré

Écrire le client HTTP à la main serait possible : construire l'URL, encoder le
JSON, vérifier le statut, décoder la réponse… pour chaque opération. Mais
l'API dispose déjà d'un **contrat OpenAPI**, à partir duquel
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) génère le
serveur. Il génère aussi le **client** : une méthode Go par opération, avec des
types qui ne peuvent pas diverger de ceux du serveur.

```yaml
# api/oapi-codegen-client.yaml
package: client
output: ../pkg/client/client.gen.go
generate:
  models: true
  client: true
```

Le client est placé dans `pkg/client`, car c'est du code destiné à être
importé par d'autres projets : tout programme Go qui veut appeler l'API des
tâches peut l'utiliser.

### Le lien avec viper

Chaque commande crée son client à partir des réglages de viper. C'est la
seconde jonction entre les briques :

```go
func (a *app) client() (*client.ClientWithResponses, error) {
	return client.NewClientWithResponses(a.v.GetString("serveur"),
		client.WithHTTPClient(&http.Client{Timeout: a.v.GetDuration("delai")}))
}
```

Le client renvoie une réponse dont les champs sont décodés selon le statut
HTTP : `JSON200`, `JSON201`, `JSON404`… Une petite fonction transforme les
réponses d'erreur de l'API en erreurs Go lisibles :

```go
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
```

```
$ taches-cli fait 42
erreur : tâche introuvable (HTTP 404)
```

## Étape 4 — Les autres commandes

Avec ces trois briques en place, chaque nouvelle commande suit le même schéma :
valider les arguments (cobra), créer le client (viper), appeler l'opération
(client), afficher le résultat. La commande `fait` ajoute seulement une option
`--annule` :

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

### La complétion : les trois briques ensemble

La complétion du shell illustre bien la collaboration des trois briques.
cobra génère le script de complétion pour bash, zsh, fish et PowerShell :

```bash
taches-cli completion fish > ~/.config/fish/completions/taches-cli.fish
source <(taches-cli completion bash)
```

Les commandes et les options sont complétées automatiquement. Pour
l'argument de `fait`, `ValidArgsFunction` propose les identifiants des tâches
existantes : cobra demande les propositions, viper fournit l'adresse du
serveur (le fichier de configuration est lu, même pendant la complétion), et
le client les obtient de l'API :

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

## Tester l'ensemble

Comme les briques sont reliées en deux points seulement, l'ensemble se teste
facilement, sans simulation : les tests démarrent la **vraie API** sur un
serveur `httptest`, avec une base [SQLite](https://sqlite.org) en mémoire, et
exécutent `taches-cli` comme une fonction. Ils couvrent donc le trajet complet
d'une commande : cobra, viper, le client généré et le serveur.

```go
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

Chaque brique a ensuite ses tests ciblés :

| Brique    | Ce que l'on vérifie                         | Exemple                                                        |
|-----------|---------------------------------------------|----------------------------------------------------------------|
| cobra     | arguments et options invalides              | `supprime` sans identifiant → `accepts 1 arg(s), received 0`   |
| viper     | l'ordre de priorité des réglages            | un fichier, contredit par `TACHES_FORMAT`, puis par `--format` |
| client    | la traduction des erreurs de l'API          | `fait 42` → `tâche introuvable (HTTP 404)`                     |
| les trois | la complétion                               | la commande cachée `__complete fait ""` propose `1  Écrire`    |

Le test de priorité de viper écrit un fichier de configuration dans un
répertoire temporaire (`XDG_CONFIG_HOME`), puis le contredit par une variable
d'environnement (`t.Setenv`), puis par une option : chaque niveau doit
l'emporter sur le précédent.

## Distribuer l'outil

Un outil Go se distribue sous la forme d'un exécutable par plateforme :

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/taches-cli-linux-amd64 ./cmd/taches-cli
GOOS=darwin  GOARCH=arm64 go build -o dist/taches-cli-darwin-arm64 ./cmd/taches-cli
GOOS=windows GOARCH=amd64 go build -o dist/taches-cli.exe ./cmd/taches-cli
```

Pour un projet publié, [GoReleaser](https://goreleaser.com) automatise ces
compilations, les archives, les sommes de contrôle, les paquets
([Homebrew](https://brew.sh), deb, rpm) et la publication sur GitHub à chaque
tag. Les utilisateurs qui ont Go peuvent aussi installer l'outil directement :

```bash
go install github.com/utilisateur/projet/cmd/taches-cli@latest
```

## Quand utiliser cobra et viper ?

| Besoin                                              | Outil                                      |
|-----------------------------------------------------|--------------------------------------------|
| Un outil à une seule commande, quelques options     | le package `flag` de la bibliothèque standard |
| Plusieurs sous-commandes, aide, complétion          | cobra                                      |
| Un service configuré par quelques variables d'environnement | `os.Getenv` et `cmp.Or` (chapitre « Une API web complète ») |
| Un outil utilisé par des humains : fichier de configuration, variables, options | viper |

viper apporte de nombreuses dépendances (formats de fichiers, surveillance des
modifications, configurations distantes). Il se justifie pour un outil qui a
réellement besoin de plusieurs sources de configuration. cobra, lui, est léger
et s'utilise très bien seul, avec le package `flag` ou des variables
d'environnement lues à la main.
