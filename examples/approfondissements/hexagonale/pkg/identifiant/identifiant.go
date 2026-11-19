// Package identifiant fournit des générateurs d'identifiants
// interchangeables : des UUID pour la production, une séquence prévisible
// pour les tests.
package identifiant

import (
	"fmt"
	"sync/atomic"
	"uuid"
)

// UUID produit des identifiants UUID v7, triables par date de création.
type UUID struct{}

// Nouveau renvoie un nouvel identifiant.
func (UUID) Nouveau() string { return uuid.NewV7().String() }

// Sequence produit des identifiants prévisibles : prefixe1, prefixe2…
type Sequence struct {
	Prefixe string
	n       atomic.Int64
}

// Nouveau renvoie l'identifiant suivant.
func (s *Sequence) Nouveau() string { return fmt.Sprintf("%s%d", s.Prefixe, s.n.Add(1)) }
