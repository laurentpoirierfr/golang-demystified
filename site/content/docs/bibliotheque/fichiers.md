---
title: "Fichiers et flux : io, os, bufio"
weight: 30
---

## io.Reader et io.Writer

Toutes les entrées-sorties de Go reposent sur deux interfaces d'une seule
méthode :

```go
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
```

Un fichier, une connexion réseau, le corps d'une requête HTTP, un tampon en
mémoire, un compresseur, une fonction de hachage… implémentent l'une ou
l'autre. Une fonction écrite pour un `io.Reader` fonctionne donc avec toutes
ces sources.

`Read` renvoie l'erreur `io.EOF` à la fin des données. On utilise rarement
`Read` directement : les fonctions utilitaires gèrent ces détails.

| Fonction                       | Rôle                                            |
|--------------------------------|-------------------------------------------------|
| `io.ReadAll(r)`                | Lit tout le contenu en mémoire                  |
| `io.Copy(w, r)`                | Copie de `r` vers `w`, par blocs                |
| `io.CopyN(w, r, n)`            | Copie `n` octets                                |
| `io.LimitReader(r, n)`         | Limite la lecture à `n` octets                  |
| `io.MultiWriter(w1, w2…)`      | Écrit dans plusieurs destinations à la fois     |
| `io.MultiReader(r1, r2…)`      | Enchaîne plusieurs sources                      |
| `io.TeeReader(r, w)`           | Recopie dans `w` ce qui est lu depuis `r`       |
| `io.Pipe()`                    | Relie un `Writer` à un `Reader`, en mémoire     |

```go
var dst bytes.Buffer
n, err := io.Copy(&dst, strings.NewReader("contenu à copier"))
// 17 <nil>
```

`strings.NewReader` et `bytes.NewReader` transforment une chaîne ou des octets
en `io.Reader` ; `bytes.Buffer` est à la fois un `Reader` et un `Writer`.

### Composer des flux

La force de ces interfaces est leur composition. Ici, on compresse un texte
tout en calculant l'empreinte SHA-256 du résultat, en un seul passage et sans
fichier intermédiaire :

```go
h := sha256.New()
var compresse bytes.Buffer
gz := gzip.NewWriter(io.MultiWriter(&compresse, h))
io.Copy(gz, strings.NewReader(strings.Repeat("gopher ", 1000)))
gz.Close()

// Décompresse en lisant au plus 13 octets.
gr, _ := gzip.NewReader(&compresse)
debut, _ := io.ReadAll(io.LimitReader(gr, 13))
fmt.Printf("%q\n", debut)
// "gopher gopher"
```

> [!TIP]
> Pour un flux de taille inconnue, préférez `io.Copy` à `io.ReadAll` : il
> travaille par blocs, avec une mémoire constante. Limitez toujours la taille
> d'une entrée qui vient de l'extérieur avec `io.LimitReader` ou
> `http.MaxBytesReader`.

## Lire et écrire des fichiers

Pour un fichier de taille raisonnable, deux fonctions suffisent (**Go 1.16**,
elles remplacent l'ancien package `io/ioutil`) :

```go
err := os.WriteFile(chemin, []byte("ligne 1\nligne 2\n"), 0o644)
data, err := os.ReadFile(chemin)
```

Une erreur d'absence se teste avec `errors.Is(err, fs.ErrNotExist)`.

Pour un contrôle plus fin, `os.Open` ouvre en lecture, `os.Create` crée ou
tronque, et `os.OpenFile` accepte des options :

```go
f, err := os.OpenFile("journal.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
	return err
}
defer f.Close()
fmt.Fprintln(f, "nouvelle entrée")
```

> [!WARNING]
> Pour un fichier ouvert en **écriture**, l'erreur de `Close` compte : c'est
> parfois à ce moment que les données sont réellement écrites. Ne l'ignorez
> pas avec un simple `defer f.Close()` lorsque l'intégrité des données
> importe.

## bufio : lire ligne par ligne

`bufio.Scanner` découpe un flux en lignes (ou en mots, avec
`sc.Split(bufio.ScanWords)`) :

```go
sc := bufio.NewScanner(entree) // un fichier, os.Stdin…
for sc.Scan() {
	ligne := sc.Text()
	…
}
if err := sc.Err(); err != nil {
	return err
}
```

Par défaut, une ligne ne peut pas dépasser 64 Ko. `sc.Buffer` augmente cette
limite.

`bufio.Writer` regroupe les petites écritures en blocs. N'oubliez pas
d'appeler `Flush` à la fin, sinon les dernières données sont perdues :

```go
w := bufio.NewWriter(os.Stdout)
defer w.Flush()
for i := range 3 {
	fmt.Fprintf(w, "ligne %d\n", i)
}
```

## Chemins et répertoires

`path/filepath` manipule les chemins selon le système d'exploitation (`/` ou
`\`) : `Join`, `Dir`, `Base`, `Ext`, `Abs`, `Rel`… Le package `path`, lui,
utilise toujours `/` : il est destiné aux URL et aux chemins dans un `fs.FS`.

```go
os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755)

filepath.WalkDir(dir, func(chemin string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() && filepath.Ext(chemin) == ".go" {
		fmt.Println(chemin)
	}
	return nil
})
```

`os.ReadDir` liste un répertoire, `os.MkdirTemp` et `os.CreateTemp` créent des
éléments temporaires. Dans un test, `t.TempDir()` crée un répertoire supprimé
automatiquement.

## Les systèmes de fichiers abstraits : io/fs

Depuis **Go 1.16**, l'interface `fs.FS` représente un système de fichiers en
lecture seule : un répertoire du disque (`os.DirFS`), une archive
(`zip.Reader`), des fichiers embarqués (`embed.FS`), ou un système en mémoire
pour les tests (`testing/fstest.MapFS`).

```go
fichiers, _ := fs.Glob(os.DirFS(dir), "a/*.go")
// [a/un.go]
```

Écrire du code qui accepte un `fs.FS` plutôt qu'un chemin le rend facile à
tester.

## Embarquer des fichiers : embed

La directive `//go:embed` (**Go 1.16**) intègre des fichiers dans l'exécutable
à la compilation. Le programme n'a plus besoin de les trouver sur le disque :
c'est idéal pour les gabarits HTML, les fichiers statiques d'un site, les
migrations SQL ou une configuration par défaut.

```go
import "embed"

//go:embed testdata/bienvenue.txt
var bienvenue string

//go:embed testdata
var donnees embed.FS

fmt.Print(bienvenue)
// Bienvenue, gopher !
entrees, _ := fs.ReadDir(donnees, "testdata")
// bienvenue.txt, config.json
```

La variable peut être de type `string`, `[]byte` ou `embed.FS`. Un `embed.FS`
se sert directement en HTTP avec `http.FileServerFS(donnees)`.

## Limiter l'accès à un répertoire : os.Root

> [!NOTE]
> **Go 1.24** — `os.Root` restreint les opérations de fichiers à un
> répertoire.

Lorsqu'un nom de fichier vient de l'extérieur (une requête HTTP, une archive à
extraire), un chemin comme `../../etc/passwd` peut sortir du répertoire
prévu. `os.Root` refuse tout chemin qui s'en échappe, y compris par un lien
symbolique :

```go
root, err := os.OpenRoot(dir)
if err != nil {
	return err
}
defer root.Close()

f, err := root.Create("autorise.txt")     // OK
_, err = root.Open("../../etc/passwd")    // erreur
```

## Variables d'environnement et processus

```go
home := os.Getenv("HOME")
port, ok := os.LookupEnv("PORT") // distingue « vide » et « absent »
args := os.Args[1:]              // arguments de la ligne de commande
os.Exit(1)                       // quitte sans exécuter les defer
```

Pour les arguments, le package `flag` analyse les options de la ligne de
commande. `os/exec` lance d'autres programmes :

```go
sortie, err := exec.Command("git", "rev-parse", "HEAD").Output()
```
