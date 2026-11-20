// Package valide vérifie une structure d'après les étiquettes de ses champs,
// par réflexion :
//
//	type Tache struct {
//		Titre    string `valide:"requis,max=100"`
//		Priorite int    `valide:"min=1,max=5"`
//	}
//
// C'est le principe de bibliothèques comme go-playground/validator : le code
// ne connaît pas le type à l'avance, il le découvre à l'exécution.
package valide

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Erreur décrit les champs invalides.
type Erreur struct {
	Champs map[string]string // nom du champ → problème
}

func (e *Erreur) Error() string {
	var b strings.Builder
	b.WriteString("validation :")
	for _, nom := range slices.Sorted(maps.Keys(e.Champs)) { // l'ordre d'une map n'est pas défini
		fmt.Fprintf(&b, " %s %s ;", nom, e.Champs[nom])
	}
	return strings.TrimSuffix(b.String(), " ;")
}

// Structure vérifie v, une structure ou un pointeur vers une structure.
func Structure(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return errors.New("validation : pointeur nil")
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("validation : structure attendue, reçu %s", val.Kind())
	}
	e := &Erreur{Champs: map[string]string{}}
	verifie(val, "", e)
	if len(e.Champs) > 0 {
		return e
	}
	return nil
}

func verifie(val reflect.Value, prefixe string, e *Erreur) {
	// Value.Fields (Go 1.26) parcourt les champs et leurs valeurs.
	for champ, valeur := range val.Fields() {
		if !champ.IsExported() {
			continue // la réflexion ne lit pas les champs non exportés
		}
		nom := prefixe + champ.Name
		if valeur.Kind() == reflect.Struct { // structure imbriquée
			verifie(valeur, nom+".", e)
			continue
		}
		regles := champ.Tag.Get("valide")
		if regles == "" {
			continue
		}
		for regle := range strings.SplitSeq(regles, ",") {
			if probleme := applique(strings.TrimSpace(regle), valeur); probleme != "" {
				e.Champs[nom] = probleme
				break
			}
		}
	}
}

// applique vérifie une règle (« requis », « min=1 », « max=100 »).
func applique(regle string, v reflect.Value) string {
	cle, arg, _ := strings.Cut(regle, "=")
	switch cle {
	case "requis":
		if v.IsZero() {
			return "est obligatoire"
		}
		return ""
	case "min", "max":
		limite, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return "règle invalide : " + regle
		}
		mesure, quoi, ok := taille(v)
		if !ok {
			return "règle " + cle + " inapplicable au type " + v.Type().String()
		}
		if quoi != "" {
			arg += " " + quoi // « 3 éléments », mais « 5 » pour un nombre
		}
		if cle == "min" && mesure < limite {
			return "doit valoir au moins " + arg
		}
		if cle == "max" && mesure > limite {
			return "doit valoir au plus " + arg
		}
		return ""
	default:
		return "règle inconnue : " + regle
	}
}

// taille mesure une valeur : sa longueur en caractères pour un texte, son
// nombre d'éléments pour une tranche, sa valeur pour un nombre.
func taille(v reflect.Value) (float64, string, bool) {
	switch v.Kind() {
	case reflect.String:
		return float64(utf8.RuneCountInString(v.String())), "caractères", true
	case reflect.Slice, reflect.Map, reflect.Array:
		return float64(v.Len()), "éléments", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), "", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), "", true
	case reflect.Float32, reflect.Float64:
		return v.Float(), "", true
	}
	return 0, "", false
}
