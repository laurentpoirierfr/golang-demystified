---
title: "Gestion des erreurs"
linkTitle: "8. Gestion des erreurs"
weight: 80
---

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/langage/erreurs`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/langage/erreurs).

## Les erreurs sont des valeurs

Go n'a pas d'exceptions. Une fonction qui peut échouer renvoie une valeur de
type `error` en dernière position, et l'appelant la teste :

```go
data, err := os.ReadFile("config.json")
if err != nil {
	return fmt.Errorf("lecture de la configuration : %w", err)
}
```

`error` est une interface intégrée au langage :

```go
type error interface {
	Error() string
}
```

Ce style est souvent jugé verbeux, mais il a un avantage décisif : chaque
chemin d'erreur est **visible** dans le code. On voit, en lisant une
fonction, où elle peut échouer et ce qu'elle fait alors.

Quelques règles d'usage :

- traitez l'erreur **immédiatement**, avant d'utiliser les autres valeurs
  renvoyées ;
- une erreur doit être soit traitée, soit renvoyée à l'appelant, mais pas les
  deux (évitez de la journaliser *et* de la renvoyer) ;
- les messages d'erreur commencent par une minuscule et ne se terminent pas par
  une ponctuation, car ils sont souvent concaténés.

## Créer une erreur

```go
errors.New("connexion refusée")
fmt.Errorf("port %d invalide", port)
```

### Les erreurs sentinelles

Une **erreur sentinelle** est une valeur exportée, que l'appelant peut
reconnaître. La bibliothèque standard en définit beaucoup : `io.EOF`,
`fs.ErrNotExist`, `sql.ErrNoRows`, `context.Canceled`…

```go
var ErrIntrouvable = errors.New("utilisateur introuvable")

func trouve(id int) (*Utilisateur, error) {
	if id != 1 {
		return nil, ErrIntrouvable
	}
	return &Utilisateur{ID: 1}, nil
}
```

### Les types d'erreur

Lorsque l'appelant a besoin d'informations, on définit un **type** qui
implémente `error` :

```go
type ErrValidation struct {
	Champ  string
	Raison string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("champ %q invalide : %s", e.Champ, e.Raison)
}
```

## Envelopper une erreur

Depuis **Go 1.13**, le verbe `%w` de `fmt.Errorf` **enveloppe** une erreur :
le message gagne du contexte, et l'erreur d'origine reste accessible.

```go
func charge(id int) (*Utilisateur, error) {
	u, err := trouve(id)
	if err != nil {
		return nil, fmt.Errorf("chargement de l'utilisateur %d : %w", id, err)
	}
	return u, nil
}

_, err := charge(42)
fmt.Println(err)
// chargement de l'utilisateur 42 : utilisateur introuvable
```

Chaque niveau ajoute son contexte, et le message final se lit comme une pile
d'appels lisible par un humain. Utilisez `%v` plutôt que `%w` lorsque vous ne
voulez pas que l'erreur d'origine fasse partie de votre API.

## Inspecter une erreur

On ne compare pas une erreur enveloppée avec `==`, ni son message avec
`strings.Contains`. Le package `errors` parcourt la chaîne des erreurs
enveloppées :

| Fonction                  | Rôle                                                  | Depuis  |
|---------------------------|-------------------------------------------------------|---------|
| `errors.Is(err, cible)`   | Une erreur de la chaîne est-elle égale à `cible` ?    | Go 1.13 |
| `errors.As(err, &cible)`  | Une erreur de la chaîne a-t-elle le type de `cible` ? | Go 1.13 |
| `errors.AsType[T](err)`   | Version générique de `As`                             | Go 1.26 |
| `errors.Unwrap(err)`      | L'erreur enveloppée directement                       | Go 1.13 |
| `errors.Join(errs...)`    | Combine plusieurs erreurs en une seule                | Go 1.20 |

```go
fmt.Println(errors.Is(err, ErrIntrouvable))
// true
```

`errors.As` cherche une erreur d'un type donné et la copie dans la variable
cible :

```go
err := fmt.Errorf("formulaire : %w", valide("-3"))

var ev *ErrValidation
if errors.As(err, &ev) {
	fmt.Println("champ :", ev.Champ, "-", ev.Raison)
}
// champ : age - négatif
```

> [!NOTE]
> **Go 1.26** — `errors.AsType` fait la même chose sans variable préalable, et
> le compilateur vérifie le type :
>
> ```go
> _, err := os.Open("/fichier/inexistant")
> if pe, ok := errors.AsType[*fs.PathError](err); ok {
> 	fmt.Println(pe.Op, pe.Path)
> }
> // open /fichier/inexistant
> ```
>
> `go fix` convertit automatiquement les appels à `errors.As`.

### Combiner des erreurs

Depuis **Go 1.20**, `errors.Join` regroupe plusieurs erreurs, par exemple lors
de la validation d'un formulaire. `errors.Is` et `errors.As` examinent chacune
d'elles. `fmt.Errorf` accepte aussi plusieurs `%w`.

```go
var errs []error
for _, v := range []string{"12", "x", "-1"} {
	if err := valide(v); err != nil {
		errs = append(errs, err)
	}
}
err := errors.Join(errs...) // nil si errs est vide
fmt.Println(err)
// champ "age" invalide : pas un nombre
// champ "age" invalide : négatif
```

## panic et recover

`panic` interrompt l'exécution normale : la fonction s'arrête, ses `defer`
s'exécutent, puis ceux de l'appelant, et ainsi de suite. Si rien ne
l'intercepte, le programme s'arrête en affichant la pile d'appels.

Le runtime panique en cas d'erreur de programmation : accès hors des bornes
d'une slice, déréférencement d'un pointeur nil, écriture dans une map nil,
division entière par zéro…

`panic` n'est **pas** un mécanisme de gestion d'erreurs. Réservez-le aux
situations impossibles, qui révèlent un bug, ou à l'initialisation d'un
programme qui ne peut pas démarrer. Par convention, les fonctions `MustXxx`
paniquent au lieu de renvoyer une erreur, comme `regexp.MustCompile` : elles
sont destinées aux valeurs connues à l'écriture du code.

`recover`, appelé dans une fonction différée, arrête la panique et renvoie la
valeur passée à `panic` :

```go
func divisionSure(a, b int) (resultat int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic récupéré : %v", r)
		}
	}()
	return a / b, nil
}

fmt.Println(divisionSure(1, 0))
// 0 panic récupéré : runtime error: integer divide by zero
```

Un serveur HTTP utilise ce mécanisme pour qu'une requête qui panique
n'arrête pas tout le serveur. `recover` ne fonctionne que dans la goroutine
qui a paniqué : une panique dans une autre goroutine arrête le programme.
