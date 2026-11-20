// Package web expose une petite API de notes, protégée : authentification par
// jeton, autorisations par portée, validation des entrées, accès aux fichiers
// confiné, protection CSRF et en-têtes de sécurité.
package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/auth"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/secret"
)

// TailleMax est la taille maximale du corps d'une requête : au-delà, la
// lecture échoue, et un client ne peut pas épuiser la mémoire du serveur.
const TailleMax = 16 << 10 // 16 Kio

// Note est une note d'un utilisateur.
type Note struct {
	ID      int    `json:"id"`
	Auteur  string `json:"auteur"`
	Titre   string `json:"titre"`
	Contenu string `json:"contenu"`
}

// Serveur porte les dépendances des handlers.
type Serveur struct {
	log      *slog.Logger
	fichiers *os.Root // les fichiers servis ne peuvent pas sortir de ce répertoire
	admin    secret.Secret

	mu    sync.Mutex
	notes []Note
}

// Nouveau crée le serveur. fichiers est le répertoire des pièces jointes ;
// admin, la clé d'administration (vide : administration désactivée).
func Nouveau(log *slog.Logger, fichiers *os.Root, admin secret.Secret) *Serveur {
	return &Serveur{log: log, fichiers: fichiers, admin: admin}
}

// Routes assemble l'API. authentifie est le middleware d'authentification.
func (s *Serveur) Routes(authentifie func(http.Handler) http.Handler) http.Handler {
	prive := http.NewServeMux()
	prive.HandleFunc("GET /moi", s.moi)
	prive.Handle("GET /notes", auth.Exige("notes:lire", http.HandlerFunc(s.listeNotes)))
	prive.Handle("POST /notes", auth.Exige("notes:ecrire", http.HandlerFunc(s.creeNote)))
	prive.Handle("GET /fichiers/{nom...}", auth.Exige("notes:lire", http.HandlerFunc(s.fichier)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /sante", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	mux.HandleFunc("GET /admin/stats", s.stats) // clé d'administration, sans jeton
	mux.Handle("/", authentifie(prive))

	// Protection CSRF (Go 1.25) : refuse les requêtes « non sûres » (POST,
	// PUT, DELETE…) envoyées par un navigateur depuis un autre site.
	csrf := http.NewCrossOriginProtection()
	return entetes(csrf.Handler(mux))
}

// entetes ajoute les en-têtes de sécurité recommandés pour une API.
func entetes(suivant http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// HTTPS uniquement, pendant deux ans, sous-domaines compris.
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		// Le navigateur ne devine pas le type d'un contenu (« MIME sniffing »).
		h.Set("X-Content-Type-Options", "nosniff")
		// Une API ne sert ni script, ni style, ni cadre.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		// Les réponses personnelles ne sont pas mises en cache.
		h.Set("Cache-Control", "no-store")
		suivant.ServeHTTP(w, r)
	})
}

func (s *Serveur) moi(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.IdentiteDe(r.Context())
	reponse := map[string]any{"sujet": id.Sujet, "portees": id.Portees}
	// Avec mTLS, le certificat du client identifie aussi le service appelant.
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		reponse["certificat"] = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	ecritJSON(w, http.StatusOK, reponse)
}

func (s *Serveur) listeNotes(w http.ResponseWriter, r *http.Request) {
	id, _ := auth.IdentiteDe(r.Context())
	s.mu.Lock()
	defer s.mu.Unlock()
	// Chacun ne voit que ses notes : l'autorisation porte aussi sur les
	// données, pas seulement sur la route.
	notes := []Note{}
	for _, n := range s.notes {
		if n.Auteur == id.Sujet {
			notes = append(notes, n)
		}
	}
	ecritJSON(w, http.StatusOK, notes)
}

// nouvelleNote est ce que le client a le droit d'envoyer : ni ID, ni auteur.
// Décoder dans un type dédié, plutôt que dans Note, empêche un client de
// choisir l'auteur de la note (« mass assignment »).
type nouvelleNote struct {
	Titre   string `json:"titre"`
	Contenu string `json:"contenu"`
}

func (s *Serveur) creeNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, TailleMax)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // « auteur » ou « id » sont refusés, pas ignorés
	var n nouvelleNote
	if err := dec.Decode(&n); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "requête trop volumineuse", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
		return
	}
	if dec.More() {
		http.Error(w, "JSON invalide : un seul objet attendu", http.StatusBadRequest)
		return
	}
	if err := n.valide(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	id, _ := auth.IdentiteDe(r.Context())
	s.mu.Lock()
	note := Note{ID: len(s.notes) + 1, Auteur: id.Sujet, Titre: strings.TrimSpace(n.Titre), Contenu: n.Contenu}
	s.notes = append(s.notes, note)
	s.mu.Unlock()
	ecritJSON(w, http.StatusCreated, note)
}

// valide applique une liste blanche : ce qui n'est pas explicitement permis
// est refusé.
func (n nouvelleNote) valide() error {
	titre := strings.TrimSpace(n.Titre)
	switch {
	case titre == "":
		return errors.New("titre : obligatoire")
	case utf8.RuneCountInString(titre) > 100:
		return errors.New("titre : 100 caractères au plus")
	case strings.ContainsFunc(titre, unicode.IsControl):
		return errors.New("titre : caractères de contrôle interdits")
	case utf8.RuneCountInString(n.Contenu) > 2000:
		return errors.New("contenu : 2000 caractères au plus")
	}
	return nil
}

// fichier sert une pièce jointe. Le nom vient du client : avec os.Root
// (Go 1.24), « ../../etc/passwd » ou un lien symbolique vers l'extérieur ne
// peuvent pas sortir du répertoire.
func (s *Serveur) fichier(w http.ResponseWriter, r *http.Request) {
	nom := r.PathValue("nom")
	f, err := s.fichiers.Open(nom)
	if err != nil {
		s.log.WarnContext(r.Context(), "fichier refusé", "nom", nom, "raison", err.Error())
		http.NotFound(w, r) // même réponse que pour un fichier absent
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	// Téléchargé, jamais affiché : un fichier HTML déposé ne peut pas
	// exécuter de script dans le navigateur.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	if _, err := io.Copy(w, f); err != nil {
		s.log.WarnContext(r.Context(), "envoi du fichier", "erreur", err)
	}
}

// stats est protégé par une clé d'administration, comparée en temps constant.
func (s *Serveur) stats(w http.ResponseWriter, r *http.Request) {
	if s.admin.Vide() || !s.admin.Egal(r.Header.Get("X-Cle-Admin")) {
		http.Error(w, "accès refusé", http.StatusForbidden)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ecritJSON(w, http.StatusOK, map[string]int{"notes": len(s.notes)})
}

func ecritJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	// encoding/json échappe <, > et & : une réponse JSON ne peut pas être
	// interprétée comme du HTML. L'en-tête est déjà parti : une erreur
	// d'écriture (client déconnecté) ne peut plus être signalée au client.
	_ = json.NewEncoder(w).Encode(v)
}
