package concurrence

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

func Example_pipeline() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nombres := Genere(ctx, 5)
	carres := Etape(ctx, nombres, func(n int) int { return n * n })
	textes := Etape(ctx, carres, func(n int) string { return fmt.Sprint(n) })

	var morceaux []string
	for t := range textes {
		morceaux = append(morceaux, t)
	}
	fmt.Println(strings.Join(morceaux, " → "))
	// Output: 1 → 4 → 9 → 16 → 25
}

func Example_annulation() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	somme := 0
	for n := range Genere(ctx, 1_000_000) {
		somme += n
		if n == 10 {
			cancel() // le générateur s'arrête au prochain envoi
			break
		}
	}
	fmt.Println(somme)
	// Output: 55
}

func Example_fanIn() {
	ctx := context.Background()
	a := Genere(ctx, 3)
	b := Etape(ctx, Genere(ctx, 3), func(n int) int { return n * 100 })

	var tous []int
	for v := range Fusionne(ctx, a, b) {
		tous = append(tous, v)
	}
	slices.Sort(tous) // l'ordre d'arrivée n'est pas déterministe
	fmt.Println(tous)
	// Output: [1 2 3 100 200 300]
}

func Example_pool() {
	ctx := context.Background()
	urls := []string{"a.example", "b.example", "erreur.example", "c.example"}

	telecharge := func(ctx context.Context, url string) (int, error) {
		if strings.HasPrefix(url, "erreur") {
			return 0, errors.New("connexion refusée")
		}
		return len(url), nil
	}

	var lignes []string
	for r := range Pool(ctx, 3, urls, telecharge) {
		if r.Err != nil {
			lignes = append(lignes, fmt.Sprintf("%s : %v", r.Entree, r.Err))
			continue
		}
		lignes = append(lignes, fmt.Sprintf("%s : %d octets", r.Entree, r.Sortie))
	}
	slices.Sort(lignes)
	fmt.Println(strings.Join(lignes, "\n"))
	// Output:
	// a.example : 9 octets
	// b.example : 9 octets
	// c.example : 9 octets
	// erreur.example : connexion refusée
}

func Example_errgroup() {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(2) // au plus deux tâches simultanées

	var annulees atomic.Int32
	for i := range 6 {
		g.Go(func() error {
			if i == 2 {
				return fmt.Errorf("tâche %d : échec", i)
			}
			select {
			case <-time.After(50 * time.Millisecond):
				return nil
			case <-ctx.Done(): // l'échec de la tâche 2 annule les autres
				annulees.Add(1)
				return ctx.Err()
			}
		})
	}
	err := g.Wait()
	fmt.Println(err, annulees.Load() > 0)
	// Output: tâche 2 : échec true
}

func Example_semaphore() {
	const limite = 3
	sem := make(chan struct{}, limite)
	var enCours, maximum atomic.Int32

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			sem <- struct{}{}        // acquiert une place
			defer func() { <-sem }() // la libère

			n := enCours.Add(1)
			for {
				m := maximum.Load()
				if n <= m || maximum.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			enCours.Add(-1)
		})
	}
	wg.Wait()
	fmt.Println(maximum.Load() <= limite)
	// Output: true
}

func Example_singleflight() {
	var groupe singleflight.Group
	var appels atomic.Int32
	charge := func() (any, error) {
		appels.Add(1)
		time.Sleep(20 * time.Millisecond) // une requête lente
		return "configuration", nil
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { groupe.Do("config", charge) })
	}
	wg.Wait()
	fmt.Println(appels.Load() < 10)
	// Output: true
}

// TestLimiteurDeDebit vérifie, avec une horloge virtuelle, qu'un limiteur à
// 10 événements par seconde (rafale de 1) traite 20 requêtes en 1,9 s.
func TestLimiteurDeDebit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiteur := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
		debut := time.Now()
		for range 20 {
			if err := limiteur.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		if ecoule := time.Since(debut); ecoule != 1900*time.Millisecond {
			t.Errorf("durée = %v, attendu 1,9 s", ecoule)
		}
	})
}

// TestTicker fait la même chose avec la seule bibliothèque standard.
func TestTicker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tic := time.NewTicker(100 * time.Millisecond)
		defer tic.Stop()
		debut := time.Now()
		for range 5 {
			<-tic.C
		}
		if ecoule := time.Since(debut); ecoule != 500*time.Millisecond {
			t.Errorf("durée = %v, attendu 500 ms", ecoule)
		}
	})
}

// fuit lance une goroutine qui attend sur un channel que personne n'écrira.
func fuit() {
	ch := make(chan int)
	go func() { <-ch }()
}

// TestDetectionDeFuite utilise le profil goroutineleak de Go 1.27.
func TestDetectionDeFuite(t *testing.T) {
	fuit()
	runtime.GC()

	var b strings.Builder
	if err := pprof.Lookup("goroutineleak").WriteTo(&b, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "concurrence.fuit") {
		t.Errorf("fuite non détectée :\n%s", b.String())
	}
}

// TestPipelineSansFuite vérifie qu'un pipeline annulé ne laisse aucune
// goroutine bloquée.
func TestPipelineSansFuite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		sortie := Etape(ctx, Genere(ctx, 1000), func(n int) int { return n * 2 })
		<-sortie
		cancel()
		synctest.Wait() // toutes les goroutines de la bulle doivent être terminées
	})
}
