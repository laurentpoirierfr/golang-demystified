---
title: "V.1 Go idiomatique"
description: "options fonctionnelles, middlewares, injection de dépendances, anti-motifs et grille de relecture."
weight: 10
---

# Chapitre V.1 — Go idiomatique : motifs, anti-motifs et relecture

Un programme Go peut compiler, passer ses tests et rester **non
idiomatique** : écrit comme du Java, du Python ou du C, avec les détours que
ces langages imposent et que Go rend inutiles. Le code idiomatique est celui
qu'un développeur Go lit sans effort, parce qu'il suit les conventions de la
communauté. Elles tiennent en quelques proverbes de Rob Pike :

- *Clear is better than clever* : le clair vaut mieux que l'astucieux ;
- *The bigger the interface, the weaker the abstraction* : plus une interface
  est grande, plus l'abstraction est faible ;
- *Make the zero value useful* : la valeur zéro doit être utilisable ;
- *Errors are values* : une erreur est une valeur, qu'on examine et qu'on
  enrichit ;
- *Don't communicate by sharing memory, share memory by communicating*.

Ce chapitre ouvre le parcours des approfondissements : les motifs qu'il
présente reviennent dans tous les suivants. Il se termine par une grille de
relecture de code.

Le code est dans [`examples/approfondissements/idiomatique`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/idiomatique).

## Motif : les options fonctionnelles

Un constructeur a souvent quelques paramètres obligatoires et beaucoup de
réglages facultatifs. Go n'a ni paramètres nommés ni valeurs par défaut. Les
**options fonctionnelles** y répondent : chaque réglage est une fonction qui
modifie l'objet en construction.

```go
// Option règle un Client. Elle peut échouer : une valeur invalide est
// signalée à la construction, pas à la première requête.
type Option func(*Client) error

func AvecDelai(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return fmt.Errorf("délai invalide : %v", d)
		}
		c.http.Timeout = d
		return nil
	}
}

func Nouveau(base string, options ...Option) (*Client, error) {
	…
	c := &Client{ // les valeurs par défaut d'abord
		base:   u,
		http:   &http.Client{Timeout: 10 * time.Second},
		essais: 1,
		…
	}
	var errs []error
	for _, o := range options {
		errs = append(errs, o(c))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return c, nil
}
```

L'appel le plus simple reste simple, et les réglages se lisent comme une
phrase :

```go
c, err := client.Nouveau("https://api.exemple.org",
	client.AvecDelai(2*time.Second),
	client.AvecEssais(3),
)
```

Toutes les options invalides sont signalées ensemble, grâce à `errors.Join` :

```console
délai invalide : 0s
nombre d'essais invalide : 42 (de 1 à 10)
```

> [!TIP]
> Les options fonctionnelles ne sont pas toujours le meilleur choix. Une
> simple **structure de configuration** (`Nouveau(Config{Delai: 2 *
> time.Second})`) est plus directe quand les réglages sont lus depuis un
> fichier, et sa valeur zéro peut servir de valeur par défaut. Les options
> brillent dans une bibliothèque, où elles s'ajoutent sans casser les appels
> existants.

## Motif : le middleware

Le motif du middleware découle de l'interface `http.Handler`, sans aucun
framework : une fonction reçoit un handler et renvoie un handler qui
l'enveloppe.

```go
type Intergiciel func(http.Handler) http.Handler

// Chaine applique les intergiciels à h. Le premier de la liste est le plus
// extérieur : il voit la requête en premier, et la réponse en dernier.
func Chaine(h http.Handler, intergiciels ...Intergiciel) http.Handler {
	for _, m := range slices.Backward(intergiciels) {
		h = m(h)
	}
	return h
}
```

Un test trace le passage dans chaque couche :

```console
→ A → B handler ← B ← A
```

L'**ordre** compte. Avec `Chaine(mux, Recupere(log), Journal(log))`, la
récupération des paniques est à l'extérieur : une panique dans un handler
devient une réponse 500, mais `Journal`, traversé par la panique, ne note pas
la requête. Le test le vérifie.

Le même motif sert partout où une interface n'a qu'une méthode :
`http.RoundTripper` côté client, les intercepteurs de gRPC et de Connect,
les `slog.Handler` qui enrichissent un journal.

## Motif : l'injection de dépendances

Go n'a pas besoin d'un conteneur d'injection de dépendances. Un service
**déclare** les petites interfaces dont il a besoin, et **reçoit** leurs
implémentations à sa construction :

```go
// Depot stocke les membres. L'interface est déclarée ici, par le code qui
// l'utilise, et ne contient que ce dont il a besoin : deux méthodes, et non
// les vingt d'un dépôt complet.
type Depot interface {
	Existe(ctx context.Context, courriel string) (bool, error)
	Enregistre(ctx context.Context, m Membre) error
}

type Courrier interface {
	Envoie(ctx context.Context, a, sujet, corps string) error
}

type Service struct {
	depot      Depot
	courrier   Courrier
	maintenant func() time.Time
}

func Nouveau(depot Depot, courrier Courrier, maintenant func() time.Time) *Service {
	return &Service{depot: depot, courrier: courrier, maintenant: maintenant}
}
```

Quatre idées se combinent ici :

- **l'interface appartient à son utilisateur**, pas à l'implémentation : le
  package `inscription` déclare `Depot`, et un dépôt PostgreSQL la satisfait
  implicitement, sans la connaître ;
- **accepter des interfaces, renvoyer des structures** : `Nouveau` reçoit des
  interfaces et renvoie un `*Service` concret ;
- **même l'heure est une dépendance** : `maintenant func() time.Time` rend le
  service testable sans attendre ni truquer l'horloge ;
- **le programme principal assemble** : c'est le seul endroit qui connaît
  les implémentations concrètes.

Les tests utilisent des doublures écrites à la main, en quelques lignes, sans
bibliothèque de *mocks* :

```go
type courrierEspion struct {
	envoyes []string
	panne   error
}

func (c *courrierEspion) Envoie(_ context.Context, a, sujet, _ string) error {
	c.envoyes = append(c.envoyes, a+" : "+sujet)
	return c.panne
}
```

### Câbler à la main, Wire ou Fx ?

Dans l'industrie, l'outil d'injection de dépendances le plus utilisé en Go
est… **aucun outil**. Kubernetes, Docker, Terraform ou la bibliothèque
standard assemblent leurs composants à la main. Deux bibliothèques
reviennent pourtant souvent : **Wire**, de Google, et **Fx**, d'Uber. Pour les
comparer, l'exemple
[`idiomatique/cmd`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/idiomatique/cmd)
assemble trois fois le même graphe : le service d'inscription, un dépôt en
mémoire, un courrier qui journalise, et un serveur HTTP qui expose
`POST /membres`. Les trois programmes passent le même test, et répondent de
la même façon :

```console
$ curl -X POST localhost:8080/membres -d '{"courriel":"Ada@Exemple.fr","nom":"Ada"}'
{"courriel":"ada@exemple.fr","nom":"Ada"}          (201 ; 409 la seconde fois)
```

#### À la main : la norme

Tout le câblage tient dans une fonction ordinaire, appelée par `main` :

```go
func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	depot := infra.NouveauDepotMemoire()
	courrier := infra.NouveauCourrierJournal(log)
	service := inscription.Nouveau(depot, courrier, time.Now)
	return web.NouveauServeur(cfg, web.Routes(service))
}
```

C'est explicite : on lit l'ordre de construction de haut en bas, l'éditeur
navigue d'un constructeur à l'autre, et **le compilateur vérifie tout** : un
argument oublié ou du mauvais type est une erreur de compilation. Le cycle
de vie s'écrit aussi à la main : démarrer le serveur, attendre un signal
(`signal.NotifyContext`), puis `srv.Shutdown`. Quand le graphe grandit,
`assemble` grandit avec lui ; découpée en quelques fonctions (`assembleDonnees`,
`assembleHTTP`…), elle reste lisible bien au-delà de ce que l'on imagine.

#### Wire : générer le même code

[Wire](https://github.com/google/wire) **génère** l'assemblage à la
compilation. On décrit le graphe dans un fichier marqué `wireinject`, jamais
compilé dans le programme :

```go
//go:build wireinject

var ensembleInfra = wire.NewSet(
	infra.NouveauDepotMemoire,
	wire.Bind(new(inscription.Depot), new(*infra.DepotMemoire)),
	infra.NouveauCourrierJournal,
	wire.Bind(new(inscription.Courrier), new(*infra.CourrierJournal)),
)

func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	wire.Build(ensembleInfra, horloge, inscription.Nouveau,
		wire.Bind(new(web.Inscripteur), new(*inscription.Service)),
		web.Routes, web.NouveauServeur)
	return nil // remplacé par le code généré
}
```

`go tool wire` (ou `go generate`) écrit alors `wire_gen.go`, qui est
**exactement** l'assemblage manuel, au nom des variables près :

```go
func assemble(cfg web.Config, log *slog.Logger) *http.Server {
	depotMemoire := infra.NouveauDepotMemoire()
	courrierJournal := infra.NouveauCourrierJournal(log)
	v := horloge()
	service := inscription.Nouveau(depotMemoire, courrierJournal, v)
	handler := web.Routes(service)
	server := web.NouveauServeur(cfg, handler)
	return server
}
```

Pas de réflexion, pas de conteneur à l'exécution : un graphe incomplet est
refusé **avant la compilation du programme**, avec la chaîne de dépendances
qui y mène :

```console
$ go tool wire check ./testdata/incomplet
wire: …/incomplet/wire.go:17:1: inject assemble: no provider found for func() time.Time
	needed by *…/inscription.Service in provider "Nouveau" (…)
	needed by …/web.Inscripteur in wire.Bind (…)
	needed by net/http.Handler in provider "Routes" (…)
	needed by *net/http.Server in provider "NouveauServeur" (…)
```

Wire ne s'occupe que de la construction : le cycle de vie reste à écrire.
Deux contraintes : un fournisseur est choisi **par son type**, d'où la
fonction `horloge` (un `func() time.Time`) et les `wire.Bind` pour chaque
interface ; et le code généré doit être versionné et tenu à jour (la CI de ce
dépôt le vérifie).

> [!WARNING]
> Google a **archivé** le dépôt de Wire le 25 août 2025 : il n'est plus
> maintenu. La dernière version fonctionne, et le code qu'elle a généré reste
> du Go ordinaire, sans dépendance à Wire à l'exécution. Des forks existent,
> mais pour un nouveau projet, Wire n'est plus un choix sûr.

#### Fx : un conteneur, et un cycle de vie

[Fx](https://github.com/uber-go/fx), construit sur dig, résout le graphe **à
l'exécution**, par réflexion. On lui donne des constructeurs, dans n'importe
quel ordre ; il les appelle quand leur résultat est demandé, une seule fois
chacun :

```go
var module = fx.Options(
	fx.Provide(
		fx.Annotate(infra.NouveauDepotMemoire, fx.As(new(inscription.Depot))),
		fx.Annotate(infra.NouveauCourrierJournal, fx.As(new(inscription.Courrier))),
		func() func() time.Time { return time.Now },
		fx.Annotate(inscription.Nouveau, fx.As(new(web.Inscripteur))),
		web.Routes,
		web.NouveauServeur,
	),
	fx.Invoke(demarre), // demande le serveur : tout le graphe est construit
)
```

Sa force est le **cycle de vie**. Chaque composant accroche ses fonctions de
démarrage et d'arrêt ; Fx les appelle dans l'ordre, puis dans l'ordre inverse
à l'arrêt, et `Run` attend lui-même `SIGINT` ou `SIGTERM` :

```go
func demarre(lc fx.Lifecycle, srv *http.Server, log *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr) // une erreur arrête le démarrage
			if err != nil {
				return err
			}
			go srv.Serve(ln)
			return nil
		},
		OnStop: srv.Shutdown,
	})
}

func main() {
	fx.New(fx.Supply(cfg, log), module).Run()
}
```

Le prix : une dépendance manquante n'est découverte **qu'à l'exécution**,
quand Fx construit l'application. Le compilateur ne voit rien ; le programme
échoue au démarrage :

```console
missing dependencies for function "…/cmd/avecfx".demarre (…/main.go:40):
missing type: *slog.Logger
```

D'où un test à écrire dans tout projet Fx : `fx.ValidateApp` construit le
graphe sans rien démarrer, et échoue de la même façon. S'y ajoutent une
lecture plus difficile (qui fournit ce `*http.Server` ? il faut chercher), et
un démarrage un peu plus lent, négligeable pour un service. Fx est maintenu activement, et Uber le présente comme le socle
de presque tous ses services Go.

#### Choisir

| | À la main | Wire | Fx |
|-|-----------|------|----|
| Résolution | votre code | code généré, à la compilation | réflexion, à l'exécution |
| Graphe incomplet | erreur de compilation | erreur de génération | erreur au démarrage (testable par `fx.ValidateApp`) |
| Cycle de vie | à écrire | à écrire | intégré (`OnStart`, `OnStop`, signaux) |
| Lisibilité | maximale | bonne (le code généré se lit) | indirecte |
| Dépendance | aucune | outil de génération | bibliothèque, à l'exécution |
| Maintenance | — | archivé depuis août 2025 | active |

- **Par défaut, câblez à la main.** C'est le choix de la plupart des projets
  Go, y compris les plus grands.
- **Fx** se justifie quand une organisation fait vivre **beaucoup de
  services** sur un socle commun (journal, métriques, traces, cycle de vie) :
  chaque équipe ajoute ses modules au même squelette. C'est le contexte pour
  lequel Uber l'a conçu.
- **Wire** : gardez-le dans un projet qui l'utilise déjà (le code généré ne
  vieillit pas), mais ne l'introduisez plus.

Quel que soit le choix, les règles précédentes ne changent pas : des
constructeurs qui reçoivent leurs dépendances, de petites interfaces
déclarées par leur utilisateur, et pas de variable globale. Wire et Fx ne
font qu'**appeler ces constructeurs** à votre place ; un code bien découpé
passe de l'un à l'autre sans réécriture, comme le montrent les trois
programmes de l'exemple, qui partagent tout sauf leur `main`.

## Motif : posséder chaque goroutine

Une goroutine lancée sans savoir comment elle se terminera finit souvent
bloquée pour toujours, avec sa pile et tout ce qu'elle référence : c'est une
**fuite de goroutine**. L'exemple classique interroge plusieurs sources et
garde la première réponse :

```go
func PremierFuyant(ctx context.Context, sources ...Source) string {
	reponses := make(chan string) // sans tampon
	for _, s := range sources {
		go func() { reponses <- s(ctx) }()
	}
	return <-reponses
}
```

Les sources plus lentes envoient leur réponse dans un canal que plus personne
ne lit, et restent bloquées. La correction tient en deux points : un tampon
d'une place par source, pour que chaque envoi réussisse, et un contexte
annulé, pour prévenir les sources que leur réponse n'intéresse plus personne.

```go
func Premier(ctx context.Context, sources ...Source) string {
	ctx, annule := context.WithCancel(ctx)
	defer annule()
	reponses := make(chan string, len(sources))
	…
}
```

Go 1.27 détecte ces fuites. Le profil **`goroutineleak`** lance un cycle de
ramasse-miettes spécial, qui repère les goroutines bloquées sur un canal
qu'aucune autre goroutine ne peut plus atteindre :

```go
pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
```

```console
goroutineleak profile: total 20
20 @ 0x48820a 0x416edc 0x416ad7 0x54e865 0x48e6a1
#	0x54e864	…/fuites.PremierFuyant.func1+0x44	…/fuites/fuites.go:19
```

Dix appels avec deux sources lentes : vingt goroutines perdues, et la ligne
exacte de l'envoi bloqué. La version corrigée n'en laisse aucune.

> [!NOTE]
> Une goroutine qui fuit **ne disparaît jamais**. Le premier essai du test
> l'a montré à ses dépens : lancé trois fois de suite (`-count=3`), il
> trouvait 20, puis 40, puis 60 fuites. Le test mesure désormais l'écart
> avant et après. Le profil est aussi exposé par `net/http/pprof`, ce qui
> permet de le consulter sur un service en production.

## Les anti-motifs

| Anti-motif | Pourquoi c'est un problème | À la place |
|------------|----------------------------|------------|
| Interface déclarée à côté de son unique implémentation | abstraction prématurée, dépendance inutile | déclarer l'interface chez l'utilisateur, quand il en a besoin |
| Grande interface (« `Repository` » à vingt méthodes) | difficile à implémenter, à simuler, à faire évoluer | des interfaces d'une à trois méthodes |
| Pointeur vers une interface (`*io.Reader`) | une interface contient déjà un pointeur | l'interface elle-même |
| Renvoyer un `*MonErreur` nil dans un `error` | l'interface n'est pas nil : `err != nil` est vrai | renvoyer `nil` explicitement (chapitre « Interfaces ») |
| `panic` pour une erreur attendue | plante le programme, contourne la gestion d'erreurs | renvoyer une `error` ; `panic` pour les bugs |
| Erreur ignorée sans explication | une panne passe inaperçue | traiter, ou `_ =` avec un commentaire qui dit pourquoi |
| `fmt.Errorf("…: %v", err)` | l'erreur d'origine devient du texte | `%w`, pour garder `errors.Is` et `errors.AsType` |
| Contexte rangé dans une structure | sa durée de vie devient floue | le passer en premier argument de chaque fonction |
| Variable globale, `init()` qui ouvre une base | état caché, tests impossibles à isoler | dépendances explicites, passées au constructeur |
| Goroutine sans propriétaire | fuite, travail jamais arrêté | un contexte, un `WaitGroup`, un `errgroup` |
| Verrou copié (structure passée par valeur) | il ne protège plus rien | passer un pointeur ; `go vet` le signale |
| Accesseurs `GetNom()`, noms bégayants (`client.ClientConfig`) | bruit | `Nom()`, `client.Config` |
| Packages `utils`, `common`, `helpers` | ne disent rien de leur contenu | nommer le package d'après ce qu'il fournit |

## Les outils qui veillent

### gofmt et go vet

`gofmt` met fin aux débats de mise en forme : il n'y a qu'un style. `go vet`
signale ce que le compilateur accepte mais qui est presque toujours une
erreur. Sur le package `testdata/pieges` de l'exemple :

```console
$ go vet ./testdata/pieges
pieges.go:40:2: result of fmt.Sprintf call not used
pieges.go:19:15: Valeur passes lock by value: …pieges.Compteur contains sync.Mutex
pieges.go:33:7: the cancel function returned by context.WithTimeout should be called,
                not discarded, to avoid a context leak
pieges.go:27:30: fmt.Sprintf format %d has arg nom of wrong type string
```

### go fix : moderniser

Go évolue, et le code écrit il y a quelques années n'utilise pas les
nouveautés. `go fix` a été repensé autour de **modernisateurs** : vingt-six
analyses qui proposent la forme actuelle d'un idiome, avec une
correction sûre. Sur un fichier écrit à l'ancienne :

```console
$ go fix -diff ./testdata/ancien
-func Premier(valeurs []interface{}) interface{} {
+func Premier(valeurs []any) any {

-	for i := 0; i < n; i++ {
+	for i := range n {

-	m := a
-	if b > m {
-		m = b
-	}
+	m := max(b, a)

-	for _, m := range liste {
-		if m == mot {
-			return true
-		}
-	}
-	return false
+	return slices.Contains(liste, mot)

-	sort.Slice(t, func(i, j int) bool { return t[i] < t[j] })
+	slices.Sort(t)

-	if i := strings.Index(s, "="); i >= 0 {
-		return s[:i]
+	if before, _, ok := strings.Cut(s, "="); ok {
+		return before

-		wg.Add(1)
-		go func() {
-			defer wg.Done()
+		wg.Go(func() {

-	var e *fs.PathError
-	if errors.As(err, &e) {
+	if e, ok := errors.AsType[*fs.PathError](err); ok {
```

Sans `-diff`, `go fix` applique les corrections. `go tool fix help` liste
les analyses ; chacune peut être désactivée.

> [!IMPORTANT]
> Une suggestion n'est pas un ordre. Lancé sur le dépôt de ce livre, `go fix`
> a proposé des modernisations dans neuf fichiers. Trois ont été appliquées.
> Six ont été écartées, pour de bonnes raisons :
>
> - la concaténation naïve du chapitre « Profilage et performance » est
>   **lente exprès** : c'est elle que le chapitre mesure ;
> - la boucle `for` en trois clauses, `errors.As` et le littéral
>   `Chien{Animal: Animal{…}}` sont **enseignés** par les chapitres du
>   langage, à côté de leur forme moderne ;
> - dans l'opérateur Kubernetes, le champ `ObjectMeta` explicite est la
>   **convention de l'écosystème**, que lisent les habitués de Kubernetes.

Pour aller plus loin, [staticcheck](https://staticcheck.dev) et
[golangci-lint](https://golangci-lint.run), présentés au chapitre « CI/CD »,
ajoutent des dizaines d'analyses, dont `errcheck` (les erreurs ignorées).

## La grille de relecture

Une relecture de code Go peut suivre ces questions, dans l'ordre :

| Thème | Questions |
|-------|-----------|
| **Lisibilité** | Le code est-il formaté (`gofmt`) ? Les noms sont-ils courts quand la portée l'est, explicites sinon ? Un lecteur comprend-il sans le commentaire ? |
| **Erreurs** | Chaque erreur est-elle traitée ou renvoyée ? Enveloppée avec `%w` et du contexte ? Les erreurs attendues sont-elles des valeurs (`ErrX`) ou des types repérables ? |
| **Interfaces** | Sont-elles petites, déclarées par leur utilisateur ? Le code accepte-t-il des interfaces et renvoie-t-il des types concrets ? |
| **Valeur zéro** | La valeur zéro du type est-elle utilisable, ou documentée comme ne l'étant pas ? |
| **Concurrence** | Chaque goroutine a-t-elle un propriétaire et une fin ? Le contexte est-il propagé ? Les accès partagés sont-ils protégés (`go test -race`) ? |
| **Ressources** | Chaque `Close`, `Unlock`, `cancel` est-il garanti par un `defer` ? Les corps de réponse HTTP sont-ils fermés ? |
| **API** | Le package expose-t-il le minimum ? Les noms évitent-ils de bégayer ? La documentation commence-t-elle par le nom de l'élément ? |
| **Tests** | Les cas d'erreur sont-ils testés ? Les tests sont-ils déterministes (horloge injectée, `synctest`) ? |
| **Outils** | `go vet`, `go fix -diff` et le linter sont-ils silencieux, ou leurs remarques justifiées ? |
| **Simplicité** | Une abstraction, une goroutine, une dépendance pourraient-elles être retirées ? |

## En résumé

- **Options fonctionnelles** pour les réglages facultatifs d'une
  bibliothèque ; une structure de configuration quand c'est plus simple.
- **Middlewares** : une fonction qui enveloppe une interface d'une méthode ;
  l'ordre des couches compte.
- **Injection de dépendances** sans framework : de petites interfaces
  déclarées par leur utilisateur, des implémentations passées au
  constructeur, assemblées par `main`.
  C'est aussi le choix de l'industrie ; Fx (Uber) apporte un cycle de
  vie pour de nombreux services, au prix d'erreurs découvertes au démarrage ;
  Wire (Google) génère le même code, mais il est archivé depuis 2025.
- **Chaque goroutine a un propriétaire** ; le profil `goroutineleak`
  (Go 1.27) détecte celles qui fuient.
- **gofmt, go vet, go fix** : la mise en forme, les erreurs probables et la
  modernisation sont automatiques. Leurs suggestions s'examinent, elles ne
  s'appliquent pas aveuglément.
