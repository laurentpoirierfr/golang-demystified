package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func construit(t *testing.T) string {
	t.Helper()
	binaire := filepath.Join(t.TempDir(), "ordonnanceur")
	if out, err := exec.Command("go", "build", "-o", binaire, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return binaire
}

// lance exécute le programme avec un délai, et renvoie sa sortie et si le
// délai a expiré.
func lance(t *testing.T, binaire string, env []string, args ...string) (string, bool) {
	t.Helper()
	ctx, annule := context.WithTimeout(t.Context(), 3*time.Second)
	defer annule()
	cmd := exec.CommandContext(ctx, binaire, args...)
	cmd.Env = append(os.Environ(), env...)
	out, _ := cmd.Output() // tué par le délai : la sortie partielle suffit
	return string(out), errors.Is(ctx.Err(), context.DeadlineExceeded)
}

func TestPreemption(t *testing.T) {
	if testing.Short() {
		t.Skip("compile et lance le programme, dont une fois bloqué 3 s")
	}
	binaire := construit(t)

	// Avec la préemption asynchrone : main reprend la main.
	out, bloque := lance(t, binaire, nil, "-boucle")
	if bloque || !strings.HasPrefix(out, "main a repris") {
		t.Errorf("avec préemption : bloqué=%v, sortie %q", bloque, out)
	}

	// Sans elle, la boucle garde le seul fil pour toujours.
	out, bloque = lance(t, binaire, []string{"GODEBUG=asyncpreemptoff=1"}, "-boucle")
	if !bloque {
		t.Errorf("sans préemption, le programme aurait dû rester bloqué ; sortie %q", out)
	}
}

func TestGOMAXPROCS(t *testing.T) {
	if testing.Short() {
		t.Skip("compile et lance le programme")
	}
	out, _ := lance(t, construit(t), []string{"GOMAXPROCS=3"})
	if !strings.Contains(out, "GOMAXPROCS                    : 3") {
		t.Errorf("sortie : %s", out)
	}
}
