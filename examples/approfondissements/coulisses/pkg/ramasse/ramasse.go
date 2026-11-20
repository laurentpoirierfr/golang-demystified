// Package ramasse observe le ramasse-miettes (GC) de Go : son rythme selon
// GOGC et GOMEMLIMIT, mesuré par runtime/metrics, et les pointeurs faibles
// (package weak, Go 1.24), qui laissent le GC libérer ce qu'ils désignent.
package ramasse

import (
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"sync"
	"weak"
)

// Mesure résume l'activité du GC pendant un travail.
type Mesure struct {
	Cycles     uint64 // cycles de GC
	ObjectifMo uint64 // taille visée du tas, en Mio, à la fin
}

// Observe exécute travail avec les réglages donnés, et compte les cycles de
// GC. gogc est le GOGC à appliquer (100 par défaut ; -1 désactive le GC
// proportionnel) ; limite est le GOMEMLIMIT en octets (0 : aucune limite).
// Les réglages précédents sont rétablis à la fin.
func Observe(gogc int, limite int64, travail func()) Mesure {
	ancienGOGC := debug.SetGCPercent(gogc)
	defer debug.SetGCPercent(ancienGOGC)
	if limite > 0 {
		ancienneLimite := debug.SetMemoryLimit(limite)
		defer debug.SetMemoryLimit(ancienneLimite)
	}
	runtime.GC() // repart d'un tas propre

	echantillons := []metrics.Sample{
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/gc/heap/goal:bytes"},
	}
	metrics.Read(echantillons)
	avant := echantillons[0].Value.Uint64()
	travail()
	metrics.Read(echantillons)
	return Mesure{
		Cycles:     echantillons[0].Value.Uint64() - avant,
		ObjectifMo: echantillons[1].Value.Uint64() >> 20,
	}
}

// Cache associe des clés à des valeurs, sans les retenir : une valeur que
// plus personne n'utilise est libérée par le GC, et son entrée disparaît du
// cache. C'est l'usage typique des pointeurs faibles : un cache qui ne
// provoque pas de fuite de mémoire.
type Cache[K comparable, V any] struct {
	mu      sync.Mutex
	entrees map[K]weak.Pointer[V]
}

// NouveauCache crée un cache vide.
func NouveauCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{entrees: map[K]weak.Pointer[V]{}}
}

// Met range v sous la clé k.
func (c *Cache[K, V]) Met(k K, v *V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entrees[k] = weak.Make(v)
	// Quand v sera libérée, le runtime appellera ce nettoyage, dans une autre
	// goroutine. L'argument est la clé, jamais v : un nettoyage qui
	// retiendrait v l'empêcherait d'être libérée.
	runtime.AddCleanup(v, c.oublie, k)
}

// Prend renvoie la valeur, ou nil si elle a été libérée.
func (c *Cache[K, V]) Prend(k K) *V {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entrees[k].Value() // nil si absente ou libérée
}

// Taille renvoie le nombre d'entrées.
func (c *Cache[K, V]) Taille() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entrees)
}

func (c *Cache[K, V]) oublie(k K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// La clé a pu être réutilisée pour une nouvelle valeur entre-temps.
	if p, ok := c.entrees[k]; ok && p.Value() == nil {
		delete(c.entrees, k)
	}
}
