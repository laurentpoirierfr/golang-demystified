// Le test est dans un package externe (packages_test) : il ne voit que ce
// que le package exporte, comme un vrai utilisateur.
package packages_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/exercices/langage/packages"
)

func TestTemperature(t *testing.T) {
	temp, err := packages.DepuisCelsius(21.5)
	if err != nil || temp.Celsius() < 21.4999 || temp.Celsius() > 21.5001 {
		t.Errorf("%v, %v", temp.Celsius(), err)
	}
	if _, err := packages.DepuisCelsius(-300); err == nil {
		t.Error("-300 °C devrait être refusé")
	}
}

func TestRegistre(t *testing.T) {
	if !slices.Contains(packages.Formats(), "inverse") { // le registre survit à go test -count=2
		packages.Enregistre("inverse", func(s string) string {
			r := []rune(s)
			slices.Reverse(r)
			return string(r)
		})
	}
	if got := packages.Formats(); !slices.Equal(got, []string{"inverse", "majuscules", "minuscules"}) {
		t.Errorf("Formats = %v", got)
	}
	if s, err := packages.Formate("majuscules", "go"); err != nil || s != "GO" {
		t.Errorf("%q, %v", s, err)
	}
	if _, err := packages.Formate("absent", "go"); err == nil {
		t.Error("un format inconnu doit être une erreur")
	}
	defer func() {
		if recover() == nil {
			t.Error("un nom en double doit paniquer")
		}
	}()
	packages.Enregistre("majuscules", strings.ToUpper)
}

func TestVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("compile un programme")
	}
	sortie, err := exec.Command("go", "run", "-ldflags", "-X main.version=1.2.3", "./cmd/version").CombinedOutput()
	if err != nil || strings.TrimSpace(string(sortie)) != "version 1.2.3" {
		t.Errorf("%s, %v", sortie, err)
	}
}
