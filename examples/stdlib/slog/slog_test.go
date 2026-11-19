// Exemples du chapitre « Journalisation : log/slog ».
package slog

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// sansHeure retire l'horodatage pour que la sortie des exemples soit stable.
func sansHeure(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func Example_texte() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: sansHeure}))
	logger.Info("serveur démarré", "port", 8080, "tls", false)
	logger.Warn("disque presque plein", "libre", "2 Go")
	logger.Debug("invisible : niveau Info par défaut")
	// Output:
	// level=INFO msg="serveur démarré" port=8080 tls=false
	// level=WARN msg="disque presque plein" libre="2 Go"
}

func Example_json() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: sansHeure,
	}))
	logger.Debug("requête", "méthode", "GET", "durée", 42*time.Millisecond)
	logger.Error("échec", slog.String("table", "commandes"), slog.Int("tentatives", 3))
	// Output:
	// {"level":"DEBUG","msg":"requête","méthode":"GET","durée":42000000}
	// {"level":"ERROR","msg":"échec","table":"commandes","tentatives":3}
}

func Example_with() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: sansHeure}))
	reqLog := logger.With("requete_id", "a1b2", "utilisateur", 42)
	reqLog.Info("commande créée", slog.Group("commande", "id", 1001, "montant", 59.9))
	// Output: level=INFO msg="commande créée" requete_id=a1b2 utilisateur=42 commande.id=1001 commande.montant=59.9
}

type Secret string

func (Secret) LogValue() slog.Value { return slog.StringValue("***") }

func Example_logValuer() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: sansHeure}))
	logger.Info("connexion", "utilisateur", "alice", "mot_de_passe", Secret("hunter2"))
	// Output: level=INFO msg=connexion utilisateur=alice mot_de_passe=***
}

func Example_niveauDynamique() {
	var niveau slog.LevelVar // Info par défaut
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       &niveau,
		ReplaceAttr: sansHeure,
	}))
	logger.Debug("caché")
	niveau.Set(slog.LevelDebug)
	logger.Debug("visible")
	// Output: level=DEBUG msg=visible
}

func Example_multiHandler() {
	opts := &slog.HandlerOptions{ReplaceAttr: sansHeure}
	logger := slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, opts),
		slog.NewJSONHandler(os.Stdout, opts),
	))
	logger.InfoContext(context.Background(), "double")
	// Output:
	// level=INFO msg=double
	// {"level":"INFO","msg":"double"}
}
