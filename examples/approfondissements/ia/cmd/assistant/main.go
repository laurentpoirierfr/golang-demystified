// Command assistant gère une liste de tâches en conversant avec un modèle de
// langage : Claude (par l'API d'Anthropic) ou un modèle local (par Ollama).
//
//	assistant                                  # Claude, ANTHROPIC_API_KEY ou ant auth login
//	assistant -fournisseur ollama -modele qwen3:1.7b
//	assistant "Ajoute « Acheter du pain »"     # une seule demande
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/claude"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/internal/ollama"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/ia/pkg/outils"
)

// Assistant est ce qu'offrent les deux agents.
type Assistant interface {
	Demande(ctx context.Context, question string, ecrit func(string)) (string, error)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, entree io.Reader, sortie io.Writer) error {
	drapeaux := flag.NewFlagSet("assistant", flag.ContinueOnError)
	fournisseur := drapeaux.String("fournisseur", "claude", "claude ou ollama")
	modele := drapeaux.String("modele", "", "modèle (par défaut : "+claude.Modele+" pour Claude)")
	urlOllama := drapeaux.String("ollama", "http://localhost:11434", "adresse d'Ollama")
	if err := drapeaux.Parse(args); err != nil {
		return err
	}

	boite, err := outils.Boite(&outils.Taches{})
	if err != nil {
		return err
	}
	// Chaque appel d'outil est affiché : on voit ce que fait le modèle.
	observe := func(a outils.Appel) {
		etat := a.Resultat
		if a.Err != nil {
			etat = "erreur : " + a.Err.Error()
		}
		fmt.Fprintf(sortie, "  ↳ %s %s → %s\n", a.Outil, a.Entree, strings.TrimSpace(etat))
	}

	var assistant Assistant
	switch *fournisseur {
	case "claude":
		a := claude.Nouveau(anthropic.NewClient(), boite)
		if *modele != "" {
			a.Modele = *modele
		}
		a.Observe = observe
		assistant = a
	case "ollama":
		if *modele == "" {
			return errors.New("-modele est obligatoire avec ollama (par exemple qwen3:1.7b)")
		}
		a := ollama.Nouveau(*urlOllama, *modele, boite)
		a.Observe = observe
		assistant = a
	default:
		return fmt.Errorf("fournisseur inconnu %q (claude ou ollama)", *fournisseur)
	}

	demande := func(question string) error {
		_, err := assistant.Demande(ctx, question, func(s string) { fmt.Fprint(sortie, s) })
		fmt.Fprintln(sortie)
		return err
	}
	// Une demande passée en argument, ou une conversation.
	if drapeaux.NArg() > 0 {
		return demande(strings.Join(drapeaux.Args(), " "))
	}
	lignes := bufio.NewScanner(entree)
	for fmt.Fprint(sortie, "> "); lignes.Scan(); fmt.Fprint(sortie, "> ") {
		if q := strings.TrimSpace(lignes.Text()); q != "" {
			if err := demande(q); err != nil {
				return err
			}
		}
	}
	return lignes.Err()
}
