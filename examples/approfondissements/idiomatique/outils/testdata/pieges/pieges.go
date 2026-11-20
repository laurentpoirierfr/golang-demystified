// Package pieges contient des erreurs que le compilateur accepte, mais que
// go vet repère (voir outils_test.go).
package pieges

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Compteur est protégé par un verrou.
type Compteur struct {
	mu sync.Mutex
	n  int
}

// Valeur copie le compteur, et donc son verrou : il ne protège plus rien.
func Valeur(c Compteur) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// Annonce se trompe de verbe : %d pour une chaîne.
func Annonce(nom string) string {
	return fmt.Sprintf("bonjour %d", nom)
}

// Attend oublie d'appeler la fonction d'annulation : le contexte, et son
// minuteur, ne sont jamais libérés.
func Attend(ctx context.Context) error {
	ctx, _ = context.WithTimeout(ctx, time.Second)
	<-ctx.Done()
	return ctx.Err()
}

// Journalise jette le résultat de Sprintf, qui n'a pas d'effet de bord.
func Journalise(n int) {
	fmt.Sprintf("n = %d", n)
}
