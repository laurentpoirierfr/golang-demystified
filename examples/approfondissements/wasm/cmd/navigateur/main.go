//go:build js && wasm

// Command navigateur expose l'analyse de texte au JavaScript d'une page web.
//
//	GOOS=js GOARCH=wasm go build -o web/analyse.wasm ./cmd/navigateur
//	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
//
// Le programme enregistre une fonction globale, analyseTexte, puis reste en
// vie : s'il se terminait, la fonction ne pourrait plus être appelée.
package main

import (
	"syscall/js"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

func main() {
	analyse := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return js.ValueOf(map[string]any{"erreur": "analyseTexte attend un texte"})
		}
		st := texte.Analyse(args[0].String(), 5)
		// js.ValueOf convertit les types simples, les []any et les
		// map[string]any en valeurs JavaScript.
		frequents := make([]any, len(st.Frequents))
		for i, f := range st.Frequents {
			frequents[i] = map[string]any{"mot": f.Mot, "nombre": f.Nombre}
		}
		return js.ValueOf(map[string]any{
			"caracteres":     st.Caracteres,
			"mots":           st.Mots,
			"phrases":        st.Phrases,
			"minutesLecture": st.MinutesLecture,
			"frequents":      frequents,
		})
	})
	defer analyse.Release()
	js.Global().Set("analyseTexte", analyse)

	// Prévient la page que Go est prêt.
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("go-pret"))
	select {} // reste en vie
}
