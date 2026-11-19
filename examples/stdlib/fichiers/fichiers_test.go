// Exemples du chapitre « Fichiers et flux : io, os, bufio ».
package fichiers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Example_lireEcrire() {
	dir, _ := os.MkdirTemp("", "exemple")
	defer os.RemoveAll(dir)
	chemin := filepath.Join(dir, "notes.txt")

	err := os.WriteFile(chemin, []byte("ligne 1\nligne 2\n"), 0o644)
	fmt.Println(err)

	data, err := os.ReadFile(chemin)
	fmt.Printf("%q %v\n", data, err)

	_, err = os.ReadFile(filepath.Join(dir, "absent.txt"))
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	// Output:
	// <nil>
	// "ligne 1\nligne 2\n" <nil>
	// true
}

func Example_scanner() {
	entree := strings.NewReader("alpha\nbeta\n\ngamma\n")
	sc := bufio.NewScanner(entree)
	n := 0
	for sc.Scan() {
		if ligne := sc.Text(); ligne != "" {
			n++
			fmt.Println(n, ligne)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Println("erreur :", err)
	}
	// Output:
	// 1 alpha
	// 2 beta
	// 3 gamma
}

func Example_writer() {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for i := range 3 {
		fmt.Fprintf(w, "ligne %d\n", i)
	}
	// Output:
	// ligne 0
	// ligne 1
	// ligne 2
}

func Example_copy() {
	var dst bytes.Buffer
	n, err := io.Copy(&dst, strings.NewReader("contenu à copier"))
	fmt.Println(n, err, dst.String())
	// Output: 17 <nil> contenu à copier
}

func Example_composition() {
	// Compresse, puis calcule l'empreinte du résultat, en un seul passage.
	h := sha256.New()
	var compresse bytes.Buffer
	gz := gzip.NewWriter(io.MultiWriter(&compresse, h))
	io.Copy(gz, strings.NewReader(strings.Repeat("gopher ", 1000)))
	gz.Close()
	fmt.Println(compresse.Len() < 100, len(h.Sum(nil)))

	// Décompresse en lisant au plus 13 octets.
	gr, _ := gzip.NewReader(&compresse)
	debut, _ := io.ReadAll(io.LimitReader(gr, 13))
	fmt.Printf("%q\n", debut)
	// Output:
	// true 32
	// "gopher gopher"
}

func Example_walkDir() {
	dir, _ := os.MkdirTemp("", "arbre")
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755)
	os.WriteFile(filepath.Join(dir, "a", "un.go"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "a", "b", "deux.go"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "lisezmoi.md"), nil, 0o644)

	fichiers, _ := fs.Glob(os.DirFS(dir), "a/*.go")
	fmt.Println(fichiers)

	filepath.WalkDir(dir, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(chemin) == ".go" {
			rel, _ := filepath.Rel(dir, chemin)
			fmt.Println(filepath.ToSlash(rel))
		}
		return nil
	})
	// Output:
	// [a/un.go]
	// a/b/deux.go
	// a/un.go
}

func Example_root() {
	dir, _ := os.MkdirTemp("", "racine")
	defer os.RemoveAll(dir)

	root, err := os.OpenRoot(dir)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer root.Close()

	f, _ := root.Create("autorise.txt")
	f.Close()
	_, err = root.Open("../../etc/passwd")
	fmt.Println(err != nil)
	// Output: true
}

//go:embed testdata/bienvenue.txt
var bienvenue string

//go:embed testdata
var donnees embed.FS

func Example_embed() {
	fmt.Print(bienvenue)
	entrees, _ := fs.ReadDir(donnees, "testdata")
	for _, e := range entrees {
		fmt.Println(e.Name())
	}
	// Output:
	// Bienvenue, gopher !
	// bienvenue.txt
	// config.json
}

func TestFichierTemporaire(t *testing.T) {
	dir := t.TempDir() // supprimé automatiquement à la fin du test
	chemin := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(chemin, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
}
