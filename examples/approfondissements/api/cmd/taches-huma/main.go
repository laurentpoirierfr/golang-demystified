// Command taches-huma démarre la variante Huma de l'API des tâches.
//
//	go run ./cmd/taches-huma        # puis http://localhost:8081/docs
package main

import (
	"cmp"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/store"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/internal/webhuma"
)

func main() {
	s, err := store.Ouvre(context.Background(), cmp.Or(os.Getenv("TACHES_BASE"), "file:taches.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	srv := &http.Server{
		Addr:              cmp.Or(os.Getenv("TACHES_ADRESSE"), ":8081"),
		Handler:           webhuma.Routes(s),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("API Huma sur %s, documentation sur /docs", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
