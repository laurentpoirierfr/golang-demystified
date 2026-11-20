// Command site sert la page de démonstration (le répertoire web) : le module
// WebAssembly doit être servi en HTTP, avec le type application/wasm, pour
// que le navigateur le compile pendant son téléchargement.
//
//	go run ./cmd/site    # puis http://localhost:8080
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	adresse := flag.String("adresse", "localhost:8080", "adresse d'écoute")
	dossier := flag.String("web", "web", "répertoire à servir")
	flag.Parse()

	srv := &http.Server{
		Addr:              *adresse,
		Handler:           http.FileServerFS(os.DirFS(*dossier)), // .wasm → application/wasm
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("http://%s", *adresse)
	log.Fatal(srv.ListenAndServe())
}
