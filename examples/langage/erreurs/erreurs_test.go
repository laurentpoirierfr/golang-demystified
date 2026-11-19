// Exemples du chapitre « Gestion des erreurs ».
package erreurs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

// Une erreur sentinelle : une valeur exportée, comparée avec errors.Is.
var ErrIntrouvable = errors.New("utilisateur introuvable")

type Utilisateur struct{ ID int }

func trouve(id int) (*Utilisateur, error) {
	if id != 1 {
		return nil, ErrIntrouvable
	}
	return &Utilisateur{ID: 1}, nil
}

func charge(id int) (*Utilisateur, error) {
	u, err := trouve(id)
	if err != nil {
		return nil, fmt.Errorf("chargement de l'utilisateur %d : %w", id, err)
	}
	return u, nil
}

func Example_wrap() {
	_, err := charge(42)
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrIntrouvable))
	fmt.Println(errors.Unwrap(err) == ErrIntrouvable)
	// Output:
	// chargement de l'utilisateur 42 : utilisateur introuvable
	// true
	// true
}

// ErrValidation est un type d'erreur qui porte des informations.
type ErrValidation struct {
	Champ  string
	Raison string
}

func (e *ErrValidation) Error() string {
	return fmt.Sprintf("champ %q invalide : %s", e.Champ, e.Raison)
}

func valide(age string) error {
	n, err := strconv.Atoi(age)
	if err != nil {
		return &ErrValidation{Champ: "age", Raison: "pas un nombre"}
	}
	if n < 0 {
		return &ErrValidation{Champ: "age", Raison: "négatif"}
	}
	return nil
}

func Example_as() {
	err := fmt.Errorf("formulaire : %w", valide("-3"))

	var ev *ErrValidation
	if errors.As(err, &ev) {
		fmt.Println("champ :", ev.Champ, "-", ev.Raison)
	}
	// Output: champ : age - négatif
}

func Example_asType() {
	_, err := os.Open("/fichier/inexistant")

	if pe, ok := errors.AsType[*fs.PathError](err); ok {
		fmt.Println(pe.Op, pe.Path)
	}
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	// Output:
	// open /fichier/inexistant
	// true
}

func Example_join() {
	var errs []error
	for _, v := range []string{"12", "x", "-1"} {
		if err := valide(v); err != nil {
			errs = append(errs, err)
		}
	}
	err := errors.Join(errs...)
	fmt.Println(err)
	_, ok := errors.AsType[*ErrValidation](err)
	fmt.Println(ok)
	// Output:
	// champ "age" invalide : pas un nombre
	// champ "age" invalide : négatif
	// true
}

func doitEtrePositif(n int) int {
	if n < 0 {
		panic(fmt.Sprintf("valeur négative : %d", n))
	}
	return n
}

func Example_panic() {
	defer func() {
		r := recover()
		fmt.Println("récupéré :", r)
	}()
	doitEtrePositif(-1)
	// Output: récupéré : valeur négative : -1
}
