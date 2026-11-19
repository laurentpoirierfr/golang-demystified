package horloge_test

import (
	"fmt"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/horloge"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/hexagonale/pkg/identifiant"
)

func ExampleManuelle() {
	h := horloge.NouvelleManuelle(time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC))
	h.Avance(21 * 24 * time.Hour)
	fmt.Println(h.Maintenant().Format(time.DateOnly))
	// Output: 2026-09-22
}

func ExampleSequence() {
	ids := &identifiant.Sequence{Prefixe: "e"}
	fmt.Println(ids.Nouveau(), ids.Nouveau())
	// Output: e1 e2
}
