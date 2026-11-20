//go:build tinygo

// Command clignotant fait clignoter la LED d'une carte, et signale en morse
// (sur la même LED) le nombre de mots d'une phrase analysée sur la carte
// elle-même : le package texte tourne aussi sur un microcontrôleur.
//
//	tinygo flash -target=pico ./cmd/clignotant     # Raspberry Pi Pico
//	tinygo build -target=pico -o clignotant.uf2 ./cmd/clignotant
//
// La contrainte de build « tinygo » écarte ce fichier du compilateur Go
// standard, qui ne connaît pas le package machine.
package main

import (
	"machine"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

func main() {
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	st := texte.Analyse("Le gopher creuse. Le gopher dort.", 1)

	for {
		// Autant d'éclats courts que de mots, puis une pause.
		for range st.Mots {
			led.High()
			time.Sleep(150 * time.Millisecond)
			led.Low()
			time.Sleep(250 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)
	}
}
