package texte_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

func ExampleAnalyse() {
	st := texte.Analyse("Go compile vers WebAssembly. Le WebAssembly tourne partout : navigateur, serveur, microcontrôleur !", 2)
	b, _ := json.Marshal(st)
	fmt.Println(string(b))
	// Output:
	// {"caracteres":99,"mots":11,"phrases":2,"minutes_lecture":1,"frequents":[{"mot":"webassembly","nombre":2},{"mot":"compile","nombre":1}]}
}

func TestAnalyse(t *testing.T) {
	tests := []struct {
		texte                  string
		mots, phrases, minutes int
	}{
		{"", 0, 0, 0},
		{"   ", 0, 0, 0},
		{"Aujourd'hui, peut-être.", 2, 1, 1},
		{"Une phrase sans point", 4, 1, 1},
		{"Quoi ?! Vraiment... Oui.", 3, 3, 1},
		{strings.Repeat("mot ", 461), 461, 1, 3}, // 461 mots à 230 par minute
	}
	for _, tt := range tests {
		st := texte.Analyse(tt.texte, 3)
		if st.Mots != tt.mots || st.Phrases != tt.phrases || st.MinutesLecture != tt.minutes {
			t.Errorf("%.30q : %d mots, %d phrases, %d min ; attendu %d, %d, %d",
				tt.texte, st.Mots, st.Phrases, st.MinutesLecture, tt.mots, tt.phrases, tt.minutes)
		}
		if st.Frequents == nil {
			t.Errorf("%.30q : Frequents ne doit jamais être nil", tt.texte)
		}
	}
}
