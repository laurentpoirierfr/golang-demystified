package divers

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestEmails(t *testing.T) {
	texte := "Écrire à ada@exemple.fr ou à rob.pike+go@golang.org, pas à @exemple ni à moi@local."
	if got := Emails(texte); !slices.Equal(got, []string{"ada@exemple.fr", "rob.pike+go@golang.org"}) {
		t.Errorf("Emails = %q", got)
	}
}

func TestTotalParCategorie(t *testing.T) {
	csv := "produit,categorie,prix\nTasse,cuisine,8.5\nLivre,lecture,25\nBol,cuisine,6\n\"Carnet, petit\",lecture,4.5\n"
	totaux, err := TotalParCategorie(strings.NewReader(csv))
	if err != nil || !maps.Equal(totaux, map[string]float64{"cuisine": 14.5, "lecture": 29.5}) {
		t.Errorf("%v, %v", totaux, err)
	}
	_, err = TotalParCategorie(strings.NewReader("produit,categorie,prix\nTasse,cuisine,8\nBol,cuisine,six\n"))
	if err == nil || err.Error() != `ligne 3 : prix "six" invalide` {
		t.Errorf("err = %v", err)
	}
	if _, err := TotalParCategorie(strings.NewReader("produit,categorie,prix\nTasse,cuisine\n")); err == nil {
		t.Error("une ligne incomplète doit être refusée")
	}
}

func TestSigne(t *testing.T) {
	cle := []byte("cle-secrete")
	s := Signe(cle, "commande 42 : 19,90 €")
	if len(s) != 64 {
		t.Errorf("signature %q", s)
	}
	if !Verifie(cle, "commande 42 : 19,90 €", s) {
		t.Error("la signature doit être acceptée")
	}
	if Verifie(cle, "commande 42 : 1,90 €", s) || Verifie([]byte("autre"), "commande 42 : 19,90 €", s) || Verifie(cle, "x", "pas-hexa") {
		t.Error("une signature fausse doit être refusée")
	}
	// Valeur de référence (RFC 4231, cas 2) : la signature est bien HMAC-SHA256.
	if Signe([]byte("Jefe"), "what do ya want for nothing?") != "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843" {
		t.Error("ce n'est pas HMAC-SHA256")
	}
}
