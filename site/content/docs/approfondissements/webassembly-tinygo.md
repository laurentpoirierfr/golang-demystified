---
title: "WebAssembly et TinyGo : Go partout"
linkTitle: "16. WebAssembly et TinyGo"
weight: 160
---

[WebAssembly](https://webassembly.org) (Wasm) est un format de code binaire
portable : un module compilé une fois s'exécute dans un navigateur, sur un
serveur, dans un moteur embarqué, en toute sécurité, parce qu'il ne voit que
sa propre mémoire. Go sait compiler vers WebAssembly, et
[TinyGo](https://tinygo.org), un autre compilateur Go, produit des binaires
bien plus petits, jusqu'à tenir sur un microcontrôleur.

Ce chapitre fait tourner **un même package Go**, une analyse de texte (mots,
phrases, temps de lecture, mots fréquents), à quatre endroits :

| Où | Cible | Compilateur |
|----|-------|-------------|
| Dans le navigateur | `GOOS=js GOARCH=wasm` | Go ou TinyGo |
| Dans un greffon, chargé par un programme Go | `GOOS=wasip1 GOARCH=wasm` | Go ou TinyGo |
| Sur le serveur, en natif | `linux/amd64`… | Go |
| Sur un microcontrôleur | `pico`, `arduino-nano33`, `esp32c3`… | TinyGo |

Le code est dans [`examples/approfondissements/wasm`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/wasm) :

```
wasm/
├── pkg/texte/          l'analyse de texte, sans dépendance
├── cmd/navigateur/     exposée au JavaScript (syscall/js)
├── web/                la page de démonstration
├── cmd/site/           un serveur pour la page
├── cmd/greffon/        un greffon WASI (//go:wasmexport)
├── internal/hote/      l'hôte qui charge les greffons (wazero)
└── cmd/clignotant/     le même code sur un microcontrôleur (TinyGo)
```

Le package `texte` ne fait rien de particulier : il utilise `strings`,
`unicode`, `slices`, `cmp`. C'est tout l'intérêt : du Go ordinaire, testé
comme d'habitude, que l'on compile ensuite pour d'autres cibles.

## Dans le navigateur

### Compiler

```bash
GOOS=js GOARCH=wasm go build -o web/analyse.wasm ./cmd/navigateur
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
```

`wasm_exec.js` est le lien entre le runtime Go et JavaScript ; il doit venir
de la **même version** de Go que le compilateur. Depuis Go 1.24, il se trouve
dans `lib/wasm` (il était auparavant dans `misc/wasm`).

### Exposer des fonctions au JavaScript

Le package `syscall/js` manipule les valeurs JavaScript depuis Go.
`js.FuncOf` transforme une fonction Go en fonction JavaScript :

```go
//go:build js && wasm

func main() {
	analyse := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) != 1 || args[0].Type() != js.TypeString {
			return js.ValueOf(map[string]any{"erreur": "analyseTexte attend un texte"})
		}
		st := texte.Analyse(args[0].String(), 5)
		// js.ValueOf convertit les types simples, les []any et les
		// map[string]any en valeurs JavaScript.
		frequents := make([]any, len(st.Frequents))
		for i, f := range st.Frequents {
			frequents[i] = map[string]any{"mot": f.Mot, "nombre": f.Nombre}
		}
		return js.ValueOf(map[string]any{
			"mots": st.Mots, "phrases": st.Phrases, … "frequents": frequents,
		})
	})
	defer analyse.Release()
	js.Global().Set("analyseTexte", analyse)

	// Prévient la page que Go est prêt.
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("go-pret"))
	select {} // reste en vie
}
```

Deux détails comptent :

- le programme **reste en vie** (`select {}`) : s'il se terminait, la
  fonction enregistrée ne pourrait plus être appelée ;
- les arguments sont **vérifiés** : JavaScript peut appeler la fonction avec
  n'importe quoi, et une panique Go arrêterait le module entier.

Côté page, `wasm_exec.js` fournit la classe `Go` :

```js
const go = new Go();
WebAssembly.instantiateStreaming(fetch("analyse.wasm"), go.importObject)
  .then((r) => go.run(r.instance));

addEventListener("go-pret", () => {
  zone.addEventListener("input", () => {
    const st = globalThis.analyseTexte(zone.value); // une fonction Go
    …
  });
});
```

`instantiateStreaming` compile le module pendant son téléchargement, à
condition qu'il soit servi avec le type `application/wasm`. Le serveur de
fichiers de Go le fait :

```bash
go run ./cmd/site      # http://localhost:8080
```

La page a été vérifiée dans Chromium, piloté par Puppeteer : le module se
charge en 75 ms, l'analyse suit la saisie, et un appel incorrect
(`analyseTexte(42)`) renvoie une erreur au lieu de faire planter le module.

```console
module chargé en 75 ms
initial  : 11 | 2 | 99 | 1 min | webassembly (2), compile (1), microcontrôleur (1), navigateur (1), partout (1)
saisi    : 11 | 3 | 64 | 1 min | gopher (3), creuse (2), dort (1), encore (1), puis (1)
appel direct : {"erreur":"analyseTexte attend un texte"}
erreurs : aucune
```

## Côté serveur : des greffons WebAssembly

WebAssembly n'est pas réservé au navigateur. **WASI** (*WebAssembly System
Interface*) donne aux modules un accès contrôlé au système : horloge, aléa,
fichiers et réseau si l'hôte les accorde. Go le cible depuis la version 1.21
(`GOOS=wasip1`).

Un usage particulièrement intéressant : les **greffons**. Une application
charge du code tiers (un filtre, une règle métier, une transformation),
écrit dans n'importe quel langage qui compile vers WebAssembly, et l'exécute
**dans un bac à sable**. C'est ainsi que fonctionnent les extensions d'Envoy,
de Redpanda ou de Zed.

### Exporter des fonctions : //go:wasmexport

Depuis Go 1.24, la directive `//go:wasmexport` exporte une fonction Go vers
l'hôte, et `-buildmode=c-shared` produit un **réacteur** : un module qui
s'initialise, puis attend les appels, sans exécuter de `main`.

WebAssembly n'échange que des **nombres**. Un texte passe donc par la mémoire
du module : l'hôte demande un tampon, y écrit le texte, appelle l'analyse, et
lit le résultat à l'adresse renvoyée.

```go
//go:build wasip1

// tampons garde une référence vers chaque tampon prêté à l'hôte : sans elle,
// le ramasse-miettes de Go pourrait réutiliser la mémoire pendant que l'hôte
// y écrit.
var tampons = map[uint32][]byte{}

//go:wasmexport alloue
func alloue(taille uint32) uint32 {
	b := make([]byte, max(taille, 1))
	adresse := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
	tampons[adresse] = b
	return adresse
}

//go:wasmexport libere
func libere(adresse uint32) {
	delete(tampons, adresse)
}

// analyse renvoie l'adresse et la longueur du résultat JSON, réunies dans un
// seul entier : adresse << 32 | longueur.
//
//go:wasmexport analyse
func analyse(adresse, n uint32) uint64 {
	tampon, ok := tampons[adresse]
	if !ok || int(n) > len(tampon) {
		return 0 // adresse inconnue : l'hôte reçoit un résultat vide
	}
	resultat, err := json.Marshal(texte.Analyse(string(tampon[:n]), 5))
	…
	sortie := alloue(uint32(len(resultat)))
	copy(tampons[sortie], resultat)
	return uint64(sortie)<<32 | uint64(len(resultat))
}

func main() {} // non appelée en mode réacteur, mais obligatoire
```

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o greffon.wasm ./cmd/greffon
```

> [!NOTE]
> L'inverse existe aussi : `//go:wasmimport` (Go 1.21) déclare une fonction
> fournie par l'hôte, que le module peut appeler. Un hôte peut ainsi offrir à
> ses greffons un journal, une base clé-valeur ou un accès HTTP filtré.

### L'hôte : wazero

[wazero](https://wazero.io) est un moteur WebAssembly écrit en **Go pur**,
sans cgo : il se compile partout où Go se compile, et s'importe comme
n'importe quel module.

```go
r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
	WithMemoryLimitPages(o.MemoirePages).
	// Un contexte annulé (délai dépassé) interrompt le greffon, même
	// s'il boucle à l'infini.
	WithCloseOnContextDone(true))
// WASI : l'interface système dont le runtime Go a besoin. Aucun répertoire
// n'est monté : le greffon ne voit aucun fichier.
wasi_snapshot_preview1.MustInstantiate(ctx, r)

compile, err := r.CompileModule(ctx, wasm)
…
// Un réacteur s'initialise par _initialize, puis attend les appels.
module, err := r.InstantiateModule(ctx, compile, wazero.NewModuleConfig().
	WithName("greffon").
	WithStartFunctions("_initialize"))
```

L'appel suit le protocole du greffon : allouer, écrire, appeler, lire,
libérer.

```go
res, err := g.alloue.Call(ctx, uint64(len(s)))
entree := uint32(res[0])
defer g.rend(ctx, entree)
if !g.module.Memory().Write(entree, []byte(s)) {
	return st, errors.New("adresse hors de la mémoire du greffon")
}

res, err = g.analyse.Call(ctx, uint64(entree), uint64(len(s)))
sortie, n := uint32(res[0]>>32), uint32(res[0])
defer g.rend(ctx, sortie)

// Read renvoie une vue sur la mémoire du greffon : on la décode avant de
// libérer le tampon.
donnees, ok := g.module.Memory().Read(sortie, n)
…
err = json.Unmarshal(donnees, &st)
```

Une instance WebAssembly n'est pas sûre pour plusieurs goroutines : l'hôte
sérialise les appels avec un `sync.Mutex`, et un test en lance huit en
parallèle.

### Le bac à sable, mis à l'épreuve

Un greffon malveillant, dans `testdata`, essaie deux attaques : une boucle
infinie, et l'épuisement de la mémoire.

```go
//go:wasmexport analyse
func analyse(_, n uint32) uint64 {
	if n == 1 { // boucle infinie
		for {
		}
	}
	for { // sinon : demande toujours plus de mémoire
		garde = append(garde, make([]byte, 1<<20))
	}
}
```

Les deux sont contenues :

```console
interrompu en 201ms : analyse : module closed with context deadline exceeded
arrêté à 512 pages : analyse : wasm error: unreachable
```

La boucle infinie s'arrête dès l'échéance du contexte (200 ms), et le
greffon qui réclame toute la mémoire se heurte à la limite fixée par l'hôte
(512 pages de 64 Kio, soit 32 Mio). Le processus hôte, lui, continue.

### Les fuites de mémoire

Un tampon jamais libéré resterait dans la mémoire du greffon, qui ne
**rétrécit jamais** : WebAssembly ne sait que l'agrandir. Le test mesure donc
la mémoire avant et après des milliers d'appels.

> [!TIP]
> La première version de ce test a conclu à tort à une fuite : 56 pages au
> départ, 104 après mille appels. Un profil a montré que la mémoire plafonnait
> à 130 pages, de 5 000 à 20 000 appels : c'était le tas de Go qui atteignait
> son régime de croisière. Le test mesure désormais **après un échauffement**,
> avec un seuil tiré de ce que serait une vraie fuite (1,1 Kio par appel, une
> cinquantaine de pages pour 3 000 appels). Et pour s'assurer qu'il détecte
> bien une fuite, on a désactivé `libere` dans le greffon : le test a échoué,
> avec 136 puis 232 pages.

## TinyGo

[TinyGo](https://tinygo.org) est un compilateur Go fondé sur LLVM, pensé pour
les petites cibles : WebAssembly, microcontrôleurs. Il implémente le langage
et une grande partie de la bibliothèque standard, avec un runtime bien plus
léger. La version 0.44 s'appuie sur Go 1.27.

On l'utilise sans l'installer, dans son conteneur :

```bash
docker run --rm -v "$PWD:/src" -w /src tinygo/tinygo:0.44.0 \
  tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o greffon.wasm ./cmd/greffon
```

Le `Makefile` du dépôt le fait (`make wasm-tinygo`), puis fait charger le
greffon par le même hôte wazero : il renvoie exactement les mêmes résultats.

### La taille, et ce qu'elle coûte

Le même code source, compilé de quatre façons, sans information de débogage
des deux côtés (`-ldflags="-s -w"` pour Go, `-no-debug` pour TinyGo) :

| Module | Go | TinyGo | Rapport |
|--------|----|--------|---------|
| Page web (`syscall/js`) | 2,0 Mo | 104 Kio | ≈ 20 × |
| Greffon WASI (avec `encoding/json`) | 4,2 Mo | 985 Kio | ≈ 4,4 × |

L'écart dépend du code : `encoding/json`, qui repose sur la réflexion, pèse
lourd même avec TinyGo. Et la petite taille a un prix, mesuré par un banc
d'essai sur le greffon :

```console
BenchmarkGreffon/go-16        2000     952234 ns/op
BenchmarkGreffon/tinygo-16    2000    2379165 ns/op
```

Le greffon TinyGo est ici **2,5 fois plus lent** : son ramasse-miettes est
plus simple, et ses tables de hachage et sa réflexion moins optimisées que
celles du runtime Go. Le choix se fait selon ce qui compte : le poids à
télécharger dans un navigateur, ou le débit d'un greffon côté serveur.

> [!WARNING]
> Le `wasm_exec.js` de TinyGo n'est **pas** celui de Go. Une page doit
> charger celui du compilateur qui a produit le module :
> `$(tinygo env TINYGOROOT)/targets/wasm_exec.js` pour TinyGo. La page de
> démonstration, compilée par TinyGo avec son `wasm_exec.js`, fonctionne à
> l'identique.

### Sur un microcontrôleur

TinyGo cible aussi plus de cent cartes : Raspberry Pi Pico, Arduino, ESP32,
micro:bit… Le package `machine` donne accès aux broches, à l'I2C, au SPI.
Le programme fait clignoter la LED de la carte autant de fois qu'une phrase,
analysée **sur la carte**, compte de mots :

```go
//go:build tinygo

func main() {
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	st := texte.Analyse("Le gopher creuse. Le gopher dort.", 1)

	for {
		for range st.Mots {
			led.High()
			time.Sleep(150 * time.Millisecond)
			led.Low()
			time.Sleep(250 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)
	}
}
```

La contrainte `//go:build tinygo` écarte ce fichier du compilateur Go
standard, qui ne connaît pas le package `machine`. Le même code se compile
pour trois architectures différentes :

```console
$ tinygo build -target=pico -size short ./cmd/clignotant
   code    data     bss |   flash     ram
  24888    7196    5616 |   32084   12812
```

| Carte | Processeur | Flash | RAM |
|-------|------------|-------|-----|
| Raspberry Pi Pico | ARM Cortex-M0+ (RP2040) | 32 Kio | 12,5 Kio |
| Arduino Nano 33 IoT | ARM Cortex-M0+ (SAMD21) | 31 Kio | 12 Kio |
| ESP32-C3 | RISC-V | 30 Kio | 22 Kio |

Moins de 32 Kio de mémoire flash pour un programme qui analyse du texte en
Unicode, avec des tables de hachage et des tris. `tinygo flash -target=pico`
le transfère sur la carte branchée en USB.

> [!NOTE]
> Les programmes pour microcontrôleur de ce chapitre ont été compilés, mais
> pas essayés sur une vraie carte. Les tailles sont celles que rapporte TinyGo.

## Quand utiliser quoi

| Besoin | Solution |
|--------|----------|
| Partager une logique métier entre serveur et navigateur | `GOOS=js`, et TinyGo si le poids compte |
| Exécuter du code tiers en sécurité | greffons WASI, hôte wazero |
| Étendre une application sans recompiler | greffons WASI chargés à l'exécution |
| Un objet connecté, un capteur | TinyGo sur microcontrôleur |
| Une interface web riche | JavaScript ou TypeScript restent plus adaptés : Go y apporte du calcul, pas l'interface |

## En résumé

- Un même package Go, sans dépendance particulière, tourne en natif, dans le
  navigateur, dans un greffon et sur un microcontrôleur.
- **Navigateur** : `GOOS=js GOARCH=wasm`, `syscall/js` et `wasm_exec.js`, de
  la même version que le compilateur.
- **Greffons** : `GOOS=wasip1`, `//go:wasmexport` et `-buildmode=c-shared`
  (Go 1.24) ; les données passent par la mémoire du module.
- **wazero** héberge les greffons en Go pur, dans un bac à sable : mémoire
  bornée, interruption par le contexte, aucun fichier ni réseau par défaut.
- **TinyGo** divise la taille des modules (de 4 à 20 fois ici), au prix de la
  vitesse d'exécution, et ouvre les microcontrôleurs.
- **Mesurer** avant de conclure : la fuite apparente n'en était pas une, et
  le test a été vérifié contre une vraie fuite.
