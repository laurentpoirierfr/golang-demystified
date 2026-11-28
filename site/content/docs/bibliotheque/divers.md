---
title: "III.10 Tour d'horizon"
weight: 100
---

# Chapitre III.10 — Tour d'horizon

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/divers`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/divers).

La bibliothèque standard de Go est vaste. Ce chapitre présente des packages
utiles au quotidien, qui n'ont pas eu de chapitre dédié.

## uuid

> [!NOTE]
> **Go 1.27** — le package `uuid` génère et analyse des identifiants
> uniques universels (RFC 9562). Il n'est plus nécessaire d'ajouter une
> dépendance externe.

```go
import "uuid"

id := uuid.New()    // version 4 : aléatoire
id7 := uuid.NewV7() // version 7 : commence par l'horodatage, donc triable

u, err := uuid.Parse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
```

Les UUID v7 sont recommandés comme clés primaires de base de données : comme
ils sont croissants, ils s'insèrent efficacement dans les index. Le type
`uuid.UUID` implémente `MarshalText` et `UnmarshalText`, et s'encode donc
directement en JSON.

## Le hasard

| Package          | Usage                                                         |
|------------------|---------------------------------------------------------------|
| `crypto/rand`    | Hasard **cryptographique** : jetons, clés, mots de passe      |
| `math/rand/v2`   | Hasard **statistique** : simulations, jeux, tirages, jitter   |

```go
jeton := rand.Text() // crypto/rand, Go 1.24 : 26 caractères base32

de := randv2.IntN(6) + 1                 // math/rand/v2, Go 1.22
attente := randv2.N(10 * time.Second)    // générique : une durée aléatoire
r := randv2.New(randv2.NewPCG(1, 2))     // générateur déterministe, pour les tests
```

Le package `math/rand/v2` (**Go 1.22**) remplace `math/rand` : il n'a plus
besoin d'être initialisé avec une graine, propose de meilleurs algorithmes
(PCG, ChaCha8) et des fonctions génériques comme `rand.N`.

> [!CAUTION]
> N'utilisez **jamais** `math/rand` pour générer un secret, un jeton de
> session ou un mot de passe : ses valeurs sont prévisibles.

## Cryptographie et encodages

```go
somme := sha256.Sum256([]byte("gopher"))
fmt.Println(hex.EncodeToString(somme[:8]))
// 9cc1ee455a3363ff
```

Les packages `crypto/...` couvrent les fonctions de hachage (`sha256`,
`sha3`), le chiffrement (`aes`, `cipher`), les signatures (`ed25519`, `ecdsa`,
`mldsa` depuis Go 1.27), l'échange de clés (`ecdh`, `mlkem`), TLS et les
certificats X.509. Ils sont maintenus par l'équipe Go et font l'objet d'audits
de sécurité réguliers. Depuis Go 1.24, ils peuvent fonctionner dans un mode
conforme FIPS 140-3.

Pour stocker des mots de passe, utilisez une fonction lente dédiée, comme
[`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) ou [`argon2`](https://pkg.go.dev/golang.org/x/crypto/argon2), et non une simple empreinte SHA-256.

Les encodages sont regroupés dans `encoding/...` : `hex`, `base64`, `csv`,
`xml`, `binary`, `pem`…

## CSV

```go
r := csv.NewReader(strings.NewReader("nom,age\nAlice,31\n\"Bob, junior\",27\n"))
lignes, err := r.ReadAll() // [][]string

w := csv.NewWriter(os.Stdout)
w.Write([]string{"Saint-Étienne, Loire", "170000"})
w.Flush()
// "Saint-Étienne, Loire",170000
```

Le package gère les guillemets et les séparateurs dans les valeurs. Pour un
gros fichier, lisez les lignes une à une avec `r.Read()`.

## Expressions régulières : regexp

```go
email := regexp.MustCompile(`^([a-z0-9._%+-]+)@([a-z0-9.-]+\.[a-z]{2,})$`)
email.MatchString("gopher@go.dev")              // true
email.FindStringSubmatch("alice@example.com")   // [alice@example.com alice example.com]

date := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
date.ReplaceAllString("sortie le 2026-08-12", "$3/$2/$1")
// sortie le 12/08/2026
```

Le moteur utilise la syntaxe **RE2** : il garantit un temps d'exécution
linéaire en fonction de la taille de l'entrée, ce qui protège contre les
attaques par expressions régulières (ReDoS). En contrepartie, il ne gère pas
les références arrière (`\1`).

Compilez les expressions **une seule fois**, dans une variable de package, et
non à chaque appel. Pour une recherche simple, les fonctions de `strings` sont
plus rapides et plus lisibles.

## La ligne de commande : flag

```go
port := flag.Int("port", 8080, "port d'écoute")
verbeux := flag.Bool("v", false, "mode verbeux")
delai := flag.Duration("delai", 5*time.Second, "délai d'expiration")
flag.Parse()

// ./serveur -port 9090 -v fichier.txt
fmt.Println(*port, *verbeux, *delai, flag.Args())
// 9090 true 5s [fichier.txt]
```

`flag` génère automatiquement l'aide (`-h`). Pour des sous-commandes à la
manière de `git` ou `go`, utilisez `flag.NewFlagSet` par sous-commande, ou une
bibliothèque comme `github.com/spf13/cobra`.

## Initialisation paresseuse : sync.Once

`sync.Once` garantit qu'une fonction n'est exécutée qu'**une seule fois**, même
appelée depuis plusieurs goroutines. `sync.OnceValue` et `sync.OnceValues`
(**Go 1.21**) renvoient en plus le résultat :

```go
var config = sync.OnceValue(func() map[string]string {
	return chargeConfig() // exécuté au premier appel seulement
})

env := config()["env"]
```

## Bases de données : database/sql

`database/sql` définit une interface commune à toutes les bases SQL. Le pilote
([PostgreSQL](https://www.postgresql.org), [MySQL](https://www.mysql.com), [SQLite](https://sqlite.org)…) est fourni par un module externe :

```go
db, err := sql.Open("pgx", "postgres://localhost/boutique")
if err != nil {
	return err
}
defer db.Close()

var nom string
err = db.QueryRowContext(ctx, "SELECT nom FROM clients WHERE id = $1", id).Scan(&nom)
if errors.Is(err, sql.ErrNoRows) {
	return ErrIntrouvable
}
```

`sql.DB` est un **pool de connexions**, sûr pour un usage concurrent : créez-en
un seul au démarrage et partagez-le. Utilisez toujours des paramètres (`$1`,
`?`) plutôt que de concaténer des valeurs dans la requête, pour éviter les
injections SQL.

## D'autres packages à connaître

| Package           | Rôle                                                         |
|-------------------|--------------------------------------------------------------|
| `os/exec`         | Lancer des commandes externes                                |
| `os/signal`       | Réagir aux signaux du système (`SIGTERM`…)                   |
| `net`             | TCP, UDP, sockets Unix, résolution DNS                       |
| `net/netip`       | Adresses IP compactes et comparables (Go 1.18)               |
| `net/url`         | Analyse et construction d'URL                                |
| `html/template`   | Gabarits HTML protégés contre les injections                 |
| `archive/zip`, `archive/tar`, `compress/gzip` | Archives et compression          |
| `image`, `image/png`, `image/jpeg` | Lecture et écriture d'images                |
| `math/big`        | Entiers et rationnels de précision arbitraire                |
| `runtime/pprof`, `net/http/pprof` | Profilage CPU et mémoire                     |
| `runtime/debug`   | Informations de build, réglages du GC                        |
| `expvar`          | Exposer des compteurs en HTTP                                |
| `hash/maphash`    | Fonctions de hachage rapides pour tables de hachage          |
| `iter`, `unique`, `weak` | Itérateurs, internement, pointeurs faibles            |

La documentation complète est sur [pkg.go.dev/std](https://pkg.go.dev/std).

## Exercices

Les solutions sont dans
[`examples/exercices/bibliotheque/divers`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/bibliotheque/divers),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Extraire des adresses ★

```go
func Emails(texte string) []string
```

Trouvez les adresses électroniques d'un texte avec `regexp`. Compilez
l'expression une seule fois, dans une variable du package.

### Totaliser un CSV ★★

```go
func TotalParCategorie(r io.Reader) (map[string]float64, error)
```

Lisez un CSV `produit,categorie,prix` (avec une ligne d'en-tête), et
additionnez les prix par catégorie. Un champ entre guillemets peut contenir
une virgule ; un prix invalide donne une erreur qui cite sa ligne :
`ligne 3 : prix "six" invalide`.

### Signer un message ★★★

```go
func Signe(cle []byte, message string) string        // HMAC-SHA256, en hexadécimal
func Verifie(cle []byte, message, signature string) bool
```

Une signature HMAC prouve qu'un message vient de quelqu'un qui connaît la clé,
et qu'il n'a pas été modifié. Comparez avec `hmac.Equal`, en temps constant.
Le test vérifie votre résultat sur une valeur de référence de la RFC 4231.
