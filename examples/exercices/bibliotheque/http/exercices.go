// Package http contient les solutions des exercices du chapitre « HTTP :
// net/http ».
package http

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
)

// Exercice 1 ★ : une route avec un paramètre.

// Bonjour répond à GET /bonjour/{nom}.
func Bonjour(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Bonjour, %s !\n", r.PathValue("nom"))
}

// Exercice 2 ★★ : une petite API JSON.

type addition struct {
	A, B int
}

type somme struct {
	Somme int `json:"somme"`
}

// Additionne répond à POST /additions : {"a": 1, "b": 2} → {"somme": 3}.
func Additionne(w http.ResponseWriter, r *http.Request) {
	var a addition
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&a); err != nil {
		http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(somme{a.A + a.B}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Routes assemble les deux handlers. Les motifs précisent la méthode : un
// GET sur /additions reçoit 405, avec l'en-tête Allow.
func Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /bonjour/{nom}", Bonjour)
	mux.HandleFunc("POST /additions", Additionne)
	return mux
}

// Exercice 3 ★★★ : un middleware d'authentification.

// ExigeCle refuse (401) les requêtes dont l'en-tête X-Cle ne vaut pas cle.
// La comparaison prend le même temps quelle que soit la clé reçue : elle ne
// renseigne pas un attaquant qui mesure les temps de réponse.
func ExigeCle(cle string, suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recue := r.Header.Get("X-Cle")
		if subtle.ConstantTimeCompare([]byte(recue), []byte(cle)) != 1 {
			http.Error(w, "clé absente ou invalide", http.StatusUnauthorized)
			return
		}
		suivant.ServeHTTP(w, r)
	})
}
