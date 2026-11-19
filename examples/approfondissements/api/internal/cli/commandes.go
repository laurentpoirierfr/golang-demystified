package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/api/pkg/client"
)

func jsonDecode(b []byte, v any) error { return json.Unmarshal(b, v) }

func (a *app) commandeListe() *cobra.Command {
	var faites, aFaire bool
	cmd := &cobra.Command{
		Use:     "liste",
		Aliases: []string{"ls"},
		Short:   "Liste les tâches",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			r, err := c.ListeTachesWithResponse(cmd.Context())
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			taches := []client.Tache{} // une liste vide s'affiche en [], pas en null
			for _, t := range *r.JSON200 {
				if (faites && !t.Faite) || (aFaire && t.Faite) {
					continue
				}
				taches = append(taches, t)
			}
			return a.affiche(taches)
		},
	}
	cmd.Flags().BoolVar(&faites, "faites", false, "seulement les tâches faites")
	cmd.Flags().BoolVar(&aFaire, "a-faire", false, "seulement les tâches à faire")
	cmd.MarkFlagsMutuallyExclusive("faites", "a-faire")
	return cmd
}

func (a *app) commandeAjoute() *cobra.Command {
	return &cobra.Command{
		Use:     "ajoute <titre>…",
		Short:   "Ajoute une tâche",
		Example: `  taches-cli ajoute Relire le chapitre sur viper`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			r, err := c.CreeTacheWithResponse(cmd.Context(),
				client.NouvelleTache{Titre: strings.Join(args, " ")})
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tâche %d créée : %s\n", r.JSON201.Id, r.JSON201.Titre)
			return nil
		},
	}
}

func (a *app) commandeFait() *cobra.Command {
	var annule bool
	cmd := &cobra.Command{
		Use:               "fait <id>",
		Short:             "Marque une tâche comme faite",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: a.completeIdentifiants,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := identifiant(args[0])
			if err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			r, err := c.ModifieTacheWithResponse(cmd.Context(), id, client.ModificationTache{Faite: !annule})
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			etat := "faite"
			if annule {
				etat = "à faire"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tâche %d %s : %s\n", id, etat, r.JSON200.Titre)
			return nil
		},
	}
	cmd.Flags().BoolVar(&annule, "annule", false, "remet la tâche à faire")
	return cmd
}

func (a *app) commandeSupprime() *cobra.Command {
	return &cobra.Command{
		Use:               "supprime <id>",
		Aliases:           []string{"rm"},
		Short:             "Supprime une tâche",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: a.completeIdentifiants,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := identifiant(args[0])
			if err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			r, err := c.SupprimeTacheWithResponse(cmd.Context(), id)
			if err != nil {
				return err
			}
			if err := erreurAPI(r.HTTPResponse, r.Body); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Tâche %d supprimée\n", id)
			return nil
		},
	}
}

// completeIdentifiants propose, pendant la complétion du shell, les
// identifiants des tâches avec leur titre en description.
func (a *app) completeIdentifiants(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, err := a.client()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	r, err := c.ListeTachesWithResponse(cmd.Context())
	if err != nil || r.JSON200 == nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var propositions []string
	for _, t := range *r.JSON200 {
		propositions = append(propositions, fmt.Sprintf("%d\t%s", t.Id, t.Titre))
	}
	return propositions, cobra.ShellCompDirectiveNoFileComp
}

func identifiant(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("identifiant invalide : %q", s)
	}
	return id, nil
}

// affiche écrit les tâches dans le format demandé par --format.
func (a *app) affiche(taches []client.Tache) error {
	switch format := a.v.GetString("format"); format {
	case "json":
		enc := json.NewEncoder(a.sortie)
		enc.SetIndent("", "  ")
		return enc.Encode(taches)
	case "table":
		w := tabwriter.NewWriter(a.sortie, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tFAITE\tTITRE")
		for _, t := range taches {
			coche := " "
			if t.Faite {
				coche = "x"
			}
			fmt.Fprintf(w, "%d\t[%s]\t%s\n", t.Id, coche, t.Titre)
		}
		return w.Flush()
	default:
		return fmt.Errorf("format inconnu : %q (table ou json)", format)
	}
}
