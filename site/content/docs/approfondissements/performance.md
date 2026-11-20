---
title: "Profilage et performance"
linkTitle: "10. Profilage et performance"
weight: 100
---

Go est rapide par défaut, mais un programme lent l'est presque toujours pour
une raison précise, souvent inattendue. Ce chapitre présente la démarche et les
outils pour la trouver : benchmarks, profils, analyse d'échappement, traces et
optimisation guidée par profil. Il les applique à un exemple réel, dont tous
les chiffres ont été mesurés.

Le code est dans [`examples/approfondissements/performance`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/performance).

## La démarche

> « Premature optimization is the root of all evil. » — Donald Knuth

1. **Mesurer** : écrire un benchmark qui reproduit le cas lent.
2. **Profiler** : trouver où le temps et la mémoire sont réellement dépensés.
3. **Corriger** le point chaud identifié, et lui seul.
4. **Vérifier** : comparer les mesures, et s'assurer que le résultat est
   toujours correct.

Optimiser sans mesurer, c'est optimiser au hasard : le code devient plus
compliqué, rarement plus rapide.

## L'exemple : compter les mots

La fonction à optimiser renvoie les mots les plus fréquents d'un texte. La
première version est simple et correcte :

```go
var separateurs = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func MotsFrequentsNaif(r io.Reader, n int) ([]Frequence, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	compte := map[string]int{}
	for _, mot := range separateurs.Split(string(data), -1) {
		if mot != "" {
			compte[strings.ToLower(mot)]++
		}
	}
	return lesPlusFrequents(compte, n), nil
}
```

Avant toute optimisation, un test garantit que la version optimisée
produira **exactement** les mêmes résultats :

```go
func TestVersionsIdentiques(t *testing.T) {
	naif, err1 := MotsFrequentsNaif(strings.NewReader(texte), 5)
	optimise, err2 := MotsFrequents(strings.NewReader(texte), 5)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if !slices.Equal(naif, optimise) {
		t.Errorf("résultats différents :\n%v\n%v", naif, optimise)
	}
}
```

## Mesurer : les benchmarks

```go
func BenchmarkMotsFrequentsNaif(b *testing.B) {
	for b.Loop() {
		MotsFrequentsNaif(strings.NewReader(texte), 10)
	}
}
```

```bash
$ go test -run '^$' -bench . -benchmem
BenchmarkMotsFrequentsNaif-16    	     91	  12037140 ns/op	 3922069 B/op	   28056 allocs/op
```

Sur un texte de 22 000 mots, la fonction prend **12 ms** et alloue près de
**4 Mo** en 28 056 allocations. Chaque colonne compte :

| Colonne      | Signification                                         |
|--------------|-------------------------------------------------------|
| `91`         | nombre d'itérations exécutées                         |
| `ns/op`      | temps par appel                                       |
| `B/op`       | mémoire allouée par appel (avec `-benchmem`)          |
| `allocs/op`  | nombre d'allocations par appel                        |

Les **allocations** sont souvent la première piste : chacune coûte du temps à
l'allocation, puis au ramasse-miettes.

> [!TIP]
> Un benchmark varie d'une exécution à l'autre. Pour comparer deux versions de
> manière fiable, exécutez chacune une dizaine de fois et comparez avec
> [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) :
>
> ```bash
> go test -bench . -count 10 > avant.txt
> # … modification …
> go test -bench . -count 10 > apres.txt
> go run golang.org/x/perf/cmd/benchstat@latest avant.txt apres.txt
> ```

## Profiler : pprof

### Le profil CPU

Le profil CPU échantillonne la pile d'appels 100 fois par seconde, et indique
où le programme passe son temps :

```bash
$ go test -run '^$' -bench MotsFrequentsNaif -cpuprofile cpu.pprof
$ go tool pprof -top cpu.pprof
      flat  flat%   sum%        cum   cum%
     360ms 13.33% 13.33%      360ms 13.33%  regexp/syntax.(*Inst).MatchRunePos
     340ms 12.59% 25.93%      450ms 16.67%  regexp.(*machine).add
     260ms  9.63% 35.56%     1520ms 56.30%  regexp.(*machine).match
     250ms  9.26% 44.81%      820ms 30.37%  regexp.(*machine).step
     120ms  4.44% 49.26%      130ms  4.81%  runtime.tryDeferToSpanScan
     …
```

- **`flat`** : le temps passé dans la fonction elle-même ;
- **`cum`** : le temps cumulé, en comptant les fonctions qu'elle appelle.

Le diagnostic est immédiat : **56 %** du temps est dans le moteur
d'expressions régulières. Pour découper un texte sur des caractères
non alphanumériques, c'est un outil bien trop puissant.

`go tool pprof -http=:8080 cpu.pprof` ouvre une interface web, avec un
*flame graph* (la vue par défaut depuis Go 1.26), le graphe des appels et le
code source annoté ligne par ligne.

### Le profil mémoire

```bash
$ go test -run '^$' -bench MotsFrequentsNaif -memprofile mem.pprof
$ go tool pprof -sample_index=alloc_space -top mem.pprof
      flat  flat%   sum%        cum   cum%
  437.22MB 58.09% 58.09%   437.22MB 58.09%  slices.AppendSeq[…]
  154.82MB 20.57% 78.66%   655.54MB 87.10%  regexp.(*Regexp).Split
   61.50MB  8.17% 86.83%   498.72MB 66.26%  regexp.(*Regexp).allStringIndex.func1-range1
   53.67MB  7.13% 93.96%    53.67MB  7.13%  io.ReadAll
```

Même conclusion : `regexp.Split` est responsable de **87 %** des allocations.
`-sample_index=alloc_space` montre tout ce qui a été alloué ;
`inuse_space` montre ce qui est encore en mémoire, pour chercher une fuite.

## Corriger, étape par étape

### Première version : sans expression régulière

On remplace l'expression régulière par un découpage en mots
(`bufio.ScanWords`), puis par `strings.FieldsFuncSeq` (Go 1.24) avec une
simple fonction de test. Le texte est lu en flux, sans tout charger en
mémoire :

```
BenchmarkMotsFrequents-16        	    516	   2085843 ns/op	  250928 B/op	   28008 allocs/op
```

Le temps est divisé par **6** et la mémoire par **15**… mais le nombre
d'allocations n'a presque pas changé. `-benchmem` révèle ce que le profil CPU
ne montrait pas : `sc.Text()` crée une nouvelle chaîne **pour chaque mot**.

### Seconde version : travailler sur des octets

On travaille directement sur les octets du `Scanner`, avec un tampon
réutilisé pour la mise en minuscules. Et on exploite une optimisation du
compilateur : une **lecture** de map avec `m[string(octets)]` n'alloue pas de
chaîne.

```go
func MotsFrequents(r io.Reader, n int) ([]Frequence, error) {
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)
	compte := make(map[string]*int, 1024) // capacité initiale : moins de réallocations
	var minuscules []byte                  // tampon réutilisé pour chaque mot
	for sc.Scan() {
		for mot := range bytes.FieldsFuncSeq(sc.Bytes(), separateur) {
			minuscules = minuscules[:0]
			for _, r := range string(mot) { // conversion sans allocation
				minuscules = utf8.AppendRune(minuscules, unicode.ToLower(r))
			}
			// m[string(b)] en lecture n'alloue pas : le compilateur évite la copie.
			if c, ok := compte[string(minuscules)]; ok {
				*c++
				continue
			}
			un := 1
			compte[string(minuscules)] = &un // nouveau mot : une seule allocation
		}
	}
	if err := sc.Err(); err != nil { // erreur de lecture, ou mot trop long
		return nil, err
	}
	…
}
```

```
BenchmarkMotsFrequentsNaif-16    	    100	  11505007 ns/op	 3922240 B/op	   28056 allocs/op
BenchmarkMotsFrequents-16        	    673	   1614901 ns/op	   59064 B/op	      25 allocs/op
```

Le bilan est sans appel :

| Version                     | Temps    | Mémoire  | Allocations |
|-----------------------------|----------|----------|-------------|
| expression régulière        | 11,5 ms  | 3,9 Mo   | 28 056      |
| découpage simple            | 2,1 ms   | 245 Ko   | 28 008      |
| octets et tampon réutilisé  | 1,6 ms   | 58 Ko    | **25**      |

La fonction est **7 fois plus rapide**, utilise **66 fois moins de mémoire**,
et n'alloue plus qu'une chaîne par mot **distinct**, et non par occurrence.

Cette dernière étape a rendu le code plus complexe. C'est le compromis
habituel : on ne la fait que si la fonction est réellement sur un chemin
critique.

### L'analyse d'échappement

L'option `-gcflags=-m` montre les décisions du compilateur. Elle confirme que
la lecture de map n'alloue pas, contrairement à l'insertion d'un nouveau mot :

```bash
$ go build -gcflags=-m . 2>&1 | grep mots.go
./mots.go:56:30: string(minuscules) does not escape
./mots.go:61:18: string(minuscules) escapes to heap
./mots.go:60:4: moved to heap: un
```

### Verrouiller le gain

Une optimisation peut être perdue par une modification anodine. Un test fige
le nombre d'allocations : une régression de performance devient alors un
échec de test, détecté par l'intégration continue.

```go
func TestAllocations(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() { Concatene(parts) }); n != 1 {
		t.Errorf("Concatene : %.0f allocations, attendu 1", n)
	}
	// Une allocation par mot distinct, pas par occurrence.
	if n := testing.AllocsPerRun(10, func() { MotsFrequents(strings.NewReader(texte), 10) }); n > 50 {
		t.Errorf("MotsFrequents : %.0f allocations, attendu 50 au plus", n)
	}
}
```

## Les optimisations les plus rentables

Dans la grande majorité des programmes, quelques règles simples suffisent :

- **Préallouer** : `make([]T, 0, n)`, `make(map[K]V, n)`, `strings.Builder.Grow`.
  Construire une chaîne de 1 000 morceaux par concaténation coûte 1 000
  allocations ; avec `Grow`, une seule :

  ```
  BenchmarkConcateneNaif-16        	   2059	    549342 ns/op	 3085152 B/op	    1000 allocs/op
  BenchmarkConcatene-16            	 212736	      9130 ns/op	    6144 B/op	       1 allocs/op
  ```

- **Travailler en flux** (`io.Reader`, `bufio`) plutôt que de tout charger en
  mémoire.
- **Éviter les conversions répétées** entre `string` et `[]byte`.
- **Réutiliser** les objets temporaires coûteux avec `sync.Pool`.
- **Choisir le bon outil** : `strings` plutôt que `regexp`, `strconv` plutôt
  que `fmt.Sprintf` dans une boucle critique.
- **Réduire les entrées-sorties** : regrouper les requêtes à la base de
  données, mettre en cache, compresser. C'est souvent là que se cache la
  vraie lenteur d'un service, bien plus que dans le CPU.

## Profiler un service en production

Les problèmes de performance se manifestent souvent en production, avec des
données réelles. L'import du package `net/http/pprof` expose les profils en
HTTP :

```go
import _ "net/http/pprof" // enregistre /debug/pprof/ sur http.DefaultServeMux

go func() {
	// Sur un port d'administration, jamais exposé publiquement.
	log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30   # CPU
go tool pprof http://localhost:6060/debug/pprof/heap                  # mémoire
go tool pprof http://localhost:6060/debug/pprof/goroutine             # goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutineleak         # fuites (Go 1.27)
go tool pprof http://localhost:6060/debug/pprof/mutex                 # contention
```

Le coût du profilage est faible : il est courant de le laisser activé en
production. Les profils `mutex` et `block` doivent être activés explicitement
(`runtime.SetMutexProfileFraction`, `runtime.SetBlockProfileRate`).

## Les traces d'exécution

Un profil répond à la question « où passe le temps ? ». Une **trace**
d'exécution répond à « que se passe-t-il, et quand ? » : quelles goroutines
s'exécutent, sur quels processeurs, pourquoi elles attendent (réseau,
verrou, channel, GC). C'est l'outil des problèmes de **latence** et de
**concurrence**.

```bash
go test -trace trace.out
go tool trace trace.out
```

Les tâches et régions annotent la trace avec la sémantique du programme :

```go
ctx, tache := trace.NewTask(ctx, "commande")
defer tache.End()
trace.WithRegion(ctx, "paiement", func() { paie(ctx, commande) })
```

### Le flight recorder

> [!NOTE]
> **Go 1.25** — le *flight recorder* garde en mémoire les dernières secondes
> de trace, et ne les écrit qu'à la demande.

Comme la boîte noire d'un avion, il permet de capturer ce qui s'est passé
**juste avant** un incident, par exemple une requête anormalement lente, sans
tracer en permanence :

```go
fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
	MinAge:   5 * time.Second, // garder au moins les 5 dernières secondes
	MaxBytes: 4 << 20,         // dans 4 Mo au plus
})
fr.Start()
defer fr.Stop()

// Dans un middleware HTTP :
debut := time.Now()
suivant.ServeHTTP(w, r)
if time.Since(debut) > 500*time.Millisecond {
	f, _ := os.Create(fmt.Sprintf("lente-%d.trace", time.Now().Unix()))
	fr.WriteTo(f) // les secondes qui ont précédé la requête lente
	f.Close()
}
```

## L'optimisation guidée par profil (PGO)

Depuis **Go 1.21**, le compilateur peut utiliser un profil CPU de production
pour mieux optimiser le programme : il intègre (*inline*) plus agressivement
les fonctions les plus appelées, et dévirtualise les appels d'interfaces
fréquents.

La mise en œuvre tient en deux étapes :

1. récupérer un profil CPU représentatif en production :

   ```bash
   curl -o default.pgo "http://localhost:6060/debug/pprof/profile?seconds=30"
   ```

2. le placer dans le répertoire du package `main`, sous le nom `default.pgo`,
   et le versionner. `go build` l'utilise automatiquement (`-pgo=auto` est la
   valeur par défaut).

Les gains annoncés vont de **2 à 14 %** de CPU, sans modifier une ligne de
code. Le profil n'a pas besoin d'être parfaitement à jour : un profil de la
version précédente reste efficace.

## Le ramasse-miettes

Lorsque le profil montre beaucoup de temps dans le runtime (`runtime.mallocgc`,
`runtime.scanObject`, `runtime.gcBgMarkWorker`), le ramasse-miettes travaille
trop. La première réponse est de **réduire les allocations**, comme dans
l'exemple. Ensuite, deux réglages existent :

| Réglage                 | Effet                                                       |
|-------------------------|-------------------------------------------------------------|
| `GOGC=200`              | GC moins fréquent, en échange de plus de mémoire            |
| `GOMEMLIMIT=900MiB`     | limite souple : le GC accélère à l'approche de la limite    |
| `GODEBUG=gctrace=1`     | affiche une ligne par cycle de GC, pour comprendre          |

Dans un conteneur, la combinaison `GOGC=off` et `GOMEMLIMIT` proche de la
limite du conteneur est une configuration courante pour les services à forte
charge : le GC ne se déclenche que lorsque la mémoire l'exige.

## En résumé

| Question                          | Outil                                         |
|-----------------------------------|-----------------------------------------------|
| Est-ce plus rapide ?              | `go test -bench -benchmem`, `benchstat`       |
| Où passe le temps ?               | profil CPU : `-cpuprofile`, `/debug/pprof/profile` |
| Où va la mémoire ?                | profil mémoire : `-memprofile`, `/debug/pprof/heap` |
| Pourquoi cette allocation ?       | `go build -gcflags=-m`                        |
| Pourquoi cette latence ?          | trace d'exécution, flight recorder            |
| Des goroutines s'accumulent ?     | profils `goroutine` et `goroutineleak`        |
| Gagner quelques pour cent de plus | PGO avec `default.pgo`                        |
| Ne pas régresser                  | `testing.AllocsPerRun` dans les tests         |
