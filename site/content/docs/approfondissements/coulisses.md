---
title: "Les coulisses de Go : réflexion, unsafe, cgo, ordonnanceur et GC"
linkTitle: "10. Les coulisses de Go"
weight: 100
---

Le chapitre « Profilage et performance » mesure ce que fait un programme. Ce
chapitre ouvre le capot : ce que le langage fait **pour** le programme, et les
portes qu'il laisse entrouvertes pour sortir de ses règles. On y trouve cinq
sujets, chacun avec son exemple testé et ses mesures :

| Sujet | Ce qu'il permet | Ce qu'il coûte |
|-------|-----------------|----------------|
| Réflexion (`reflect`) | traiter un type inconnu à la compilation | vitesse, sûreté du typage |
| `unsafe` | voir et manipuler la mémoire directement | toutes les garanties de Go |
| cgo | appeler du C, être appelé par du C | coût d'appel, compilation croisée |
| L'ordonnanceur | des milliers de goroutines sur quelques fils | rien : il faut le comprendre |
| Le ramasse-miettes | ne jamais libérer la mémoire soi-même | un peu de processeur et de mémoire |

Le code est dans [`examples/approfondissements/coulisses`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/coulisses).

## La réflexion

La réflexion permet à un programme d'examiner ses propres types et valeurs à
l'exécution. `encoding/json`, `database/sql`, `text/template` et les
validateurs l'utilisent : ils reçoivent un `any` et découvrent ce qu'il
contient.

L'exemple vérifie une structure d'après les étiquettes de ses champs, comme
le fait la bibliothèque go-playground/validator :

```go
type Tache struct {
	Titre      string   `valide:"requis,max=20"`
	Priorite   int      `valide:"min=1,max=5"`
	Etiquettes []string `valide:"max=3"`
	Adresse    Adresse
}
```

Le cœur parcourt les champs. Depuis Go 1.26, `reflect.Value.Fields` et
`reflect.Type.Fields` sont des **itérateurs** :

```go
func verifie(val reflect.Value, prefixe string, e *Erreur) {
	for champ, valeur := range val.Fields() {
		if !champ.IsExported() {
			continue // la réflexion ne lit pas les champs non exportés
		}
		nom := prefixe + champ.Name
		if valeur.Kind() == reflect.Struct { // structure imbriquée
			verifie(valeur, nom+".", e)
			continue
		}
		for regle := range strings.SplitSeq(champ.Tag.Get("valide"), ",") {
			if probleme := applique(strings.TrimSpace(regle), valeur); probleme != "" {
				e.Champs[nom] = probleme
				break
			}
		}
	}
}
```

Trois notions suffisent pour lire ce code :

- **`reflect.Type`** décrit un type (ses champs, ses méthodes, ses
  étiquettes) ; `reflect.TypeFor[Tache]()` (Go 1.22) l'obtient sans instance ;
- **`reflect.Value`** contient une valeur ; `Kind()` dit de quelle sorte elle
  est (`String`, `Int`, `Struct`, `Slice`…) ;
- une valeur n'est **modifiable** (`CanSet`) que si on l'a obtenue à travers
  un pointeur, et que le champ est exporté.

```console
validation : Adresse.Ville est obligatoire ; Etiquettes doit valoir au plus 3 éléments ;
             Priorite doit valoir au plus 5 ; Titre est obligatoire
```

### Ce que coûte la réflexion

Le même contrôle, écrit à la main, sert de point de comparaison :

```console
BenchmarkReflexion-16     845673     1254 ns/op    400 B/op   13 allocs/op
BenchmarkALaMain-16    145740312     8.34 ns/op      0 B/op    0 allocs/op
```

La réflexion est ici **150 fois plus lente**, et alloue. Elle perd aussi la
vérification du compilateur : une étiquette mal écrite (`min=abc`) n'est
découverte qu'à l'exécution. D'où la règle : la réflexion dans les
bibliothèques génériques, à la frontière du programme (décodage,
configuration) ; du code typé, ou des génériques, partout ailleurs. Quand le
type est connu à la compilation, `go generate` produit souvent un code plus
rapide et plus sûr (c'est ce que font sqlc et oapi-codegen).

## unsafe : voir la mémoire

Le package `unsafe` donne accès à ce que Go cache : l'adresse des valeurs,
leur taille, leur place en mémoire. Son nom est un avertissement : un
programme qui l'utilise renonce aux garanties du langage.

### La place des champs

Chaque champ d'une structure est **aligné** sur sa taille : un `float64`
commence à une adresse multiple de 8. Le compilateur insère donc des octets
de remplissage, et l'ordre des champs change la taille de la structure.

```go
type MalRangee struct {
	Actif   bool    // 1 octet, puis 7 de remplissage
	Montant float64 // 8 octets
	Code    bool    // 1 octet, puis 3 de remplissage
	Nombre  int32   // 4 octets
	Visible bool    // 1 octet, puis 7 de remplissage pour aligner la suivante
}

type BienRangee struct { // les mêmes champs, du plus grand au plus petit
	Montant float64
	Nombre  int32
	Actif   bool
	Code    bool
	Visible bool
}
```

`unsafe.Offsetof` et `unsafe.Sizeof` le montrent (sur une architecture
64 bits) :

```console
Actif    décalage  0, taille 1
Montant  décalage  8, taille 8
Code     décalage 16, taille 1
Nombre   décalage 20, taille 4
Visible  décalage 24, taille 1
MalRangee : 32 octets
BienRangee : 16 octets
```

Pour une tranche de 10 000 éléments, c'est 320 Kio au lieu de 160 Kio, et une
allocation deux fois plus rapide. L'analyseur `fieldalignment` (du module
`golang.org/x/tools`) signale les structures mal rangées. Cela ne vaut la
peine que pour les structures nombreuses en mémoire.

### Convertir sans copier

Convertir des `[]byte` en `string` copie les octets, parce qu'une chaîne
est immuable. `unsafe.String` (Go 1.20) crée une chaîne qui **partage** la
mémoire des octets :

```go
func Chaine(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
```

```console
BenchmarkConversionCopie-16        769.5 ns/op   4096 B/op   1 allocs/op
BenchmarkConversionSansCopie-16    1.148 ns/op      0 B/op   0 allocs/op
```

> [!CAUTION]
> Le test le montre : après la conversion, modifier les octets **modifie la
> chaîne**, pourtant « immuable ». Une chaîne utilisée comme clé de map, ou
> partagée entre goroutines, devient alors un piège. Dans l'autre sens,
> écrire dans les octets d'une chaîne peut faire planter le programme : sa
> mémoire peut être en lecture seule. Le compilateur évite déjà la copie dans
> les cas sûrs (`m[string(b)]`, une comparaison, une concaténation) ;
> `unsafe` ne se justifie qu'après une mesure.

## cgo : appeler du C

cgo relie Go et C : un programme Go appelle une bibliothèque C (SQLite,
OpenSSL, une bibliothèque système), et du code C peut rappeler du Go.

```go
// #include "ccalcul.h"
import "C"

func Somme(t []int64) int64 {
	if len(t) == 0 {
		return 0
	}
	return int64(C.somme((*C.int64_t)(unsafe.Pointer(&t[0])), C.size_t(len(t))))
}
```

Le code C est dans un fichier `.c` du même package, que `go build` compile
avec le compilateur C du système.

### Les règles des pointeurs

Le ramasse-miettes de Go déplace et libère la mémoire Go ; le code C n'en
sait rien. D'où des règles, vérifiées à l'exécution (`GODEBUG=cgocheck`) :

- passer à C un pointeur vers de la mémoire Go est permis **le temps de
  l'appel**, si cette mémoire ne contient pas elle-même de pointeurs Go
  (c'est le cas d'un `[]int64`) ;
- C ne doit pas **garder** un pointeur Go après l'appel ;
- une valeur Go (une fonction, une structure contenant des pointeurs) se
  transmet par une **poignée** : `cgo.Handle`, un simple entier que C rend à
  Go, qui retrouve la valeur.

```go
func Parcourt(t []int64, f func(int64)) {
	poignee := cgo.NewHandle(f)
	defer poignee.Delete() // sinon, la fonction ne serait jamais libérée
	C.parcourt((*C.int64_t)(unsafe.Pointer(&t[0])), C.size_t(len(t)), C.uintptr_t(poignee))
}

// Appelée par le code C, pour chaque élément.
//
//export appelleGo
func appelleGo(poignee C.uintptr_t, valeur C.int64_t) {
	f := cgo.Handle(poignee).Value().(func(int64))
	f(int64(valeur))
}
```

> [!NOTE]
> Un fichier qui exporte des fonctions (`//export`) ne peut que **déclarer**
> dans son préambule C (`#include`), pas définir de fonctions : elles vont
> dans un fichier `.c`. Le premier essai de cet exemple l'a rappelé, avec
> « could not determine what C.int64_t refers to » : il manquait
> `#include <stdint.h>` dans le fichier qui exporte.

### Ce que coûte un appel

```console
BenchmarkPetitC-16     32.04 ns/op     4 éléments, en C
BenchmarkPetitGo-16     3.60 ns/op     4 éléments, en Go
BenchmarkGrandC-16     30077 ns/op     100 000 éléments, en C
BenchmarkGrandGo-16    45923 ns/op     100 000 éléments, en Go
```

Franchir la frontière coûte une trentaine de nanosecondes : pour un petit
calcul, Go est neuf fois plus rapide. Sur un gros volume, le compilateur C
vectorise la boucle et l'emporte. cgo convient donc aux appels **rares et
substantiels**, pas aux appels nombreux et petits.

### Le prix caché : la construction

Avec cgo, le binaire dépend d'un compilateur C et souvent de la bibliothèque
C du système : la compilation croisée (`GOOS=windows`) se complique, et
l'image Docker minimale (`scratch`, `distroless/static`) ne suffit plus. C'est
pourquoi le dépôt utilise un pilote SQLite écrit en Go pur
(`modernc.org/sqlite`) plutôt que celui qui passe par cgo.

L'exemple montre la solution classique : un fichier `pur.go`, avec la
contrainte `//go:build !cgo`, fournit les mêmes fonctions en Go. Le package se
compile partout ; `CGO_ENABLED=0` choisit simplement la version Go. Les tests
passent dans les deux modes, et l'intégration continue, qui lance les tests
avec le détecteur de courses (lequel exige cgo), exerce la version C.

## L'ordonnanceur

Une goroutine coûte quelques kilo-octets ; un programme peut en lancer des
centaines de milliers. L'ordonnanceur de Go les répartit sur un petit nombre
de fils du système, selon le modèle **G-M-P** :

- **G** : une goroutine, avec sa pile qui grandit à la demande ;
- **M** : un fil (*thread*) du système d'exploitation ;
- **P** : un processeur logique, qui porte une file de goroutines prêtes.
  Il y en a `GOMAXPROCS`.

Un M n'exécute du Go que s'il tient un P. Quand une goroutine se bloque sur
un appel système, son M lâche le P, qu'un autre M reprend : les autres
goroutines continuent. Un P sans travail **vole** la moitié de la file d'un
autre.

### La préemption

Une goroutine qui calcule sans jamais appeler de fonction ne s'arrête jamais
d'elle-même. Avant Go 1.14, elle pouvait monopoliser son P. Depuis, le
runtime l'interrompt par un signal (**préemption asynchrone**) au bout de
10 ms environ.

L'exemple le prouve avec un seul P (`GOMAXPROCS=1`) et une boucle infinie :

```go
var compteur atomic.Uint64

func calcule() {
	for { // aucune fonction appelée, aucun point d'arrêt naturel
		compteur.Add(1)
	}
}

runtime.GOMAXPROCS(1)
go calcule()
time.Sleep(50 * time.Millisecond) // main cède le seul fil à calcule
fmt.Printf("main a repris après %v ; la boucle en était à %d tours\n", …)
```

```console
$ ordonnanceur -boucle
main a repris après 55ms ; la boucle en était à 12354068 tours

$ GODEBUG=asyncpreemptoff=1 ordonnanceur -boucle
                                  (bloqué : main ne reprend jamais la main)
```

Le test lance les deux versions : la première se termine, la seconde est
interrompue par le délai du test au bout de trois secondes.

### GOMAXPROCS dans un conteneur

`GOMAXPROCS` vaut par défaut le nombre de processeurs disponibles. Dans un
conteneur limité à deux processeurs sur une machine qui en a seize, l'ancien
comportement lançait seize fils pour deux processeurs de quota : le noyau
suspendait alors le programme à chaque période, ce qui dégradait les
latences. Depuis Go 1.25, le runtime lit la **limite de processeur des
cgroups** :

```console
$ ordonnanceur                                   # sur l'hôte
processeurs logiques (NumCPU) : 16
GOMAXPROCS                    : 16

$ docker run --cpus=2 … ordonnanceur
processeurs logiques (NumCPU) : 16
GOMAXPROCS                    : 2

$ docker run --cpus=1.5 … ordonnanceur
GOMAXPROCS                    : 2                 # arrondi au supérieur

$ docker run --cpus=2 -e GODEBUG=containermaxprocs=0 … ordonnanceur
GOMAXPROCS                    : 16                # l'ancien comportement
```

Le runtime suit aussi les changements de limite pendant l'exécution. Sur
Kubernetes, c'est la limite (`resources.limits.cpu`) qui compte. La
bibliothèque `go.uber.org/automaxprocs`, longtemps indispensable, ne l'est
plus.

### Observer

`go tool trace` montre l'ordonnanceur à l'œuvre : chaque P, chaque goroutine,
chaque blocage, sur une ligne de temps. Le chapitre « Profilage et
performance » explique comment produire une trace.

## Le ramasse-miettes

Go libère la mémoire inutilisée avec un ramasse-miettes (GC) **concurrent** :
il marque les objets accessibles pendant que le programme continue de
tourner, avec des pauses de l'ordre de la dizaine de microsecondes. Depuis
Go 1.26, sa version **Green Tea** est activée par défaut : elle parcourt la
mémoire par blocs plutôt qu'objet par objet, ce qui profite aux processeurs
modernes. `GOEXPERIMENT=nogreenteagc` revient à l'ancien.

### GOGC et GOMEMLIMIT

Deux réglages pilotent son rythme :

- **`GOGC`** (100 par défaut) : un cycle démarre quand le tas a grandi de
  GOGC % depuis le précédent. Plus grand, le GC passe moins souvent, et la
  mémoire grossit ;
- **`GOMEMLIMIT`** (Go 1.19) : une limite douce de la mémoire totale. Le GC
  accélère à son approche.

L'exemple mesure, avec `runtime/metrics`, le nombre de cycles pour un même
travail (400 Mio alloués, 32 Mio gardés en vie) :

```go
func Observe(gogc int, limite int64, travail func()) Mesure {
	ancienGOGC := debug.SetGCPercent(gogc)
	defer debug.SetGCPercent(ancienGOGC)
	…
	echantillons := []metrics.Sample{
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/gc/heap/goal:bytes"},
	}
	metrics.Read(echantillons)
	…
}
```

```console
GOGC=50                       29 cycles, objectif du tas   49 Mio
GOGC=100                      15 cycles, objectif du tas   65 Mio
GOGC=400                       4 cycles, objectif du tas  164 Mio
GOGC=off, GOMEMLIMIT=64Mio    20 cycles, objectif du tas   54 Mio
```

C'est un compromis entre processeur et mémoire. La dernière ligne montre la
configuration la plus utile pour un conteneur : **désactiver le GC
proportionnel, et fixer une limite** un peu sous la mémoire du conteneur. Le
GC ne travaille que lorsque c'est nécessaire, et la mémoire reste bornée.

> [!WARNING]
> `GOMEMLIMIT` est une limite **douce** : si le programme garde réellement
> plus de mémoire vivante que la limite, le GC tourne sans cesse
> (*thrashing*) sans pouvoir l'atteindre. Le runtime borne alors le temps
> qu'il y consacre, et la mémoire dépasse la limite. Elle protège d'un pic,
> pas d'une fuite.

### Les pointeurs faibles et les nettoyages

Un cache retient ses valeurs : tant qu'elles y sont, le GC ne peut pas les
libérer. Un **pointeur faible** (package `weak`, Go 1.24) désigne une valeur
sans la retenir. `runtime.AddCleanup` (Go 1.24) exécute une fonction quand une
valeur est libérée ; il remplace avantageusement `runtime.SetFinalizer`.

```go
func (c *Cache[K, V]) Met(k K, v *V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entrees[k] = weak.Make(v)
	// Quand v sera libérée, le runtime appellera ce nettoyage, dans une autre
	// goroutine. L'argument est la clé, jamais v : un nettoyage qui
	// retiendrait v l'empêcherait d'être libérée.
	runtime.AddCleanup(v, c.oublie, k)
}

func (c *Cache[K, V]) Prend(k K) *V {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entrees[k].Value() // nil si absente ou libérée
}
```

Le test range deux images, n'en garde qu'une, et provoque le GC : l'image
oubliée disparaît du cache, l'autre y reste.

> [!NOTE]
> Les tests qui dépendent du GC sont délicats : le moment d'un cycle ou d'un
> nettoyage n'est pas garanti. Le test du cache appelle `runtime.GC()` et
> attend le nettoyage avec un délai maximal. Il a été lancé dix fois de suite
> sans échec. Le test de GOGC ne vérifie que des **relations** (moins de
> cycles quand GOGC augmente), jamais des nombres exacts.

## En résumé

- **La réflexion** traite un type inconnu à la compilation, à la frontière du
  programme ; elle est ici 150 fois plus lente qu'un code typé.
- **`unsafe`** montre la mémoire : un bon rangement des champs divise la
  taille par deux ; une conversion sans copie est rapide, et dangereuse.
- **cgo** coûte une trentaine de nanosecondes par appel, et complique la
  construction : une version en Go pur (`//go:build !cgo`) garde le package
  portable.
- **L'ordonnanceur** répartit les goroutines sur `GOMAXPROCS` fils, préempte
  les boucles infinies, et respecte, depuis Go 1.25, la limite de processeur
  d'un conteneur.
- **Le ramasse-miettes** se règle par `GOGC` et `GOMEMLIMIT` ; les pointeurs
  faibles et `AddCleanup` permettent des caches qui ne fuient pas.
