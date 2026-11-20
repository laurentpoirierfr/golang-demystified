package salon_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/temps-reel/internal/salon"
)

func TestDiffusionParSalon(t *testing.T) {
	h := salon.Nouveau()
	golang, _ := h.Abonne("golang", 0)
	rust, _ := h.Abonne("rust", 0)

	if _, err := h.Publie("golang", "alice", "  Bonjour  "); err != nil {
		t.Fatal(err)
	}
	m := <-golang.Messages()
	if m.ID != 1 || m.Texte != "Bonjour" || m.Auteur != "alice" {
		t.Errorf("message : %+v", m)
	}
	select {
	case m := <-rust.Messages():
		t.Errorf("message reçu dans le mauvais salon : %+v", m)
	default:
	}
}

func TestReprise(t *testing.T) {
	h := salon.Nouveau()
	for _, texte := range []string{"un", "deux", "trois"} {
		h.Publie("golang", "alice", texte)
	}
	h.Publie("rust", "bob", "ailleurs")

	// Le client a reçu le message 1 avant sa coupure : il reçoit 2 et 3.
	_, manques := h.Abonne("golang", 1)
	if len(manques) != 2 || manques[0].Texte != "deux" || manques[1].Texte != "trois" {
		t.Errorf("messages manqués : %+v", manques)
	}
}

func TestHistoriqueBorne(t *testing.T) {
	h := salon.Nouveau()
	for range salon.TailleHistorique + 50 {
		h.Publie("golang", "alice", "x")
	}
	_, manques := h.Abonne("golang", 1)
	if len(manques) != salon.TailleHistorique {
		t.Errorf("%d messages, attendu %d", len(manques), salon.TailleHistorique)
	}
}

func TestAbonneTropLent(t *testing.T) {
	h := salon.Nouveau()
	lent, _ := h.Abonne("golang", 0)
	rapide, _ := h.Abonne("golang", 0)

	recus := 0
	for range salon.TailleTampon + 1 {
		h.Publie("golang", "alice", "x")
		<-rapide.Messages() // l'abonné rapide lit au fil de l'eau
		recus++
	}
	// L'abonné lent n'a rien lu : il a été déconnecté, sans bloquer personne.
	n := 0
	for range lent.Messages() {
		n++
	}
	if n != salon.TailleTampon || recus != salon.TailleTampon+1 || h.Abonnes() != 1 {
		t.Errorf("lent : %d messages, rapide : %d, abonnés : %d", n, recus, h.Abonnes())
	}
}

func TestValidation(t *testing.T) {
	h := salon.Nouveau()
	if _, err := h.Publie("golang", "alice", "   "); !errors.Is(err, salon.ErrVide) {
		t.Errorf("vide : %v", err)
	}
	if _, err := h.Publie("golang", "alice", strings.Repeat("é", salon.LongueurMax+1)); !errors.Is(err, salon.ErrTropLong) {
		t.Errorf("trop long : %v", err)
	}
}

func TestDesabonneDeuxFois(t *testing.T) {
	h := salon.Nouveau()
	a, _ := h.Abonne("golang", 0)
	h.Desabonne(a)
	h.Desabonne(a) // sans panique
	if _, ok := <-a.Messages(); ok {
		t.Error("le canal devrait être fermé")
	}
}
