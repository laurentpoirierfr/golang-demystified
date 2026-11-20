// Package ancien est écrit comme on écrivait Go il y a quelques années.
// go fix le modernise (voir outils_test.go).
package ancien

import (
	"errors"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Premier renvoie la première valeur, quelle qu'elle soit.
func Premier(valeurs []interface{}) interface{} {
	if len(valeurs) == 0 {
		return nil
	}
	return valeurs[0]
}

// Somme additionne les entiers de 0 à n-1.
func Somme(n int) int {
	s := 0
	for i := 0; i < n; i++ {
		s += i
	}
	return s
}

// Plus grand des deux.
func Plus(a, b int) int {
	m := a
	if b > m {
		m = b
	}
	return m
}

// Contient indique si mot est dans liste.
func Contient(liste []string, mot string) bool {
	for _, m := range liste {
		if m == mot {
			return true
		}
	}
	return false
}

// Trie les entiers.
func Trie(t []int) {
	sort.Slice(t, func(i, j int) bool { return t[i] < t[j] })
}

// Cle renvoie la clé de « cle=valeur ».
func Cle(s string) string {
	if i := strings.Index(s, "="); i >= 0 {
		return s[:i]
	}
	return s
}

// SansPrefixe retire « go: ».
func SansPrefixe(s string) (string, bool) {
	if strings.HasPrefix(s, "go:") {
		return strings.TrimPrefix(s, "go:"), true
	}
	return s, false
}

// Mots compte les mots non vides.
func Mots(texte string) int {
	n := 0
	for _, m := range strings.Split(texte, " ") {
		if m != "" {
			n++
		}
	}
	return n
}

// Parallele exécute les fonctions en parallèle.
func Parallele(fonctions []func()) {
	var wg sync.WaitGroup
	for _, f := range fonctions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}

// Chemin renvoie le chemin d'une erreur de fichier.
func Chemin(err error) string {
	var e *fs.PathError
	if errors.As(err, &e) {
		return e.Path
	}
	return ""
}

// NomDuType renvoie le nom du type int.
func NomDuType() string {
	return reflect.TypeOf(0).String()
}
