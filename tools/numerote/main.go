// Command numerote numérote les parties et les chapitres du site, d'après
// l'ordre de leur champ « weight » : « V - Approfondissements » pour une
// partie, « 3. Une API web complète » pour un chapitre. Il écrit le champ
// linkTitle du front matter, utilisé par le menu du site ; le titre (title),
// lui, ne change pas, et le livre PDF numérote ses chapitres lui-même.
//
//	go run ./tools/numerote            # met à jour les fichiers
//	go run ./tools/numerote -verifie   # échoue si la numérotation n'est pas à jour
//
// Ajouter, retirer ou déplacer un chapitre se résume donc à régler son poids,
// puis à relancer l'outil.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func main() {
	docs := flag.String("content", "site/content/docs", "répertoire racine du contenu")
	verifie := flag.Bool("verifie", false, "ne rien écrire ; échouer si un fichier doit changer")
	flag.Parse()

	racine, err := content.LoadSection(*docs)
	if err != nil {
		log.Fatal(err)
	}
	changements, err := numerote(racine, !*verifie)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range changements {
		fmt.Println(c)
	}
	if *verifie && len(changements) > 0 {
		fmt.Fprintln(os.Stderr, "numérotation à mettre à jour : make numerote")
		os.Exit(1)
	}
}

// numerote calcule les titres de menu, et les écrit si ecrit est vrai. Il
// renvoie la liste des fichiers modifiés (ou à modifier).
func numerote(racine *content.Node, ecrit bool) ([]string, error) {
	var changements []string
	applique := func(chemin, titre string) error {
		modifie, err := fixe(chemin, titre, ecrit)
		if modifie {
			changements = append(changements, fmt.Sprintf("%s : %q", chemin, titre))
		}
		return err
	}
	partie := 0
	for _, p := range racine.Children {
		if !p.Section {
			continue
		}
		partie++
		if err := applique(p.Path, content.Romain(partie)+" - "+p.Title); err != nil {
			return changements, err
		}
		chapitre := 0
		for _, c := range p.Children {
			if c.Section {
				continue
			}
			chapitre++
			if err := applique(c.Path, fmt.Sprintf("%d. %s", chapitre, Court(c.Path, c.Title))); err != nil {
				return changements, err
			}
		}
	}
	return changements, nil
}

var (
	numeroRe    = regexp.MustCompile(`^\d+\. `)
	linkTitleRe = regexp.MustCompile(`(?m)^linkTitle: .*$`)
	titleRe     = regexp.MustCompile(`(?m)^title: .*$`)
)

// Court renvoie le nom court d'un chapitre : son linkTitle actuel, sans son
// numéro, s'il en a un (on peut donc choisir un nom court à la main) ; sinon
// son titre, coupé avant le premier « : » (« JSON : encoding/json » → « JSON »).
func Court(chemin, titre string) string {
	if brut, err := os.ReadFile(chemin); err == nil {
		if l := linkTitleRe.Find(frontMatter(string(brut))); l != nil {
			v := strings.TrimSpace(strings.TrimPrefix(string(l), "linkTitle:"))
			if s, err := strconv.Unquote(v); err == nil {
				v = s
			}
			return numeroRe.ReplaceAllString(v, "")
		}
	}
	court, _, _ := strings.Cut(titre, " : ")
	return strings.TrimSpace(court)
}

// fixe écrit linkTitle dans le front matter du fichier : il remplace la ligne
// existante, ou l'ajoute après le titre. Il renvoie vrai si le fichier change.
func fixe(chemin, titre string, ecrit bool) (bool, error) {
	brut, err := os.ReadFile(chemin)
	if err != nil {
		return false, err
	}
	texte := string(brut)
	fm := frontMatter(texte)
	ligne := "linkTitle: " + strconv.Quote(titre)
	var nouveau string
	switch {
	case linkTitleRe.Match(fm):
		nouveau = string(linkTitleRe.ReplaceAllLiteral(fm, []byte(ligne))) + texte[len(fm):]
	case titleRe.Match(fm):
		loc := titleRe.FindIndex(fm)
		nouveau = texte[:loc[1]] + "\n" + ligne + texte[loc[1]:]
	default:
		return false, fmt.Errorf("%s : titre manquant", chemin)
	}
	if nouveau == texte {
		return false, nil
	}
	if !ecrit {
		return true, nil
	}
	return true, os.WriteFile(chemin, []byte(nouveau), 0o644)
}

// frontMatter renvoie le début du fichier jusqu'à la fin du front matter.
func frontMatter(texte string) []byte {
	if !strings.HasPrefix(texte, "---\n") {
		return nil
	}
	fin := strings.Index(texte[4:], "\n---\n")
	if fin < 0 {
		return nil
	}
	return []byte(texte[:4+fin])
}
