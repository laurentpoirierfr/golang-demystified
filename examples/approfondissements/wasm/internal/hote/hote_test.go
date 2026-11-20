package hote_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/internal/hote"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

// construit compile un greffon avec Go, pour WASI, en mode réacteur.
func construit(t *testing.T, paquet string) []byte {
	t.Helper()
	sortie := filepath.Join(t.TempDir(), "greffon.wasm")
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", sortie, paquet)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation de %s : %v\n%s", paquet, err, out)
	}
	wasm, err := os.ReadFile(sortie)
	if err != nil {
		t.Fatal(err)
	}
	return wasm
}

var (
	unGreffon     []byte
	unGreffonUnic sync.Once
)

func greffon(t *testing.T) []byte {
	unGreffonUnic.Do(func() { unGreffon = construit(t, "../../cmd/greffon") })
	return unGreffon
}

func charge(t *testing.T, wasm []byte, o hote.Options) *hote.Greffon {
	t.Helper()
	g, err := hote.Charge(t.Context(), wasm, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close(context.Background()) })
	return g
}

func TestAnalyse(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un greffon WebAssembly")
	}
	g := charge(t, greffon(t), hote.Options{})
	texteSource := "Go compile vers WebAssembly. Le WebAssembly tourne partout : navigateur, serveur, microcontrôleur !"
	st, err := g.Analyse(t.Context(), texteSource)
	if err != nil {
		t.Fatal(err)
	}
	// Le greffon et le code natif donnent le même résultat.
	natif := texte.Analyse(texteSource, 5)
	if st.Mots != natif.Mots || st.Phrases != natif.Phrases || len(st.Frequents) != len(natif.Frequents) || st.Frequents[0] != natif.Frequents[0] {
		t.Errorf("greffon %+v\nnatif   %+v", st, natif)
	}

	// Les tampons sont libérés. La mémoire d'un module WebAssembly ne
	// rétrécit jamais, et le tas de Go grandit avant d'atteindre son régime
	// de croisière : on mesure donc après un échauffement. Une fuite ferait
	// grandir la mémoire avec le nombre d'appels : environ 1,1 Kio par appel
	// (le texte et le résultat), soit une cinquantaine de pages pour 3 000
	// appels. Le seuil de 16 pages (1 Mio) sépare nettement les deux cas.
	long := strings.Repeat("Encore une phrase. ", 50)
	appels := func(n int) {
		for range n {
			if _, err := g.Analyse(t.Context(), long); err != nil {
				t.Fatal(err)
			}
		}
	}
	appels(2000)
	avant := memoire(t, g)
	appels(3000)
	if apres := memoire(t, g); apres > avant+16 {
		t.Errorf("fuite de mémoire : %d puis %d pages", avant, apres)
	}
}

// memoire renvoie la taille de la mémoire du greffon, en pages.
func memoire(t *testing.T, g *hote.Greffon) uint32 {
	t.Helper()
	return g.Pages()
}

func TestTexteVide(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un greffon WebAssembly")
	}
	g := charge(t, greffon(t), hote.Options{})
	st, err := g.Analyse(t.Context(), "")
	if err != nil || st.Mots != 0 || st.Frequents == nil {
		t.Errorf("%+v %v", st, err)
	}
}

func TestAppelsConcurrents(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un greffon WebAssembly")
	}
	g := charge(t, greffon(t), hote.Options{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				if st, err := g.Analyse(t.Context(), "Un deux trois."); err != nil || st.Mots != 3 {
					t.Errorf("%+v %v", st, err)
					return
				}
			}
		})
	}
	wg.Wait()
}

// Un greffon qui boucle est interrompu par l'annulation du contexte.
func TestBoucleInfinie(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un greffon WebAssembly")
	}
	g := charge(t, construit(t, "./testdata/boucle"), hote.Options{})
	ctx, annule := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer annule()
	debut := time.Now()
	_, err := g.Analyse(ctx, "x") // un octet : la boucle infinie
	if err == nil {
		t.Fatal("la boucle infinie aurait dû être interrompue")
	}
	if d := time.Since(debut); d > 2*time.Second {
		t.Errorf("interrompu en %v", d)
	}
	t.Logf("interrompu en %v : %v", time.Since(debut).Round(time.Millisecond), err)
}

// Un greffon qui veut toute la mémoire se heurte à la limite fixée par l'hôte.
func TestMemoireBornee(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un greffon WebAssembly")
	}
	g := charge(t, construit(t, "./testdata/boucle"), hote.Options{MemoirePages: 512}) // 32 Mio
	_, err := g.Analyse(t.Context(), "xx")
	if err == nil {
		t.Fatal("l'épuisement de la mémoire aurait dû échouer")
	}
	if g.Pages() > 512 {
		t.Errorf("%d pages, au-delà de la limite", g.Pages())
	}
	t.Logf("arrêté à %d pages : %v", g.Pages(), firstLine(err.Error()))
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(s, "\n")
	return l
}

func TestModuleInvalide(t *testing.T) {
	if _, err := hote.Charge(t.Context(), []byte("pas du WebAssembly"), hote.Options{}); err == nil {
		t.Error("un module invalide doit être refusé")
	}
}

// TestTinyGo charge le même greffon, compilé par TinyGo (voir le Makefile :
// make wasm-tinygo), si son chemin est donné dans GREFFON_TINYGO.
func TestTinyGo(t *testing.T) {
	chemin := os.Getenv("GREFFON_TINYGO")
	if chemin == "" {
		t.Skip("GREFFON_TINYGO non défini")
	}
	wasm, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	g := charge(t, wasm, hote.Options{})
	for _, s := range []string{"", "Go compile vers WebAssembly. Le WebAssembly tourne partout !", strings.Repeat("Encore une phrase. ", 200)} {
		st, err := g.Analyse(t.Context(), s)
		if err != nil {
			t.Fatal(err)
		}
		natif := texte.Analyse(s, 5)
		if st.Mots != natif.Mots || st.Phrases != natif.Phrases || st.Caracteres != natif.Caracteres || len(st.Frequents) != len(natif.Frequents) {
			t.Errorf("TinyGo %+v\nnatif  %+v", st, natif)
		}
	}
}

// BenchmarkGreffon compare les greffons compilés par Go et par TinyGo.
func BenchmarkGreffon(b *testing.B) {
	modules := map[string][]byte{"go": nil}
	if chemin := os.Getenv("GREFFON_TINYGO"); chemin != "" {
		modules["tinygo"], _ = os.ReadFile(chemin)
	}
	long := strings.Repeat("Encore une phrase. ", 50)
	for nom, wasm := range modules {
		b.Run(nom, func(b *testing.B) {
			if wasm == nil {
				wasm = construitB(b, "../../cmd/greffon")
			}
			g, err := hote.Charge(b.Context(), wasm, hote.Options{})
			if err != nil {
				b.Fatal(err)
			}
			defer g.Close(context.Background())
			for b.Loop() {
				if _, err := g.Analyse(b.Context(), long); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func construitB(b *testing.B, paquet string) []byte {
	sortie := filepath.Join(b.TempDir(), "greffon.wasm")
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", sortie, paquet)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		b.Fatalf("%v\n%s", err, out)
	}
	wasm, _ := os.ReadFile(sortie)
	return wasm
}
