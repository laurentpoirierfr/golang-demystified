// Package fichiers contient les solutions des exercices du chapitre
// « Fichiers et flux : io, os, bufio ».
package fichiers

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Exercice 1 ★ : compter les lignes.

// CompteLignes compte les lignes non vides de r. Elle accepte tout
// io.Reader : un fichier, une chaîne, une réponse HTTP.
func CompteLignes(r io.Reader) (int, error) {
	n := 0
	s := bufio.NewScanner(r)
	for s.Scan() {
		if strings.TrimSpace(s.Text()) != "" {
			n++
		}
	}
	return n, s.Err()
}

// Exercice 2 ★★ : parcourir un système de fichiers.

// ParExtension renvoie les chemins des fichiers d'extension ext, dans
// l'ordre lexical. Elle reçoit un fs.FS : un répertoire (os.DirFS), des
// fichiers embarqués, ou fstest.MapFS dans les tests.
func ParExtension(fsys fs.FS, ext string) ([]string, error) {
	var chemins []string
	err := fs.WalkDir(fsys, ".", func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(chemin) == ext {
			chemins = append(chemins, chemin)
		}
		return nil
	})
	return chemins, err
}

// Exercice 3 ★★★ : écrire un fichier de façon atomique.

// EcritAtomique remplace le contenu d'un fichier sans risquer de le laisser
// à moitié écrit : elle écrit un fichier temporaire dans le même
// répertoire, puis le renomme, ce qui est atomique sur un même système de
// fichiers. En cas d'erreur, le fichier d'origine est intact.
func EcritAtomique(chemin string, donnees []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(chemin), ".tmp-"+filepath.Base(chemin)+"-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(tmp.Name()) // ne pas laisser de fichier temporaire
		}
	}()
	if _, err = tmp.Write(donnees); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil { // sur le disque avant le renommage
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), chemin)
}
