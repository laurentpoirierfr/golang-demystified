package texte

import "testing"

func TestInitiales(t *testing.T) {
	for nom, attendu := range map[string]string{
		"ada lovelace": "A.L.", "  Émile   Zola ": "É.Z.", "Grace Brewster Hopper": "G.B.H.", "": "",
	} {
		if got := Initiales(nom); got != attendu {
			t.Errorf("Initiales(%q) = %q, attendu %q", nom, got, attendu)
		}
	}
}

func TestEstPalindrome(t *testing.T) {
	for s, attendu := range map[string]bool{
		"kayak": true, "Engage le jeu que je le gagne": true, "Ésope reste ici et se repose": false,
		"A man, a plan, a canal: Panama": true, "gopher": false, "": true,
	} {
		if EstPalindrome(s) != attendu {
			t.Errorf("EstPalindrome(%q) = %v", s, !attendu)
		}
	}
}

func TestTableau(t *testing.T) {
	got := Tableau([][]string{
		{"Langage", "Version", "Étoiles"},
		{"Go", "1.27", "130000"},
		{"Rust", "1.99", "105000"},
	})
	attendu := "Langage  Version  Étoiles\n" +
		"Go       1.27     130000\n" +
		"Rust     1.99     105000\n"
	if got != attendu {
		t.Errorf("Tableau :\n%s\nattendu :\n%s", got, attendu)
	}
}
