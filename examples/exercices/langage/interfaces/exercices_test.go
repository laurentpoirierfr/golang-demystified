package interfaces

import (
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
)

func TestAireTotale(t *testing.T) {
	total := AireTotale(Carre{2}, Cercle{1}, Carre{3})
	if math.Abs(total-(4+math.Pi+9)) > 1e-9 {
		t.Errorf("AireTotale = %v", total)
	}
	if AireTotale() != 0 {
		t.Error("aucune forme, aucune aire")
	}
}

func TestDuree(t *testing.T) {
	for d, attendu := range map[Duree]string{5: "5s", 65: "1m05s", 3725: "1h02m05s", 3600: "1h00m00s"} {
		if got := fmt.Sprint(d); got != attendu {
			t.Errorf("%d secondes : %q, attendu %q", int(d), got, attendu)
		}
	}
}

func TestMajuscules(t *testing.T) {
	r := Majuscules(strings.NewReader("le gopher creuse, l'été"))
	b, err := io.ReadAll(r)
	if err != nil || string(b) != "LE GOPHER CREUSE, L'ÉTÉ" {
		t.Errorf("%q, %v", b, err)
	}
}
