// Package templates illustre le chapitre « Templates ».
package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"time"
)

//go:embed gabarits/*.html
var gabarits embed.FS

// Article est une donnée affichée par les gabarits.
type Article struct {
	Titre  string
	Slug   string
	Auteur string
	Tags   []string
	Resume template.HTML // HTML de confiance, non échappé
}

// fonctions est la bibliothèque de fonctions disponible dans les gabarits.
var fonctions = template.FuncMap{
	"date": func(t time.Time) string { return t.Format("02/01/2006") },
	"add":  func(a, b int) int { return a + b },
}

// Site regroupe les pages, chacune composée du gabarit de base et de son
// propre contenu.
type Site struct {
	pages map[string]*template.Template
}

// NouveauSite analyse les gabarits une fois pour toutes, au démarrage.
func NouveauSite() (*Site, error) {
	s := &Site{pages: make(map[string]*template.Template)}
	for _, page := range []string{"accueil", "article"} {
		t, err := template.New(page).Funcs(fonctions).ParseFS(gabarits,
			"gabarits/base.html", "gabarits/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("gabarit %s : %w", page, err)
		}
		s.pages[page] = t
	}
	return s, nil
}

// Affiche exécute la page demandée.
func (s *Site) Affiche(w io.Writer, page string, donnees any) error {
	t, ok := s.pages[page]
	if !ok {
		return fmt.Errorf("page inconnue : %q", page)
	}
	return t.ExecuteTemplate(w, "base", donnees)
}
