---
title: "Aide-mémoire"
linkTitle: "1. Aide-mémoire"
weight: 10
---

L'essentiel de Go en quelques pages : la syntaxe, les idiomes, les pièges et
les commandes. Il existe aussi en
[PDF séparé](https://github.com/laurentpoirierfr/golang-demystified/releases/latest/download/aide-memoire-go.pdf),
en deux colonnes, à imprimer. Les sorties citées sont vérifiées par les tests de
[`examples/annexes/aidememoire`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/annexes/aidememoire).

## Les commandes go

| Commande | Rôle |
|----------|------|
| `go mod init exemple.fr/app` | créer un module |
| `go run .` | compiler et exécuter |
| `go build -o app ./cmd/app` | compiler un binaire |
| `go test ./...` | tester tous les packages |
| `go test -race -cover ./...` | avec détecteur de courses et couverture |
| `go test -run TestX -v` | un seul test, en détail |
| `go test -bench . -benchmem` | les benchmarks |
| `go test -fuzz FuzzX` | le fuzzing |
| `go vet ./...` | l'analyse statique |
| `gofmt -l .` ou `go fmt ./...` | le formatage |
| `go fix ./...` | moderniser le code |
| `go get exemple.fr/lib@v1.2.3` | ajouter ou changer une dépendance |
| `go mod tidy` | ranger `go.mod` et `go.sum` |
| `go install outil@latest` | installer un outil |
| `go get -tool outil` puis `go tool outil` | un outil versionné dans `go.mod` |
| `go doc strings.Cut` | la documentation |
| `go env GOPATH` | une variable d'environnement |
| `GOOS=linux GOARCH=arm64 go build` | compiler pour une autre plateforme |
| `go build -ldflags "-s -w"` | un binaire sans symboles |
| `go tool pprof cpu.out` | analyser un profil |

## Variables et types

```go
var n int            // valeur zéro : 0
var s = "texte"      // type déduit
x, y := 1, 2.5       // dans une fonction
const Pi = 3.14159   // constante non typée

const (
	Lundi = iota     // 0
	Mardi            // 1
)
```

| Type | Valeur zéro | Remarque |
|------|-------------|----------|
| `bool` | `false` | |
| `int`, `int8`…`int64` | `0` | `int` : 64 bits sur les machines 64 bits |
| `uint`, `uint8` (`byte`)… | `0` | dépasse en « faisant le tour » |
| `float64`, `float32` | `0` | |
| `string` | `""` | immuable, en octets UTF-8 |
| `rune` (`int32`) | `0` | un point de code Unicode |
| pointeur, slice, map, channel, fonction, interface | `nil` | |
| `struct` | chaque champ à sa valeur zéro | |

Les conversions sont toujours explicites : `float64(n)`, `int(f)` (tronque :
`int(3.99)` vaut 3), `string(r)` pour une rune, `strconv.Itoa(n)` pour écrire
un nombre. La division entière tronque vers zéro : `-7/2` vaut -3, `-7%3`
vaut -1.

## Contrôle de flux

```go
if v, err := f(); err != nil {
	return err
}                    // v et err n'existent que dans le if

for i := 0; i < n; i++ { }
for cond { }         // un « while »
for { }              // boucle infinie
for i := range 10 { }        // 0 à 9
for i, v := range slice { }
for k, v := range m { }      // ordre aléatoire
for i, r := range "héllo" { } // runes, i en octets
for v := range ch { }        // jusqu'à close(ch)

switch {             // sans expression : if/else
case x < 0:
	…
case x == 0, x == 1: // plusieurs valeurs
	…
default:
}                    // pas de fallthrough implicite

defer f.Close()      // à la sortie de la fonction,
                     // en ordre inverse
```

Chaque itération d'une boucle `for` a ses propres variables (depuis Go
1.22) : une fermeture capture la valeur de son tour.

## Slices et maps

```go
s := []int{1, 2, 3}
s = make([]int, 0, 10) // longueur 0, capacité 10
s = append(s, 4, 5)
t := s[1:3]          // partage le tableau de s
u := s[1:3:3]        // capacité limitée : append copiera
copy(dst, src)
clear(s)             // met tout à zéro

m := map[string]int{"a": 1}
m = make(map[string]int)
v, ok := m["clé"]    // ok faux si absente
delete(m, "clé")
```

| Package | Fonctions utiles |
|---------|------------------|
| `slices` | `Sort`, `SortFunc`, `Contains`, `Index`, `Equal`, `Reverse`, `Max`, `BinarySearch`, `Collect`, `Sorted`, `Values` |
| `maps` | `Keys`, `Values`, `Clone`, `Copy`, `Equal` |
| intégrées | `len`, `cap`, `min`, `max`, `clear` |

## Fonctions

```go
func div(a, b int) (int, error) { … }
func somme(n ...int) int { … }  // somme(1, 2), somme(s...)
func f() (n int, err error) { … } // résultats nommés

carre := func(x int) int { return x * x }
compteur := func() func() int {
	n := 0
	return func() int { n++; return n }
}()                              // fermeture
```

## Structures, méthodes, interfaces

```go
type Point struct{ X, Y int }

p := Point{X: 1, Y: 2}
q := &Point{1, 2}    // pointeur ; q.X marche

func (p Point) Norme() int     // récepteur valeur
func (p *Point) Deplace(dx int) // pointeur :
                                // modifie p

type Nomme struct {
	Point            // intégration : Point.X,
	Nom string       // Norme… sont promus
}

type Forme interface {
	Aire() float64   // satisfaite implicitement
}

var _ Forme = Carre{} // vérifié à la compilation

f, ok := v.(Carre)   // assertion de type
switch x := v.(type) {
case Carre: …
case nil: …
}
```

Une interface qui contient un pointeur nil **n'est pas** nil : elle porte un
type. Renvoyez `nil` directement, pas un `*MonErreur` nil.

## Erreurs

```go
var ErrAbsent = errors.New("absent")

return fmt.Errorf("lire %s : %w", nom, err)

if errors.Is(err, fs.ErrNotExist) { }
var pe *fs.PathError
if errors.As(err, &pe) { }
if e, ok := errors.AsType[*fs.PathError](err); ok { }
err = errors.Join(err1, err2) // nil si tous nil

defer func() {
	if r := recover(); r != nil { … }
}()
```

`panic` est réservé aux erreurs de programmation ; une erreur attendue est
une valeur `error`, renvoyée en dernier résultat.

## Génériques

```go
func Max[T cmp.Ordered](s []T) T { … }

type Pile[T any] struct{ elements []T }
func (p *Pile[T]) Empile(v T) { … }

type Nombre interface {
	~int | ~int64 | ~float64 // ~ : types dérivés
}
```

Contraintes courantes : `any`, `comparable` (`==`, clé de map),
`cmp.Ordered` (`<`). Une méthode ne peut pas avoir ses propres paramètres de
type.

## Concurrence

```go
go f(x)              // une goroutine

var wg sync.WaitGroup
wg.Go(func() { … })  // Add, go, Done en un
wg.Wait()

ch := make(chan int)    // sans tampon
ch = make(chan int, 10) // avec tampon
ch <- v; v = <-ch; v, ok = <-ch
close(ch)            // par l'émetteur

select {
case v := <-ch: …
case <-ctx.Done(): return ctx.Err()
case <-time.After(time.Second): …
default:             // ne bloque pas
}

var mu sync.Mutex
mu.Lock(); defer mu.Unlock()
var n atomic.Int64; n.Add(1)
```

| Opération | channel nil | ouvert | fermé |
|-----------|-------------|--------|-------|
| envoi | bloque toujours | bloque sans place ni receveur | **panique** |
| réception | bloque toujours | bloque sans valeur | immédiate : valeur zéro, `ok` faux |
| `close` | **panique** | ferme | **panique** |

```go
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()       // toujours
```

Le contexte est le premier paramètre (`ctx context.Context`), jamais rangé
dans une structure. `go test -race` trouve les accès concurrents.

## fmt : les verbes

| Verbe | Exemple | Résultat |
|-------|---------|----------|
| `%v`, `%+v`, `%#v` | `Point{1, 2}` | `{1 2}`, `{X:1 Y:2}`, `main.Point{X:1, Y:2}` |
| `%T` | `Point{}` | `main.Point` |
| `%d`, `%5d`, `%-5d`, `%05d` | `42` | `42`, `␣␣␣42`, `42␣␣␣`, `00042` |
| `%x`, `%X`, `%o`, `%b` | `255`, `255`, `8`, `5` | `ff`, `FF`, `10`, `101` |
| `%c`, `%U` | `'G'` | `G`, `U+0047` |
| `%f`, `%.2f`, `%8.3f` | `3.14159` | `3.141590`, `3.14`, `␣␣␣3.142` |
| `%e`, `%g` | `1234.5678`, `0.000012` | `1.234568e+03`, `1.2e-05` |
| `%s`, `%q`, `%x` | `"go"` | `go`, `"go"`, `676f` |
| `%10s`, `%-10s` | `"go"` | `␣␣␣␣␣␣␣␣go`, `go␣␣␣␣␣␣␣␣` |
| `%t`, `%%` | `true` | `true`, `%` |

Dans ce tableau, `␣` représente une espace.

`Sprintf` renvoie le texte, `Fprintf` écrit dans un `io.Writer`, `Errorf`
crée une erreur. Un type qui a une méthode `String() string` choisit son
affichage.

## time : formats et durées

La date de référence est **Mon Jan 2 15:04:05 MST 2006** : 1 2 3 4 5 6.

| Format | Résultat |
|--------|----------|
| `"2006-01-02 15:04:05"` | `2026-11-20 14:05:09` |
| `"02/01/2006 à 15h04"` | `20/11/2026 à 14h05` |
| `time.RFC3339` | `2026-11-20T14:05:09Z` |
| `time.DateOnly`, `time.Kitchen` | `2026-11-20`, `2:05PM` |

```go
d := 90*time.Minute + 30*time.Second // 1h30m30s
time.Since(debut); t.Add(d); t.Sub(u)
t.AddDate(0, 1, 0)   // un mois plus tard
time.ParseInLocation(format, s, loc)
time.ParseDuration("1h30m")
```

## Tests

```go
func TestX(t *testing.T) {
	tests := []struct{ nom, in, want string }{…}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			if got := F(tt.in); got != tt.want {
				t.Errorf("F(%q) = %q, veut %q",
					tt.in, got, tt.want)
			}
		})
	}
}

func BenchmarkX(b *testing.B) {
	for b.Loop() { F("x") }
}

func FuzzX(f *testing.F) {
	f.Add("graine")
	f.Fuzz(func(t *testing.T, s string) { … })
}

func ExampleF() {
	fmt.Println(F("x"))
	// Output: X
}
```

`t.Helper()`, `t.Cleanup(f)`, `t.TempDir()`, `t.Context()`,
`t.Parallel()` ; `testing/synctest` pour le temps et la concurrence ;
`net/http/httptest` pour HTTP.

## Pièges courants

| Piège | Ce qui se passe |
|-------|-----------------|
| écrire dans une map nil | panique : `make` d'abord |
| `append` sur une sous-slice | peut écraser le tableau d'origine |
| `len("héllo")` | 6 : des octets, pas des caractères |
| `s[1]` sur une chaîne | un octet, pas une rune |
| interface contenant un pointeur nil | `err != nil` est vrai |
| `defer` dans une boucle | exécuté à la fin de la fonction, pas du tour |
| goroutine qui attend un channel abandonné | fuite : elle ne s'arrête jamais |
| oublier `cancel()` | le contexte et son timer restent en mémoire |
| copier un `sync.Mutex` | `go vet` le signale : passez un pointeur |
| comparer des flottants avec `==` | utilisez une tolérance |

## Organisation d'un projet

```text
monprojet/
├── go.mod
├── cmd/app/main.go   # les programmes
├── internal/         # importable seulement ici
└── pkg/              # réutilisable par d'autres
```

Un nom en majuscule est exporté (`Client`), en minuscule il est privé au
package (`client`). Un répertoire, un package ; le package `main` donne un
programme. Les tests sont dans `x_test.go`, à côté du code.
