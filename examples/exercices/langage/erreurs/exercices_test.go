package erreurs

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestLitAge(t *testing.T) {
	if age, err := LitAge("42"); err != nil || age != 42 {
		t.Errorf("LitAge(42) = %d, %v", age, err)
	}
	if _, err := LitAge("200"); !errors.Is(err, ErrHorsLimites) {
		t.Errorf("200 : %v", err)
	}
	_, err := LitAge("douze")
	if !errors.Is(err, strconv.ErrSyntax) || !strings.Contains(err.Error(), `"douze"`) {
		t.Errorf("douze : %v", err)
	}
}

func TestValide(t *testing.T) {
	if err := Valide(Utilisateur{"Ada", "ada@exemple.fr"}); err != nil {
		t.Errorf("valide : %v", err)
	}
	err := Valide(Utilisateur{})
	var champ *ErrChamp
	if !errors.As(err, &champ) || champ.Champ != "nom" {
		t.Fatalf("errors.As : %v", err)
	}
	if err.Error() != "nom : obligatoire\nemail : adresse invalide" {
		t.Errorf("message : %q", err)
	}
}

func TestProtege(t *testing.T) {
	if err := Protege(func() {}); err != nil {
		t.Error(err)
	}
	err := Protege(func() {
		panic("boum")
	})
	if err == nil || !strings.HasPrefix(err.Error(), "panique : ") {
		t.Errorf("err = %v", err)
	}
}
