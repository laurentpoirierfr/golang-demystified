// Package glossaire vérifie le glossaire du livre
// (site/content/docs/annexes/glossaire.md) : l'ordre alphabétique, la
// concordance des deux tables, et chaque renvoi vers un chapitre.
package glossaire

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const docs = "../../../site/content/docs"

type entree struct {
	terme, anglais, lien string
}

// cle donne l'ordre du dictionnaire : sans casse ni accents.
func cle(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var lienRe = regexp.MustCompile(`\]\(\.\./\.\./([a-z0-9-]+/[a-z0-9-]+)/\)`)

// lit renvoie les entrées de la table principale, et les paires de la table
// inverse (anglais, terme du livre).
func lit(t *testing.T) (principales []entree, inverses [][2]string) {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join(docs, "annexes", "glossaire.md"))
	if err != nil {
		t.Fatal(err)
	}
	inverse := false
	for ligne := range strings.Lines(string(brut)) {
		ligne = strings.TrimSpace(ligne)
		if strings.HasPrefix(ligne, "## De l'anglais") {
			inverse = true
		}
		if !strings.HasPrefix(ligne, "| ") || strings.HasPrefix(ligne, "| Terme") || strings.HasPrefix(ligne, "| Anglais") {
			continue
		}
		cellules := strings.Split(strings.Trim(ligne, "| "), " | ")
		if inverse {
			inverses = append(inverses, [2]string{cellules[0], cellules[1]})
			continue
		}
		if len(cellules) != 4 {
			t.Fatalf("ligne à %d cellules : %s", len(cellules), ligne)
		}
		e := entree{terme: cellules[0], anglais: cellules[1]}
		if m := lienRe.FindStringSubmatch(cellules[3]); m != nil {
			e.lien = m[1]
		} else {
			t.Errorf("%s : renvoi illisible %q", e.terme, cellules[3])
		}
		principales = append(principales, e)
	}
	return principales, inverses
}

func TestOrdreAlphabetique(t *testing.T) {
	principales, inverses := lit(t)
	if len(principales) < 50 {
		t.Fatalf("%d entrées seulement", len(principales))
	}
	for i := 1; i < len(principales); i++ {
		if cle(principales[i-1].terme) >= cle(principales[i].terme) {
			t.Errorf("%q devrait venir après %q", principales[i-1].terme, principales[i].terme)
		}
	}
	for i := 1; i < len(inverses); i++ {
		if cle(inverses[i-1][0]) > cle(inverses[i][0]) {
			t.Errorf("table inverse : %q devrait venir après %q", inverses[i-1][0], inverses[i][0])
		}
	}
}

// Chaque entrée est sous la bonne lettre : la section « ## A » ne contient
// que des termes en A.
func TestLettres(t *testing.T) {
	brut, err := os.ReadFile(filepath.Join(docs, "annexes", "glossaire.md"))
	if err != nil {
		t.Fatal(err)
	}
	lettre := ""
	for ligne := range strings.Lines(string(brut)) {
		if l, ok := strings.CutPrefix(ligne, "## "); ok {
			lettre = strings.TrimSpace(l)
			continue
		}
		if len(lettre) != 1 || !strings.HasPrefix(ligne, "| ") || strings.HasPrefix(ligne, "| Terme") || strings.HasPrefix(ligne, "|--") {
			continue
		}
		terme := strings.SplitN(strings.Trim(ligne, "| \n"), " | ", 2)[0]
		if !strings.HasPrefix(cle(terme), strings.ToLower(lettre)) {
			t.Errorf("%q est sous la lettre %s", terme, lettre)
		}
	}
}

// La table inverse contient exactement les termes que le livre traduit.
func TestTableInverse(t *testing.T) {
	principales, inverses := lit(t)
	var attendues [][2]string
	for _, e := range principales {
		if cle(e.terme) != cle(e.anglais) {
			attendues = append(attendues, [2]string{e.anglais, e.terme})
		}
	}
	trie := func(s [][2]string) {
		slices.SortFunc(s, func(a, b [2]string) int { return cmp.Compare(cle(a[0])+a[1], cle(b[0])+b[1]) })
	}
	trie(attendues)
	inv := slices.Clone(inverses)
	trie(inv)
	if !slices.Equal(attendues, inv) {
		for _, a := range attendues {
			if !slices.Contains(inv, a) {
				t.Errorf("manque dans la table inverse : %s → %s", a[0], a[1])
			}
		}
		for _, a := range inv {
			if !slices.Contains(attendues, a) {
				t.Errorf("en trop dans la table inverse : %s → %s", a[0], a[1])
			}
		}
	}
}

// Chaque renvoi mène à un chapitre qui existe, et qui emploie le terme (en
// français ou en anglais) : le glossaire ne renvoie pas au mauvais endroit.
func TestRenvois(t *testing.T) {
	principales, _ := lit(t)
	for _, e := range principales {
		brut, err := os.ReadFile(filepath.Join(docs, e.lien+".md"))
		if err != nil {
			t.Errorf("%s : chapitre introuvable %s", e.terme, e.lien)
			continue
		}
		texte := cle(string(brut))
		if !strings.Contains(texte, cle(e.terme)) && !strings.Contains(texte, cle(e.anglais)) {
			t.Errorf("%s (%s) : absent de %s", e.terme, e.anglais, e.lien)
		}
	}
}
