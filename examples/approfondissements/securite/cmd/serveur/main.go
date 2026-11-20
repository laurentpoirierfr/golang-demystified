// Command serveur démarre l'API de notes en HTTPS, authentifiée par un
// fournisseur OpenID Connect.
//
// Configuration (variables d'environnement) :
//
//	OIDC_EMETTEUR       URL de l'émetteur des jetons (obligatoire)
//	OIDC_AUDIENCE       audience attendue dans les jetons (api-notes)
//	SECURITE_ADRESSE    adresse d'écoute (localhost:8443)
//	SECURITE_FICHIERS   répertoire des pièces jointes (fichiers)
//	TLS_CERT, TLS_CLE   certificat et clé du serveur (PEM) ; sans eux, un
//	                    certificat éphémère est créé, pour le développement
//	TLS_AC_CLIENTS      autorité des certificats clients : active mTLS
//	CLE_ADMIN           clé d'administration (ou CLE_ADMIN_FILE)
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/auth"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/internal/web"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/pki"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/secret"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv, os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func valeur(getenv func(string) string, nom, defaut string) string {
	if v := getenv(nom); v != "" {
		return v
	}
	return defaut
}

func run(ctx context.Context, getenv func(string) string, sortie io.Writer, pret chan<- string) error {
	log := slog.New(slog.NewJSONHandler(sortie, nil))

	emetteur := getenv("OIDC_EMETTEUR")
	if emetteur == "" {
		return errors.New("OIDC_EMETTEUR est obligatoire")
	}
	audience := valeur(getenv, "OIDC_AUDIENCE", "api-notes")
	cleAdmin, err := secret.Lit(getenv, "CLE_ADMIN")
	if err != nil {
		return err
	}
	// Journaliser la configuration est utile… et sans risque : le secret
	// s'affiche « [masqué] ».
	log.Info("configuration", "emetteur", emetteur, "audience", audience, "cle_admin", cleAdmin)

	// Le serveur refuse de démarrer si le fournisseur d'identité est
	// injoignable : mieux vaut un échec clair qu'une API qui refuse tout.
	verificateur, err := auth.NouveauVerificateur(ctx, emetteur, audience)
	if err != nil {
		return fmt.Errorf("fournisseur d'identité : %w", err)
	}

	fichiers, err := os.OpenRoot(valeur(getenv, "SECURITE_FICHIERS", "fichiers"))
	if err != nil {
		return fmt.Errorf("répertoire des fichiers : %w", err)
	}
	defer fichiers.Close()

	confTLS, err := configurationTLS(getenv, log)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           web.Nouveau(log, fichiers, cleAdmin).Routes(auth.Authentifie(verificateur, log)),
		TLSConfig:         confTLS,
		ReadHeaderTimeout: 5 * time.Second, // contre les clients trop lents (Slowloris)
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	ecoute, err := net.Listen("tcp", valeur(getenv, "SECURITE_ADRESSE", "localhost:8443"))
	if err != nil {
		return err
	}
	log.Info("serveur démarré", "adresse", ecoute.Addr().String(), "mtls", confTLS.ClientAuth == tls.RequireAndVerifyClientCert)
	if pret != nil {
		pret <- ecoute.Addr().String()
	}

	erreurs := make(chan error, 1)
	go func() { erreurs <- srv.ServeTLS(ecoute, "", "") }() // certificats dans TLSConfig

	select {
	case err := <-erreurs:
		return err
	case <-ctx.Done():
	}
	arret, annule := context.WithTimeout(context.Background(), 10*time.Second)
	defer annule()
	return srv.Shutdown(arret)
}

// configurationTLS charge le certificat du serveur et, si une autorité des
// clients est fournie, exige un certificat client (mTLS).
func configurationTLS(getenv func(string) string, log *slog.Logger) (*tls.Config, error) {
	conf := &tls.Config{
		// TLS 1.3 seulement : des suites de chiffrement toutes sûres, et un
		// échange de clés hybride post-quantique (X25519MLKEM768) par défaut
		// depuis Go 1.24. Pour des clients anciens, VersionTLS12 reste
		// acceptable : Go n'active que des suites sûres.
		MinVersion: tls.VersionTLS13,
	}

	var cert tls.Certificate
	var err error
	if fichierCert := getenv("TLS_CERT"); fichierCert != "" {
		cert, err = tls.LoadX509KeyPair(fichierCert, getenv("TLS_CLE"))
	} else {
		log.Warn("certificat éphémère auto-signé : à réserver au développement")
		cert, err = certificatEphemere()
	}
	if err != nil {
		return nil, fmt.Errorf("certificat du serveur : %w", err)
	}
	conf.Certificates = []tls.Certificate{cert}

	if fichierAC := getenv("TLS_AC_CLIENTS"); fichierAC != "" {
		pem, err := os.ReadFile(fichierAC)
		if err != nil {
			return nil, fmt.Errorf("autorité des clients : %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("autorité des clients : aucun certificat PEM")
		}
		conf.ClientCAs = pool
		conf.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return conf, nil
}

func certificatEphemere() (tls.Certificate, error) {
	ac, err := pki.NouvelleAC("Développement")
	if err != nil {
		return tls.Certificate{}, err
	}
	c, err := ac.Serveur("localhost", "127.0.0.1", "::1")
	if err != nil {
		return tls.Certificate{}, err
	}
	return c.TLS()
}
