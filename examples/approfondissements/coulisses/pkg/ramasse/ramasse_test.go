package ramasse_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/coulisses/pkg/ramasse"
)

// travail alloue 400 Mio en tout, par blocs de 64 Kio, en gardant les 32
// derniers Mio en vie : un service qui traite des requêtes.
func travail() {
	const bloc = 64 << 10
	vivants := make([][]byte, 512) // 512 × 64 Kio = 32 Mio
	for i := range 6400 {
		vivants[i%len(vivants)] = make([]byte, bloc)
	}
	runtime.KeepAlive(vivants)
}

func TestGOGC(t *testing.T) {
	if testing.Short() {
		t.Skip("alloue plusieurs centaines de Mio")
	}
	resultats := map[string]ramasse.Mesure{}
	for _, cas := range []struct {
		nom    string
		gogc   int
		limite int64
	}{
		{"GOGC=50", 50, 0},
		{"GOGC=100", 100, 0},
		{"GOGC=400", 400, 0},
		{"GOGC=off, GOMEMLIMIT=64Mio", -1, 64 << 20},
	} {
		m := ramasse.Observe(cas.gogc, cas.limite, travail)
		resultats[cas.nom] = m
		t.Logf("%-28s %3d cycles, objectif du tas %4d Mio", cas.nom, m.Cycles, m.ObjectifMo)
	}
	// Plus GOGC est grand, moins le GC passe souvent.
	if resultats["GOGC=50"].Cycles <= resultats["GOGC=100"].Cycles || resultats["GOGC=100"].Cycles <= resultats["GOGC=400"].Cycles {
		t.Errorf("les cycles devraient décroître quand GOGC augmente : %+v", resultats)
	}
	// Sans GC proportionnel, seule la limite de mémoire déclenche le GC.
	if m := resultats["GOGC=off, GOMEMLIMIT=64Mio"]; m.Cycles == 0 || m.ObjectifMo > 64 {
		t.Errorf("GOMEMLIMIT : %+v", m)
	}
}

type Image struct{ pixels []byte }

func TestCacheFaible(t *testing.T) {
	c := ramasse.NouveauCache[string, Image]()
	garde := &Image{pixels: make([]byte, 1<<20)}
	c.Met("logo", garde)
	c.Met("fond", &Image{pixels: make([]byte, 1<<20)}) // personne ne la retient

	// Le GC libère l'image que personne ne retient ; le nettoyage, lancé dans
	// une autre goroutine, retire son entrée.
	limite := time.Now().Add(2 * time.Second)
	for c.Taille() > 1 && time.Now().Before(limite) {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	if c.Prend("fond") != nil || c.Taille() != 1 {
		t.Errorf("l'image « fond » aurait dû être libérée ; taille %d", c.Taille())
	}
	if c.Prend("logo") != garde {
		t.Error("l'image « logo », encore utilisée, doit rester dans le cache")
	}
	runtime.KeepAlive(garde)
}
