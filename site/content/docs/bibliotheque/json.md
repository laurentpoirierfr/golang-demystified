---
title: "III.5 JSON"
description: "encoder, décoder, personnaliser, traiter des flux, et encoding/json/v2."
weight: 50
---

# Chapitre III.5 — JSON : encoding/json

> [!NOTE]
> Les exemples de ce chapitre sont compilés et testés à chaque modification :
> [`examples/stdlib/json`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/stdlib/json).

## Encoder et décoder

`json.Marshal` transforme une valeur Go en JSON, `json.Unmarshal` fait
l'inverse. Seuls les champs **exportés** sont pris en compte, et les
**balises** `json:"…"` précisent le nom et les options de chaque champ :

```go
type Adresse struct {
	Ville string `json:"ville"`
	Pays  string `json:"pays,omitempty"`
}

type Utilisateur struct {
	ID         int       `json:"id"`
	Nom        string    `json:"nom"`
	Email      string    `json:"email,omitempty"`
	Admin      bool      `json:"admin,omitzero"`
	Creation   time.Time `json:"creation,omitzero"`
	Adresse    *Adresse  `json:"adresse,omitempty"`
	Tags       []string  `json:"tags"`
	MotDePasse string    `json:"-"`
}

u := Utilisateur{ID: 1, Nom: "Alice", Adresse: &Adresse{Ville: "Lyon"}, MotDePasse: "secret"}
data, err := json.Marshal(u)
// {"id":1,"nom":"Alice","adresse":{"ville":"Lyon"},"tags":null}
```

| Option de balise | Effet                                                       |
|------------------|-------------------------------------------------------------|
| `json:"nom"`     | Nom de la clé JSON                                          |
| `json:"-"`       | Champ ignoré                                                |
| `omitempty`      | Omis si vide : `""`, `0`, `false`, nil, slice ou map vide   |
| `omitzero`       | Omis si égal à sa valeur zéro, ou si `IsZero()` renvoie vrai (Go 1.24) |
| `string`         | Nombre encodé dans une chaîne : `"42"`                      |

> [!TIP]
> `omitempty` n'a aucun effet sur une structure, et notamment sur un
> `time.Time`. `omitzero` (**Go 1.24**) résout ce problème : il utilise la
> méthode `IsZero()` si elle existe.

`json.MarshalIndent(v, "", "  ")` produit un JSON indenté, lisible par un
humain.

Au décodage, les champs JSON inconnus sont ignorés et les champs absents
gardent leur valeur actuelle. La correspondance des noms ne tient pas compte de
la casse :

```go
entree := `{"id": 2, "nom": "Bob", "creation": "2026-08-12T10:00:00Z", "inconnu": true}`
var u Utilisateur
if err := json.Unmarshal([]byte(entree), &u); err != nil {
	return err
}
// u.ID = 2, u.Nom = "Bob", u.Creation = 2026-08-12 10:00:00 UTC
```

## Correspondance des types

| JSON            | Go                                                    |
|-----------------|-------------------------------------------------------|
| objet           | `struct` ou `map[string]T`                            |
| tableau         | slice ou tableau                                      |
| chaîne          | `string`, `time.Time` (RFC 3339), `[]byte` (Base64)   |
| nombre          | `int`, `float64`…                                     |
| `true`, `false` | `bool`                                                |
| `null`          | `nil` (pointeur, slice, map, interface)               |

Décodé dans un `any`, un nombre JSON devient un `float64`, un objet un
`map[string]any` et un tableau un `[]any`. Préférez autant que possible des
structures typées.

## Les erreurs

```go
err := json.Unmarshal([]byte(`{"id": "trois"}`), &u)
var typeErr *json.UnmarshalTypeError
if errors.As(err, &typeErr) {
	fmt.Println(typeErr.Field) // id
}
```

Les erreurs de syntaxe sont de type `*json.SyntaxError`, et indiquent la
position de l'erreur dans l'entrée.

## Les flux

`json.NewDecoder` lit depuis un `io.Reader` : le corps d'une requête HTTP, un
fichier, ou une suite de documents JSON (format *JSON Lines*).
`json.NewEncoder` écrit vers un `io.Writer`.

```go
dec := json.NewDecoder(entree)
for dec.More() {
	var a Adresse
	if err := dec.Decode(&a); err != nil {
		return err
	}
	fmt.Println(a.Ville)
}

enc := json.NewEncoder(os.Stdout)
enc.Encode(Adresse{Ville: "Pau"})
// {"ville":"Pau"}
```

`dec.DisallowUnknownFields()` rejette les champs inconnus, ce qui est utile
pour valider une entrée.

## Personnaliser l'encodage

Un type contrôle sa représentation en implémentant `json.Marshaler` et
`json.Unmarshaler` (`MarshalJSON`, `UnmarshalJSON`). Lorsque la valeur se
représente par une simple chaîne, il est plus simple d'implémenter
`encoding.TextMarshaler` et `encoding.TextUnmarshaler` : la représentation
textuelle sert alors aussi aux clés de map, aux formats YAML, TOML, etc.

```go
type Statut int

const (
	Brouillon Statut = iota
	Publie
)

func (s Statut) MarshalText() ([]byte, error) {
	switch s {
	case Brouillon:
		return []byte("brouillon"), nil
	case Publie:
		return []byte("publié"), nil
	}
	return nil, fmt.Errorf("statut inconnu : %d", s)
}

func (s *Statut) UnmarshalText(b []byte) error {
	switch string(b) {
	case "brouillon":
		*s = Brouillon
	case "publié":
		*s = Publie
	default:
		return fmt.Errorf("statut inconnu : %q", b)
	}
	return nil
}

type Article struct {
	Titre  string `json:"titre"`
	Statut Statut `json:"statut"`
}

data, _ := json.Marshal(Article{"Go 1.27", Publie})
// {"titre":"Go 1.27","statut":"publié"}
```

## encoding/json/v2

> [!NOTE]
> **Go 1.27** — le package `encoding/json/v2` et son compagnon
> `encoding/json/jsontext` rejoignent la bibliothèque standard. Ils étaient
> expérimentaux depuis Go 1.25 (`GOEXPERIMENT=jsonv2`).

`encoding/json` existe depuis Go 1.0, et la promesse de compatibilité
empêchait de corriger ses défauts : performances moyennes, correspondance des
noms insensible à la casse, acceptation de doublons, etc. La version 2 les
corrige dans un nouveau package :

```go
import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
)

data, err := jsonv2.Marshal(cfg, jsontext.WithIndent("  "))

err = jsonv2.Unmarshal(entree, &cfg, jsonv2.RejectUnknownMembers(true))
```

Les fonctions acceptent des **options** variadiques plutôt que des méthodes de
configuration. `MarshalWrite` et `UnmarshalRead` travaillent directement avec
un `io.Writer` et un `io.Reader`.

Les principales différences avec la version 1 :

| Comportement                    | v1                       | v2                         |
|---------------------------------|--------------------------|----------------------------|
| Correspondance des noms         | insensible à la casse    | **exacte**                 |
| Noms en double dans un objet    | acceptés (le dernier gagne) | **erreur**              |
| UTF-8 invalide                  | remplacé                 | **erreur**                 |
| Slice ou map nil                | `null`                   | `[]` ou `{}`               |
| `omitempty`                     | selon la valeur Go       | selon la valeur JSON       |
| Ordre des clés d'une map        | trié                     | non déterministe (option `Deterministic`) |

Depuis Go 1.27, `encoding/json` est lui-même **réimplémenté au-dessus de la
v2**, avec les options qui reproduisent son comportement historique. Le
décodage est nettement plus rapide, sans changer une ligne de code. Seul le
texte de certains messages d'erreur peut différer. `GOEXPERIMENT=nojsonv2`
permet temporairement de revenir à l'ancienne implémentation.

Pour un nouveau projet, `encoding/json/v2` est le choix recommandé. Pour un
projet existant, `encoding/json` reste parfaitement supporté.

## Exercices

Les solutions sont dans
[`examples/exercices/bibliotheque/json`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices/bibliotheque/json),
avec leurs tests. Les tests servent de **spécification** : pour vous exercer,
copiez `exercices_test.go` dans un dossier à vous, écrivez les fonctions
demandées, puis lancez `go test`. Les étoiles indiquent la difficulté.

### Des tags ★

Écrivez le type `Produit` (champs `Nom`, `Prix`, `Stock`, `Tags`, `Interne`)
pour que `json.Marshal` produise :

```json
{"nom":"Gopher","prix":12.5}
```

Le stock et les tags sont omis s'ils sont vides, et `Interne` n'est jamais
encodé.

### Décoder strictement ★★

```go
type Config struct {
	Adresse string `json:"adresse"`
	Port    int    `json:"port"`
}

func LitConfig(r io.Reader) (Config, error)
```

Refusez les champs inconnus (`"prot"` au lieu de `"port"` est une faute de
frappe, pas une option), une adresse absente et un port hors de [1, 65535].

### Un encodage personnalisé ★★★

```go
type Duree time.Duration
```

Faites que `Duree` s'encode en texte lisible, `"1h30m0s"`, et se décode
depuis ce texte, en satisfaisant `json.Marshaler` et `json.Unmarshaler`.
Lequel des deux récepteurs doit être un pointeur ?
