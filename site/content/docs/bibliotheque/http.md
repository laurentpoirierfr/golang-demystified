---
title: "HTTP : net/http"
linkTitle: "6. HTTP"
weight: 60
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/http`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/http).

Le package `net/http` fournit un client et un serveur HTTP complets, prêts pour
la production : HTTP/1.1, HTTP/2, TLS, keep-alive, compression… De nombreux
services Go n'utilisent aucun framework web.

## Un serveur minimal

```go
func main() {
	http.HandleFunc("GET /bonjour", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bonjour, %s !\n", r.URL.Query().Get("nom"))
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Un **handler** reçoit deux paramètres :

- `http.ResponseWriter` : pour écrire la réponse (en-têtes, code de statut,
  corps). C'est un `io.Writer` ;
- `*http.Request` : la requête (méthode, URL, en-têtes, corps, contexte).

Chaque requête est traitée dans sa **propre goroutine** : un handler doit
donc protéger les données partagées.

## Le routage

> [!NOTE]
> **Go 1.22** — `http.ServeMux` accepte des **méthodes** et des
> **paramètres de chemin** dans ses motifs. Pour la plupart des API, un
> routeur externe n'est plus nécessaire.

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /taches", a.liste)
mux.HandleFunc("POST /taches", a.cree)
mux.HandleFunc("GET /taches/{id}", a.detail)
mux.HandleFunc("DELETE /taches/{id}", a.supprime)
mux.HandleFunc("GET /{$}", accueil)
```

| Motif                      | Correspondance                                        |
|----------------------------|-------------------------------------------------------|
| `/taches`                  | Le chemin exact `/taches`, toutes méthodes            |
| `GET /taches`              | `GET` (et `HEAD`) sur `/taches`                       |
| `/static/`                 | Tout chemin qui commence par `/static/`               |
| `GET /taches/{id}`         | Un segment variable, lu avec `r.PathValue("id")`      |
| `/fichiers/{chemin...}`    | Le reste du chemin, sur plusieurs segments            |
| `/{$}`                     | Uniquement `/`, et non tout le site                   |
| `api.example.com/`         | Les requêtes destinées à cet hôte                     |

Lorsque plusieurs motifs correspondent, le **plus spécifique** l'emporte. Le
serveur répond automatiquement `405 Method Not Allowed`, avec l'en-tête
`Allow`, si le chemin existe pour une autre méthode.

## Une API JSON

Les handlers d'une API sont souvent des méthodes d'une structure qui porte
leurs dépendances (stockage, configuration, journal) :

```go
type API struct{ store *Store }

func (a *API) cree(w http.ResponseWriter, r *http.Request) {
	var t Tache
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 Mo maximum
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || t.Titre == "" {
		http.Error(w, "tâche invalide", http.StatusBadRequest)
		return
	}
	a.store.Ajoute(&t)
	w.Header().Set("Location", "/taches/"+strconv.Itoa(t.ID))
	ecritJSON(w, http.StatusCreated, t)
}

func (a *API) detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identifiant invalide", http.StatusBadRequest)
		return
	}
	t, ok := a.store.Trouve(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	ecritJSON(w, http.StatusOK, t)
}

func ecritJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	json.NewEncoder(w).Encode(v)
}
```

> [!WARNING]
> L'ordre compte : les en-têtes doivent être définis **avant**
> `WriteHeader`, et `WriteHeader` doit être appelé **avant** d'écrire le
> corps. Le premier `Write` envoie implicitement un statut `200`.

## Les middlewares

Un middleware est une fonction qui enveloppe un `http.Handler` pour ajouter un
comportement commun : journalisation, authentification, récupération des
paniques, compression…

```go
func journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		suivant.ServeHTTP(w, r)
		slog.Info("requête", "méthode", r.Method, "chemin", r.URL.Path,
			"durée", time.Since(debut))
	})
}

handler := journalise(mux)
```

`http.HandlerFunc` convertit une fonction en `http.Handler`. Les middlewares se
composent en les imbriquant : `journalise(authentifie(mux))`.

> [!NOTE]
> **Go 1.25** — `http.CrossOriginProtection` protège contre les attaques CSRF
> en s'appuyant sur les en-têtes `Sec-Fetch-Site` des navigateurs modernes,
> sans jeton.

## Un serveur prêt pour la production

`http.ListenAndServe` n'impose aucun délai : un client lent peut monopoliser
une connexion indéfiniment. En production, configurez un `http.Server` et
gérez l'arrêt propre :

```go
srv := &http.Server{
	Addr:              ":8080",
	Handler:           handler,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       10 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       2 * time.Minute,
}

go func() {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}()

// Attend SIGINT ou SIGTERM, puis laisse 10 s aux requêtes en cours.
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
<-ctx.Done()

arret, annule := context.WithTimeout(context.Background(), 10*time.Second)
defer annule()
srv.Shutdown(arret)
```

Pour servir des fichiers statiques, `http.FileServerFS` (**Go 1.22**) accepte
un `fs.FS`, par exemple un `embed.FS` intégré à l'exécutable :

```go
mux.Handle("GET /static/", http.FileServerFS(statiques))
```

## Le client

```go
client := &http.Client{Timeout: 5 * time.Second}

resp, err := client.Get(url)
if err != nil {
	return err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
	return fmt.Errorf("statut inattendu : %s", resp.Status)
}
var taches []Tache
if err := json.NewDecoder(resp.Body).Decode(&taches); err != nil {
	return err
}
```

Trois règles à retenir :

- **toujours** fermer `resp.Body`, sinon la connexion n'est pas réutilisée ;
- **ne jamais** utiliser `http.DefaultClient` ni `http.Get` en production : ils
  n'ont pas de délai d'expiration ;
- une erreur `nil` ne signifie pas un succès : un statut 404 ou 500 n'est
  **pas** une erreur pour le client. Testez `resp.StatusCode`.

Pour une requête plus complexe, construisez-la avec `http.NewRequestWithContext`,
qui permet aussi de l'annuler :

```go
req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(corps))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+jeton)
resp, err := client.Do(req)
```

## Tester avec httptest

`net/http/httptest` permet de tester un handler **sans réseau** :

```go
h := api.Routes()

req := httptest.NewRequest("POST", "/taches", strings.NewReader(`{"titre": "Écrire le livre"}`))
rec := httptest.NewRecorder()
h.ServeHTTP(rec, req)
fmt.Println(rec.Code, rec.Header().Get("Location"))
// 201 /taches/1

rec = httptest.NewRecorder()
h.ServeHTTP(rec, httptest.NewRequest("PUT", "/taches/1", nil))
fmt.Println(rec.Code, rec.Header().Get("Allow"))
// 405 DELETE, GET, HEAD
```

`httptest.NewServer` démarre un vrai serveur sur un port local, pour tester un
client :

```go
srv := httptest.NewServer(api.Routes())
defer srv.Close()
resp, err := client.Get(srv.URL + "/taches")
```

> [!NOTE]
> **Go 1.27** — `httptest.NewTestServer` utilise un réseau simulé en mémoire,
> compatible avec `testing/synctest` pour tester les délais et les timeouts
> sans attendre réellement.

## Exercices

Les solutions sont dans
[`examples/exercices/bibliotheque/http`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/bibliotheque/http),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

Les tests utilisent `httptest` : aucun serveur n'est démarré.

### Une route avec un paramètre ★

```go
func Bonjour(w http.ResponseWriter, r *http.Request)
```

Enregistrée sur `GET /bonjour/{nom}`, elle répond `Bonjour, Ada !`.

### Une petite API JSON ★★

```go
func Additionne(w http.ResponseWriter, r *http.Request)
func Routes() http.Handler
```

`POST /additions` avec `{"a": 40, "b": 2}` répond `{"somme":42}`, de type
`application/json`. Un JSON invalide reçoit 400, un corps de plus de 1 Kio
aussi (`http.MaxBytesReader`). `Routes` assemble les deux handlers : un `GET
/additions` doit recevoir 405.

### Un middleware d'authentification ★★★

```go
func ExigeCle(cle string, suivant http.Handler) http.Handler
```

Refusez (401) les requêtes dont l'en-tête `X-Cle` est absent ou faux.
Comparez avec `subtle.ConstantTimeCompare` : une comparaison ordinaire
s'arrête au premier caractère différent, et son temps de réponse renseigne un
attaquant.
