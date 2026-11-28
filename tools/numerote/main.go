// Command numerote numérote les parties et les chapitres du site au format du
// livre « Basic Knowledge », d'après l'ordre de leur champ « weight » :
//
//   - une partie : title « II. Le langage », titre « # Partie II — Le langage » ;
//   - un chapitre : title « II.1 Variables, constantes et types » (le nom
//     court, affiché dans le menu), titre « # Chapitre II.1 — … » (le titre
//     complet) ;
//   - la partie « annexes » n'a pas de numéro ; ses chapitres sont lettrés :
//     « A. Aide-mémoire », « # Annexe A — Aide-mémoire » ;
//   - la page des parties (docs/_index.md) : « Chapitres ».
//
// Il tient aussi à jour les sommaires placés entre les marqueurs
// <!-- sommaire --> et <!-- /sommaire --> : page d'accueil, page « Chapitres »
// et introduction de chaque partie.
//
//	go run ./tools/numerote            # met à jour les fichiers
//	go run ./tools/numerote -verifie   # échoue si un fichier doit changer
//
// Ajouter, retirer ou déplacer un chapitre se résume donc à régler son poids,
// puis à relancer l'outil.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/laurentpoirierfr/golang-demystified/tools/internal/content"
)

func main() {
	docs := flag.String("content", "site/content/docs", "répertoire des parties du livre")
	verifie := flag.Bool("verifie", false, "ne rien écrire ; échouer si un fichier doit changer")
	flag.Parse()

	racine, err := content.LoadSection(*docs)
	if err != nil {
		log.Fatal(err)
	}
	changements, err := numerote(racine, filepath.Join(filepath.Dir(*docs), "_index.md"), !*verifie)
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

// chapitre est un chapitre numéroté, pour les sommaires.
type chapitre struct {
	numero string // « II.1 », ou « A » pour une annexe
	court  string
	lien   string // relatif à la partie : « bases/ »
}

// partie est une partie numérotée, pour les sommaires.
type partie struct {
	titre     string // « Partie II — Le langage », ou « Annexes »
	lien      string // relatif à docs : « langage/ »
	chapitres []chapitre
	page      string // fichier _index.md
}

// numerote calcule titres et sommaires, et les écrit si ecrit est vrai. Il
// renvoie la liste des fichiers modifiés (ou à modifier). accueil est la page
// d'accueil du site (son sommaire est mis à jour si elle existe).
func numerote(racine *content.Node, accueil string, ecrit bool) ([]string, error) {
	var changements []string
	applique := func(chemin string, f func(texte string) string) error {
		brut, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		nouveau := f(string(brut))
		if nouveau == string(brut) {
			return nil
		}
		if !slices.Contains(changements, chemin) {
			changements = append(changements, chemin)
		}
		if !ecrit {
			return nil
		}
		return os.WriteFile(chemin, []byte(nouveau), 0o644)
	}

	var parties []partie
	numero := 0
	for _, p := range racine.Children {
		if !p.Section {
			continue
		}
		nomPartie := Court(p.Path, titreBrut(p.Path), partieRe)
		annexes := filepath.Base(filepath.Dir(p.Path)) == "annexes"
		pt := partie{lien: filepath.Base(filepath.Dir(p.Path)) + "/", page: p.Path}
		var titre string
		if annexes {
			titre, pt.titre = nomPartie, nomPartie
		} else {
			numero++
			titre = content.Romain(numero) + ". " + nomPartie
			pt.titre = "Partie " + content.Romain(numero) + " — " + nomPartie
		}
		aEnfants := false
		for _, c := range p.Children {
			if !c.Section {
				aEnfants = true
			}
		}
		if err := applique(p.Path, func(t string) string {
			t = fixeCle(t, "title", strconv.Quote(titre))
			t = retireCle(t, "linkTitle")
			if aEnfants {
				t = fixeCle(t, "bookCollapseSection", "true")
			}
			return fixeTitre(t, pt.titre)
		}); err != nil {
			return changements, err
		}

		n := 0
		for _, c := range p.Children {
			if c.Section {
				continue
			}
			n++
			var num, ftitre, h1 string
			brut := titreBrut(c.Path)
			court := Court(c.Path, brut, chapitreRe)
			complet := Complet(c.Path, brut)
			if annexes {
				num = string(rune('A' + n - 1))
				ftitre = num + ". " + court
				h1 = "Annexe " + num + " — " + complet
			} else {
				num = fmt.Sprintf("%s.%d", content.Romain(numero), n)
				ftitre = num + " " + court
				h1 = "Chapitre " + num + " — " + complet
			}
			if err := applique(c.Path, func(t string) string {
				t = fixeCle(t, "title", strconv.Quote(ftitre))
				t = retireCle(t, "linkTitle")
				return fixeTitre(t, h1)
			}); err != nil {
				return changements, err
			}
			nom := strings.TrimSuffix(filepath.Base(c.Path), ".md")
			if nom == "index" {
				nom = filepath.Base(filepath.Dir(c.Path))
			}
			pt.chapitres = append(pt.chapitres, chapitre{numero: num, court: court, lien: nom + "/"})
		}
		parties = append(parties, pt)
	}

	// Sommaire de chaque partie
	for _, pt := range parties {
		if err := applique(pt.page, func(t string) string {
			return fixeSommaire(t, sommairePartie(pt))
		}); err != nil {
			return changements, err
		}
	}
	// Page « Chapitres »
	if err := applique(racine.Path, func(t string) string {
		t = fixeCle(t, "title", strconv.Quote("Chapitres"))
		t = retireCle(t, "linkTitle")
		t = fixeCle(t, "bookFlatSection", "true")
		t = fixeTitre(t, "Chapitres")
		return fixeSommaire(t, sommaireChapitres(parties, ""))
	}); err != nil {
		return changements, err
	}
	// Page d'accueil
	if _, err := os.Stat(accueil); err == nil {
		if err := applique(accueil, func(t string) string {
			return fixeSommaire(t, sommaireAccueil(parties))
		}); err != nil {
			return changements, err
		}
	}
	return changements, nil
}

func listeChapitres(b *strings.Builder, chapitres []chapitre, prefixe string) {
	for _, c := range chapitres {
		fmt.Fprintf(b, "- **%s** [%s](%s%s)\n", c.numero, c.court, prefixe, c.lien)
	}
}

func sommairePartie(pt partie) string {
	var b strings.Builder
	listeChapitres(&b, pt.chapitres, "")
	return b.String()
}

func sommaireChapitres(parties []partie, prefixe string) string {
	var b strings.Builder
	for i, pt := range parties {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "**[%s](%s%s)**\n", pt.titre, prefixe, pt.lien)
		if len(pt.chapitres) > 0 {
			b.WriteString("\n")
			listeChapitres(&b, pt.chapitres, prefixe+pt.lien)
		}
	}
	return b.String()
}

func sommaireAccueil(parties []partie) string {
	var b strings.Builder
	for i, pt := range parties {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n", pt.titre)
		if len(pt.chapitres) > 0 {
			b.WriteString("\n")
			listeChapitres(&b, pt.chapitres, "docs/"+pt.lien)
		}
	}
	return b.String()
}

var (
	// Préfixes de numéro dans le champ title.
	partieRe   = regexp.MustCompile(`^[IVXLC]+\. `)
	chapitreRe = regexp.MustCompile(`^([IVXLC]+\.\d+|[A-Z]\.) `)
	// Ancien format : « 3. Nom court » dans linkTitle.
	ancienRe    = regexp.MustCompile(`^\d+\. `)
	linkTitleRe = regexp.MustCompile(`(?m)^linkTitle: .*$`)
	// Préfixe du titre de niveau 1.
	h1Re = regexp.MustCompile(`^(Partie [IVXLC]+|Chapitre [IVXLC]+\.\d+|Annexe [A-Z]) — `)
)

// Court renvoie le nom court d'une page, celui du menu : son ancien
// linkTitle s'il en a un, sinon son titre sans numéro ; un titre jamais
// numéroté est coupé avant le premier « : » (« JSON : encoding/json » →
// « JSON »).
func Court(chemin, titre string, numeroRe *regexp.Regexp) string {
	if brut, err := os.ReadFile(chemin); err == nil {
		if l := linkTitleRe.FindString(frontMatter(string(brut))); l != "" {
			v := strings.TrimSpace(strings.TrimPrefix(l, "linkTitle:"))
			if s, err := strconv.Unquote(v); err == nil {
				v = s
			}
			v = ancienRe.ReplaceAllString(v, "")
			v = regexp.MustCompile(`^[IVXLC]+ - `).ReplaceAllString(v, "")
			return v
		}
	}
	if numeroRe.MatchString(titre) {
		return numeroRe.ReplaceAllString(titre, "")
	}
	if numeroRe == partieRe {
		return titre
	}
	court, _, _ := strings.Cut(titre, " : ")
	return strings.TrimSpace(court)
}

// Complet renvoie le titre complet d'un chapitre : celui de son titre de
// niveau 1, sans numéro, s'il en a un ; sinon son titre.
func Complet(chemin, titre string) string {
	if brut, err := os.ReadFile(chemin); err == nil {
		if h := premierTitre(corps(string(brut))); h != "" {
			return h1Re.ReplaceAllString(h, "")
		}
	}
	return chapitreRe.ReplaceAllString(titre, "")
}

// titreBrut renvoie le champ title du fichier, tel qu'il est écrit (le
// paquet content, lui, en retire le numéro).
func titreBrut(chemin string) string {
	brut, err := os.ReadFile(chemin)
	if err != nil {
		return ""
	}
	l := regexp.MustCompile(`(?m)^title:.*$`).FindString(frontMatter(string(brut)))
	v := strings.TrimSpace(strings.TrimPrefix(l, "title:"))
	if s, err := strconv.Unquote(v); err == nil {
		return s
	}
	return strings.Trim(v, `'`)
}

// frontMatter renvoie le début du fichier jusqu'à la fin du front matter.
func frontMatter(texte string) string {
	if !strings.HasPrefix(texte, "---\n") {
		return ""
	}
	fin := strings.Index(texte[4:], "\n---\n")
	if fin < 0 {
		return ""
	}
	return texte[:4+fin]
}

// corps renvoie le texte après le front matter.
func corps(texte string) string {
	fm := frontMatter(texte)
	if fm == "" {
		return texte
	}
	return texte[len(fm)+len("\n---\n"):]
}

// premierTitre renvoie le titre « # … » qui ouvre le corps, ou "".
func premierTitre(c string) string {
	l := strings.TrimLeft(c, "\n")
	ligne, _, _ := strings.Cut(l, "\n")
	if strings.HasPrefix(ligne, "# ") {
		return strings.TrimSpace(ligne[2:])
	}
	return ""
}

// fixeCle écrit « cle: valeur » dans le front matter : elle remplace la
// ligne existante (et son commentaire), ou l'ajoute après le titre.
func fixeCle(texte, cle, valeur string) string {
	fm := frontMatter(texte)
	if fm == "" {
		return texte
	}
	ligne := cle + ": " + valeur
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(cle) + `:.*$`)
	if loc := re.FindStringIndex(fm); loc != nil {
		existante := fm[loc[0]:loc[1]]
		if v, _, _ := strings.Cut(strings.TrimSpace(strings.TrimPrefix(existante, cle+":")), " #"); strings.TrimSpace(v) == valeur {
			return texte // même valeur : on garde un éventuel commentaire
		}
		return fm[:loc[0]] + ligne + texte[loc[1]:]
	}
	if cle == "title" {
		return "---\n" + ligne + "\n" + texte[4:]
	}
	titre := regexp.MustCompile(`(?m)^title:.*$`).FindStringIndex(fm)
	if titre == nil {
		return texte
	}
	return texte[:titre[1]] + "\n" + ligne + texte[titre[1]:]
}

// retireCle retire la ligne « cle: … » du front matter.
func retireCle(texte, cle string) string {
	fm := frontMatter(texte)
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(cle) + `:.*\n`)
	if loc := re.FindStringIndex(fm + "\n"); loc != nil {
		return texte[:loc[0]] + texte[loc[1]:]
	}
	return texte
}

// fixeTitre écrit le titre « # titre » en tête du corps : il remplace celui
// qui s'y trouve, ou l'ajoute.
func fixeTitre(texte, titre string) string {
	fm := frontMatter(texte)
	if fm == "" {
		return texte
	}
	debut := len(fm) + len("\n---\n")
	c := texte[debut:]
	reste := strings.TrimLeft(c, "\n")
	if premierTitre(reste) != "" {
		_, apres, _ := strings.Cut(reste, "\n")
		reste = strings.TrimLeft(apres, "\n")
	}
	nouveau := "\n# " + titre + "\n"
	if reste != "" {
		nouveau += "\n" + reste
	}
	return texte[:debut] + nouveau
}

const (
	debutSommaire = "<!-- sommaire -->"
	finSommaire   = "<!-- /sommaire -->"
)

// fixeSommaire remplace le texte entre les marqueurs de sommaire, s'ils sont
// présents.
func fixeSommaire(texte, sommaire string) string {
	d := strings.Index(texte, debutSommaire)
	f := strings.Index(texte, finSommaire)
	if d < 0 || f < d {
		return texte
	}
	return texte[:d+len(debutSommaire)] + "\n\n" + sommaire + "\n" + texte[f:]
}
