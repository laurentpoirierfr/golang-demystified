---
title: "Texte : fmt, strings, strconv"
linkTitle: "1. Texte"
weight: 10
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/texte`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/texte).

## fmt : formater et afficher

Le package `fmt` formate des valeurs selon des **verbes** inspirés du `printf`
du C. Ses fonctions se déclinent en trois familles :

| Famille                     | Destination                               |
|-----------------------------|-------------------------------------------|
| `Print`, `Println`, `Printf` | la sortie standard                       |
| `Sprint`, `Sprintln`, `Sprintf` | une chaîne renvoyée                   |
| `Fprint`, `Fprintln`, `Fprintf` | un `io.Writer` (fichier, réponse HTTP…) |
| `Append`, `Appendln`, `Appendf` | une slice d'octets (Go 1.19)          |

`fmt.Errorf` crée une erreur à partir d'un format, et enveloppe une autre
erreur avec `%w`.

### Les verbes

```go
p := Point{1, 2}
fmt.Printf("%v | %+v | %#v | %T\n", p, p, p, p)
// {1 2} | {X:1 Y:2} | texte.Point{X:1, Y:2} | texte.Point
fmt.Printf("%d | %5d | %-5d| %05d | %x | %b\n", 42, 42, 42, 42, 255, 5)
// 42 |    42 | 42   | 00042 | ff | 101
fmt.Printf("%f | %.2f | %8.3f | %e | %g\n", 3.14159, 3.14159, 3.14159, 1234.5678, 0.000012)
// 3.141590 | 3.14 |    3.142 | 1.234568e+03 | 1.2e-05
fmt.Printf("%s | %q | %10s | %-10s|\n", "go", "go", "droite", "gauche")
// go | "go" |     droite | gauche    |
fmt.Printf("%t | %c | %U\n", true, 'é', 'é')
// true | é | U+00E9
```

| Verbe       | Rôle                                                       |
|-------------|------------------------------------------------------------|
| `%v`        | Format par défaut ; `%+v` ajoute les noms des champs       |
| `%#v`       | Syntaxe Go de la valeur                                    |
| `%T`        | Type de la valeur                                          |
| `%d`, `%b`, `%o`, `%x` | Entier en décimal, binaire, octal, hexadécimal  |
| `%f`, `%e`, `%g` | Flottant : décimal, scientifique, le plus compact     |
| `%s`, `%q`  | Chaîne, chaîne entre guillemets échappée                   |
| `%t`        | Booléen                                                    |
| `%c`, `%U`  | Caractère, point de code Unicode                           |
| `%p`        | Adresse d'un pointeur                                      |
| `%w`        | Erreur enveloppée (`fmt.Errorf` uniquement)                |
| `%%`        | Le caractère `%`                                           |

Un nombre entre `%` et le verbe fixe la **largeur**, `.n` la **précision**, `-`
aligne à gauche et `0` complète avec des zéros. `go vet` vérifie la cohérence
entre les verbes et les arguments.

Un type qui implémente `String() string` (l'interface `fmt.Stringer`) contrôle
son affichage par `%v` et `%s`.

## strings : manipuler des chaînes

Les chaînes sont immuables : les fonctions de `strings` renvoient de nouvelles
chaînes.

```go
s := "  Le Gopher creuse  "
strings.TrimSpace(s)                       // "Le Gopher creuse"
strings.ToUpper("go")                      // "GO"
strings.Contains(s, "Gopher")              // true
strings.HasPrefix("golang", "go")          // true
strings.HasSuffix("main.go", ".go")        // true
strings.Index("chercher", "er")            // 2
strings.Count("cheese", "e")               // 3
strings.ReplaceAll("a-b-c", "-", "+")      // "a+b+c"
strings.Split("a,b,c", ",")                // [a b c]
strings.Fields(" un  deux trois ")         // [un deux trois]
strings.Join([]string{"a", "b"}, "/")      // "a/b"
strings.Repeat("ab", 3)                    // "ababab"
strings.EqualFold("Go", "GO")              // true : comparaison sans casse
```

Le package `bytes` propose les mêmes fonctions pour les `[]byte`.

### Couper une chaîne

`strings.Cut` (**Go 1.18**) découpe une chaîne autour de la première
occurrence d'un séparateur. Il remplace la plupart des usages de `Index` suivis
d'un découpage manuel :

```go
cle, valeur, ok := strings.Cut("port=8080", "=")
// "port", "8080", true

reste, ok := strings.CutPrefix("Bearer abc123", "Bearer ") // Go 1.20
// "abc123", true

dossier, fichier, _ := strings.CutLast("/home/gopher/notes.txt", "/")
// "/home/gopher", "notes.txt"
```

> [!NOTE]
> **Go 1.27** — `strings.CutLast` et `bytes.CutLast` découpent autour de la
> **dernière** occurrence du séparateur.

### Construire une chaîne

Concaténer avec `+` dans une boucle recopie la chaîne à chaque tour.
`strings.Builder` accumule les morceaux dans un tampon. Il implémente
`io.Writer`, et donc accepte `fmt.Fprintf` :

```go
var b strings.Builder
for i := range 3 {
	fmt.Fprintf(&b, "ligne %d\n", i)
}
b.WriteString("fin")
fmt.Println(b.String())
```

### Parcourir sans allouer

Depuis **Go 1.24**, des itérateurs évitent de construire une slice
intermédiaire : `Lines`, `SplitSeq`, `SplitAfterSeq`, `FieldsSeq` et
`FieldsFuncSeq`.

```go
for ligne := range strings.Lines("alpha\nbeta\ngamma\n") {
	fmt.Printf("%q;", ligne) // chaque ligne garde son \n
}
// "alpha\n";"beta\n";"gamma\n";

for mot := range strings.FieldsSeq("un deux  trois") {
	fmt.Print("[", mot, "]")
}
// [un][deux][trois]
```

## strconv : texte et nombres

`strconv` convertit entre chaînes et types de base, avec un contrôle précis
du format et des erreurs :

```go
n, err := strconv.Atoi("42")              // int
f, err := strconv.ParseFloat("3.14", 64)  // float64
b, err := strconv.ParseBool("true")       // bool
i, err := strconv.ParseInt("ff", 16, 64)  // 255, en base 16

strconv.Itoa(42)                          // "42"
strconv.FormatFloat(3.14159, 'f', 2, 64)  // "3.14"
strconv.FormatInt(255, 2)                 // "11111111"
strconv.Quote(`dit "bonjour"`)            // `"dit \"bonjour\""`
```

Les erreurs indiquent la fonction, l'entrée et la cause :

```go
_, err := strconv.ParseInt("999999999999999999999", 10, 64)
fmt.Println(err)
// strconv.ParseInt: parsing "999999999999999999999": value out of range
```

## unicode et utf8

Le package `unicode` classe les runes (`IsLetter`, `IsDigit`, `IsSpace`,
`IsUpper`, `ToUpper`…), et `unicode/utf8` encode et décode l'UTF-8
(`RuneCountInString`, `ValidString`, `DecodeRuneInString`…).

```go
compte := map[string]int{}
for _, r := range "Go 1.27, été !" {
	switch {
	case unicode.IsLetter(r):
		compte["lettres"]++
	case unicode.IsDigit(r):
		compte["chiffres"]++
	case unicode.IsSpace(r):
		compte["espaces"]++
	default:
		compte["autres"]++
	}
}
fmt.Println(compte)
// map[autres:3 chiffres:3 espaces:3 lettres:5]
```

> [!NOTE]
> **Go 1.27** — les tables du package `unicode` passent à Unicode 17.

## Aligner des colonnes : text/tabwriter

`text/tabwriter` aligne des colonnes séparées par des tabulations. Il est très
utile pour les outils en ligne de commande :

```go
w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "NOM\tVERSION\tANNÉE")
fmt.Fprintln(w, "Go\t1.27\t2026")
fmt.Fprintln(w, "Generics\t1.18\t2022")
w.Flush()
// NOM       VERSION  ANNÉE
// Go        1.27     2026
// Generics  1.18     2022
```

Pour des besoins plus avancés, `regexp` gère les expressions régulières
(syntaxe RE2, en temps linéaire garanti), et `text/template` génère du texte à
partir de gabarits (voir le chapitre consacré aux templates).
