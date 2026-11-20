// Package hote charge et exécute des greffons WebAssembly avec wazero
// (https://wazero.io), un moteur WebAssembly écrit en Go pur, sans cgo.
//
// Un greffon s'exécute dans un bac à sable : il ne voit que sa propre
// mémoire, n'accède ni aux fichiers ni au réseau (sauf ce que l'hôte lui
// accorde), sa mémoire est bornée, et un contexte annulé l'interrompt. On
// peut donc charger un greffon écrit par un tiers, dans n'importe quel
// langage qui compile vers WebAssembly.
package hote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/wasm/pkg/texte"
)

// Options borne les ressources d'un greffon.
type Options struct {
	// MemoirePages limite la mémoire du greffon, en pages de 64 Kio
	// (0 : 1024 pages, soit 64 Mio).
	MemoirePages uint32
}

// Greffon est un greffon chargé. Une instance WebAssembly n'est pas sûre pour
// plusieurs goroutines : les appels sont sérialisés.
type Greffon struct {
	mu      sync.Mutex
	runtime wazero.Runtime
	module  api.Module
	alloue  api.Function
	libere  api.Function
	analyse api.Function
}

// Charge compile et instancie un greffon.
func Charge(ctx context.Context, wasm []byte, o Options) (*Greffon, error) {
	if o.MemoirePages == 0 {
		o.MemoirePages = 1024
	}
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithMemoryLimitPages(o.MemoirePages).
		// Un contexte annulé (délai dépassé) interrompt le greffon, même
		// s'il boucle à l'infini.
		WithCloseOnContextDone(true))
	// WASI : l'interface système (horloge, aléa, sorties) dont le runtime Go
	// a besoin. Aucun répertoire n'est monté : le greffon ne voit aucun
	// fichier.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	compile, err := r.CompileModule(ctx, wasm)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("compilation du greffon : %w", err)
	}
	// Un « réacteur » (-buildmode=c-shared) s'initialise par _initialize,
	// puis attend les appels ; il n'exécute pas de main.
	module, err := r.InstantiateModule(ctx, compile, wazero.NewModuleConfig().
		WithName("greffon").
		WithStartFunctions("_initialize"))
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("instanciation du greffon : %w", err)
	}
	g := &Greffon{runtime: r, module: module,
		alloue: module.ExportedFunction("alloue"), libere: module.ExportedFunction("libere"),
		analyse: module.ExportedFunction("analyse")}
	if g.alloue == nil || g.libere == nil || g.analyse == nil {
		r.Close(ctx)
		return nil, errors.New("le greffon doit exporter alloue, libere et analyse")
	}
	return g, nil
}

// Close libère le greffon et son moteur.
func (g *Greffon) Close(ctx context.Context) error { return g.runtime.Close(ctx) }

// Analyse confie le texte au greffon et décode son résultat.
func (g *Greffon) Analyse(ctx context.Context, s string) (texte.Stats, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var st texte.Stats

	// 1. Un tampon dans la mémoire du greffon, où l'on copie le texte.
	res, err := g.alloue.Call(ctx, uint64(len(s)))
	if err != nil {
		return st, fmt.Errorf("alloue : %w", err)
	}
	entree := uint32(res[0])
	defer g.rend(ctx, entree)
	if !g.module.Memory().Write(entree, []byte(s)) {
		return st, errors.New("adresse hors de la mémoire du greffon")
	}

	// 2. L'analyse ; le résultat est désigné par une adresse et une longueur.
	res, err = g.analyse.Call(ctx, uint64(entree), uint64(len(s)))
	if err != nil {
		return st, fmt.Errorf("analyse : %w", err)
	}
	sortie, n := uint32(res[0]>>32), uint32(res[0])
	if n == 0 {
		return st, errors.New("le greffon n'a rien renvoyé")
	}
	defer g.rend(ctx, sortie)

	// 3. Lecture du résultat. Read renvoie une vue sur la mémoire du greffon :
	// on la décode avant de libérer le tampon.
	donnees, ok := g.module.Memory().Read(sortie, n)
	if !ok {
		return st, errors.New("résultat hors de la mémoire du greffon")
	}
	if err := json.Unmarshal(donnees, &st); err != nil {
		return st, fmt.Errorf("résultat du greffon illisible : %w", err)
	}
	return st, nil
}

// Pages renvoie la taille de la mémoire du greffon, en pages de 64 Kio.
func (g *Greffon) Pages() uint32 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.module.Memory().Size() / 65536
}

// rend libère un tampon du greffon. Un échec n'appelle aucune réaction : il
// signifie que le module est déjà fermé (contexte annulé, erreur fatale), et
// sa mémoire disparaît avec lui.
func (g *Greffon) rend(ctx context.Context, adresse uint32) {
	_, _ = g.libere.Call(ctx, uint64(adresse))
}
