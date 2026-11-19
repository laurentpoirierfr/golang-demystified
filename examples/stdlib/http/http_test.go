// Exemples du chapitre « HTTP : net/http ».
package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type Tache struct {
	ID    int    `json:"id"`
	Titre string `json:"titre"`
	Faite bool   `json:"faite"`
}

// Store est un stockage en mémoire, protégé par un mutex.
type Store struct {
	mu      sync.Mutex
	taches  map[int]Tache
	suivant int
}

func NouveauStore() *Store { return &Store{taches: make(map[int]Tache), suivant: 1} }

// API regroupe les handlers et leurs dépendances.
type API struct{ store *Store }

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /taches", a.liste)
	mux.HandleFunc("POST /taches", a.cree)
	mux.HandleFunc("GET /taches/{id}", a.detail)
	mux.HandleFunc("DELETE /taches/{id}", a.supprime)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "API des tâches")
	})
	return journalise(mux)
}

func (a *API) liste(w http.ResponseWriter, r *http.Request) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	taches := make([]Tache, 0, len(a.store.taches))
	for i := 1; i < a.store.suivant; i++ {
		if t, ok := a.store.taches[i]; ok {
			taches = append(taches, t)
		}
	}
	ecritJSON(w, http.StatusOK, taches)
}

func (a *API) cree(w http.ResponseWriter, r *http.Request) {
	var t Tache
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || t.Titre == "" {
		http.Error(w, "tâche invalide", http.StatusBadRequest)
		return
	}
	a.store.mu.Lock()
	t.ID = a.store.suivant
	a.store.suivant++
	a.store.taches[t.ID] = t
	a.store.mu.Unlock()

	w.Header().Set("Location", "/taches/"+strconv.Itoa(t.ID))
	ecritJSON(w, http.StatusCreated, t)
}

func (a *API) detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "identifiant invalide", http.StatusBadRequest)
		return
	}
	a.store.mu.Lock()
	t, ok := a.store.taches[id]
	a.store.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	ecritJSON(w, http.StatusOK, t)
}

func (a *API) supprime(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	a.store.mu.Lock()
	delete(a.store.taches, id)
	a.store.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func ecritJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	json.NewEncoder(w).Encode(v)
}

// journalise est un middleware : il enveloppe un handler.
func journalise(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		debut := time.Now()
		suivant.ServeHTTP(w, r)
		slog.Debug("requête", "méthode", r.Method, "chemin", r.URL.Path, "durée", time.Since(debut))
	})
}

func Example_recorder() {
	api := &API{store: NouveauStore()}
	h := api.Routes()

	req := httptest.NewRequest("POST", "/taches", strings.NewReader(`{"titre": "Écrire le livre"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	fmt.Println(rec.Code, rec.Header().Get("Location"), strings.TrimSpace(rec.Body.String()))

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/taches/1", nil))
	fmt.Println(rec.Code, strings.TrimSpace(rec.Body.String()))

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("PUT", "/taches/1", nil))
	fmt.Println(rec.Code, rec.Header().Get("Allow"))

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/inconnu", nil))
	fmt.Println(rec.Code)
	// Output:
	// 201 /taches/1 {"id":1,"titre":"Écrire le livre","faite":false}
	// 200 {"id":1,"titre":"Écrire le livre","faite":false}
	// 405 DELETE, GET, HEAD
	// 404
}

func Example_client() {
	srv := httptest.NewServer((&API{store: NouveauStore()}).Routes())
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(srv.URL+"/taches", "application/json",
		strings.NewReader(`{"titre": "Relire"}`))
	if err != nil {
		fmt.Println(err)
		return
	}
	resp.Body.Close()

	resp, err = client.Get(srv.URL + "/taches")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	var taches []Tache
	json.NewDecoder(resp.Body).Decode(&taches)
	fmt.Println(resp.StatusCode, resp.Header.Get("Content-Type"), taches)
	// Output: 200 application/json [{1 Relire false}]
}

func TestCreationInvalide(t *testing.T) {
	h := (&API{store: NouveauStore()}).Routes()
	cas := []struct {
		nom   string
		corps string
	}{
		{"JSON invalide", `{"titre":`},
		{"titre vide", `{"titre": ""}`},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("POST", "/taches", strings.NewReader(c.corps)))
			if rec.Code != http.StatusBadRequest {
				corps, _ := io.ReadAll(rec.Body)
				t.Errorf("code = %d, attendu 400 (%s)", rec.Code, corps)
			}
		})
	}
}
