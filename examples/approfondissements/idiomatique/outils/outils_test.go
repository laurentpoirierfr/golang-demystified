// Package outils_test fait tourner go fix et go vet sur des exemples, pour
// que le chapitre cite leur vraie sortie.
package outils_test

import (
	"os/exec"
	"strings"
	"testing"
)

func lance(t *testing.T, args ...string) string {
	t.Helper()
	out, _ := exec.Command("go", args...).CombinedOutput() // un diagnostic fait échouer la commande
	return string(out)
}

// go fix -diff montre les modernisations sans les appliquer.
func TestGoFix(t *testing.T) {
	if testing.Short() {
		t.Skip("lance go fix")
	}
	diff := lance(t, "fix", "-diff", "./testdata/ancien")
	t.Logf("go fix -diff :\n%s", diff)
	for _, attendu := range []string{
		"+func Premier(valeurs []any) any {",   // any
		"+	for i := range n {",                 // rangeint
		"+	m := max(b, a)",                     // minmax : go fix garde l'ordre d'évaluation du code d'origine
		"+	return slices.Contains(liste, mot)", // slicescontains
		"+	slices.Sort(t)",                     // slicessort
		"strings.Cut(",                         // stringscut
		"strings.CutPrefix(",                   // stringscutprefix
		"strings.SplitSeq(",                    // stringsseq
		"+		wg.Go(func() {",                    // waitgroupgo
		"errors.AsType[*fs.PathError](err)",    // errorsastype
		"reflect.TypeFor[int]()",               // reflecttypefor
	} {
		if !strings.Contains(diff, attendu) {
			t.Errorf("modernisation absente : %q", attendu)
		}
	}
}

// go vet repère ce que le compilateur laisse passer.
func TestGoVet(t *testing.T) {
	if testing.Short() {
		t.Skip("lance go vet")
	}
	sortie := lance(t, "vet", "./testdata/pieges")
	t.Logf("go vet :\n%s", sortie)
	for _, attendu := range []string{
		"passes lock by value", // copylocks
		"fmt.Sprintf format %d has arg nom of wrong type string",               // printf
		"the cancel function returned by context.WithTimeout should be called", // lostcancel
		"result of fmt.Sprintf call not used",                                  // unusedresult
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("diagnostic absent : %q", attendu)
		}
	}
}
