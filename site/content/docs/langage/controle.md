---
title: "Contrôle de flux"
linkTitle: "3. Contrôle de flux"
weight: 30
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/controle`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/controle).

Go a peu d'instructions de contrôle : `if`, `for`, `switch`, `select` (vu avec
la concurrence), `defer` et `goto`. Il n'y a pas de parenthèses autour des
conditions, et les accolades sont **obligatoires**.

## if

```go
temperature := 23
if temperature > 30 {
	fmt.Println("chaud")
} else if temperature > 15 {
	fmt.Println("doux")
} else {
	fmt.Println("froid")
}
// doux
```

Un `if` peut commencer par une **instruction d'initialisation**. La variable
déclarée n'est visible que dans le `if` et ses `else`. C'est la forme
idiomatique pour traiter une erreur :

```go
if _, err := os.Stat("/fichier/inexistant"); err != nil {
	fmt.Println("introuvable")
}
```

Le code Go privilégie le **retour anticipé** : on traite les cas d'erreur
d'abord, avec un `return`, et le cas nominal reste aligné à gauche, sans
`else`.

```go
func charger(chemin string) (*Config, error) {
	data, err := os.ReadFile(chemin)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

## for : la seule boucle

`for` couvre toutes les formes de boucle :

```go
// La forme classique, en trois clauses.
for i := 0; i < 3; i++ {
	fmt.Print(i)
}
// 012

// La forme « while ».
n := 1
for n < 100 {
	n *= 3
}
// n vaut 243

// La boucle infinie, interrompue par break.
for {
	if termine() {
		break
	}
}
```

`continue` passe à l'itération suivante, `break` sort de la boucle.

### range

`range` parcourt une slice, un tableau, une chaîne, une map, un channel, un
entier ou une fonction itérateur :

```go
langages := []string{"Go", "Rust", "Zig"}
for i, l := range langages { // indice et valeur
	fmt.Println(i, l)
}
for _, l := range langages { // valeur seule
	fmt.Print(strings.ToUpper(l))
}
for i := range langages { // indice seul
	fmt.Print(i)
}
```

| Expression parcourue | Première valeur | Seconde valeur |
|----------------------|-----------------|----------------|
| slice, tableau       | indice          | élément (copie) |
| `string`             | indice en octets | `rune`         |
| map                  | clé             | valeur (ordre aléatoire) |
| channel              | valeur reçue    | —              |
| entier `n`           | `0` à `n-1`     | —              |
| fonction itérateur   | selon l'itérateur | selon l'itérateur |

> [!NOTE]
> **Go 1.22** — `range` accepte un entier :
>
> ```go
> for i := range 3 {
> 	fmt.Print(i)
> }
> // 012
> ```

> [!NOTE]
> **Go 1.23** — `range` accepte une fonction itérateur, comme
> `maps.Keys(m)` ou `strings.SplitSeq(s, ",")`. Les itérateurs sont présentés
> dans le chapitre sur les fonctions.

### Une variable par itération

Depuis **Go 1.22**, chaque itération d'une boucle `for` crée une **nouvelle
variable**. Une closure ou une goroutine qui capture la variable de boucle voit
donc la valeur de son itération :

```go
var fonctions []func()
for i := range 3 {
	fonctions = append(fonctions, func() { fmt.Print(i) })
}
for _, f := range fonctions {
	f()
}
// 012
```

Avant Go 1.22, la variable était partagée par toutes les itérations, et ce
programme affichait `333`. Il fallait recopier la variable (`i := i`) dans le
corps de la boucle. Ce comportement dépend de la version déclarée dans
`go.mod`, et `go fix` supprime les copies devenues inutiles.

### Les labels

`break` et `continue` agissent sur la boucle la plus interne. Un **label**
permet de cibler une boucle externe :

```go
grille := [][]int{{1, 2}, {3, -1}, {5, 6}}
recherche:
	for i, ligne := range grille {
		for j, v := range ligne {
			if v < 0 {
				fmt.Println("négatif en", i, j)
				break recherche
			}
		}
	}
// négatif en 1 1
```

`goto label` existe aussi, mais il est rarement utilisé.

## switch

Le `switch` de Go est plus souple que celui du C : les `case` ne « tombent »
pas dans le suivant (pas de `break` à écrire), un `case` peut lister plusieurs
valeurs, et les valeurs ne sont pas limitées aux constantes.

```go
func jour(n int) string {
	switch n {
	case 1, 2, 3, 4, 5:
		return "semaine"
	case 6, 7:
		return "week-end"
	default:
		return "inconnu"
	}
}
```

Un `switch` **sans expression** équivaut à `switch true`. C'est une façon
lisible d'écrire une suite de `if`/`else if` :

```go
func mention(note int) string {
	switch {
	case note >= 16:
		return "très bien"
	case note >= 14:
		return "bien"
	case note >= 10:
		return "passable"
	default:
		return "ajourné"
	}
}
```

Comme `if`, un `switch` accepte une instruction d'initialisation. Le mot-clé
`fallthrough` force l'exécution du `case` suivant :

```go
switch n := 1; n {
case 1:
	fmt.Println("un")
	fallthrough
case 2:
	fmt.Println("deux")
case 3:
	fmt.Println("trois")
}
// un
// deux
```

Le *type switch*, qui teste le type dynamique d'une interface, est présenté
dans le chapitre sur les interfaces.

## defer

`defer` programme l'exécution d'un appel de fonction **à la sortie** de la
fonction englobante, qu'elle se termine normalement, par un `return` ou par un
`panic`. Il sert à libérer une ressource juste après l'avoir obtenue, ce qui
évite de l'oublier :

```go
f, err := os.Open(chemin)
if err != nil {
	return err
}
defer f.Close()

mu.Lock()
defer mu.Unlock()
```

Les appels différés sont exécutés dans l'**ordre inverse** de leur déclaration
(pile LIFO). Leurs arguments sont évalués **au moment du `defer`** :

```go
fmt.Println("début")
for i := range 3 {
	defer fmt.Println("defer", i)
}
fmt.Println("fin")
// début
// fin
// defer 2
// defer 1
// defer 0
```

> [!TIP]
> `defer` s'exécute à la fin de la **fonction**, pas du bloc. Dans une boucle qui
> ouvre de nombreux fichiers, placez le traitement de chaque fichier dans une
> fonction séparée, sinon tous les fichiers restent ouverts jusqu'à la fin de
> la boucle.
