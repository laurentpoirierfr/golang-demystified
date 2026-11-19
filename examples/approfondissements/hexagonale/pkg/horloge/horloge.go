// Package horloge fournit des horloges interchangeables : l'horloge système
// pour la production, et une horloge manuelle pour les tests. Il ne dépend
// d'aucun projet : il est placé dans pkg/ pour être réutilisé.
package horloge

import (
	"sync"
	"time"
)

// Systeme donne l'heure réelle.
type Systeme struct{}

// Maintenant renvoie l'heure courante.
func (Systeme) Maintenant() time.Time { return time.Now() }

// Manuelle est une horloge que l'on fait avancer à la main. Elle est sûre
// pour un usage concurrent.
type Manuelle struct {
	mu sync.Mutex
	t  time.Time
}

// NouvelleManuelle crée une horloge arrêtée à l'instant t.
func NouvelleManuelle(t time.Time) *Manuelle { return &Manuelle{t: t} }

// Maintenant renvoie l'instant courant de l'horloge.
func (m *Manuelle) Maintenant() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.t
}

// Avance fait avancer l'horloge de d.
func (m *Manuelle) Avance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.t = m.t.Add(d)
}
