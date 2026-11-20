// Package client illustre les options fonctionnelles : un constructeur qui
// reçoit ses paramètres obligatoires en arguments, et ses réglages
// facultatifs sous forme de fonctions.
//
//	c, err := client.Nouveau("https://api.exemple.org",
//		client.AvecDelai(5*time.Second),
//		client.AvecEssais(3),
//	)
//
// L'appel le plus simple reste simple (client.Nouveau(url)), chaque option se
// documente, se valide et s'ajoute sans casser les appels existants.
package client

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Client appelle une API HTTP.
type Client struct {
	base   *url.URL
	http   *http.Client
	essais int
	log    *slog.Logger
	agent  string
}

// Option règle un Client. Elle peut échouer : une valeur invalide est
// signalée à la construction, pas à la première requête.
type Option func(*Client) error

// AvecDelai fixe le délai maximal d'une requête (10 s par défaut).
func AvecDelai(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return fmt.Errorf("délai invalide : %v", d)
		}
		c.http.Timeout = d
		return nil
	}
}

// AvecEssais fixe le nombre de tentatives (1 par défaut).
func AvecEssais(n int) Option {
	return func(c *Client) error {
		if n < 1 || n > 10 {
			return fmt.Errorf("nombre d'essais invalide : %d (de 1 à 10)", n)
		}
		c.essais = n
		return nil
	}
}

// AvecJournal fixe le journal (aucun par défaut).
func AvecJournal(l *slog.Logger) Option {
	return func(c *Client) error {
		c.log = l
		return nil
	}
}

// AvecAgent fixe l'en-tête User-Agent.
func AvecAgent(agent string) Option {
	return func(c *Client) error {
		c.agent = agent
		return nil
	}
}

// Nouveau crée un client. Les valeurs par défaut sont posées d'abord, puis
// chaque option les modifie, dans l'ordre.
func Nouveau(base string, options ...Option) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("adresse invalide %q", base)
	}
	c := &Client{
		base:   u,
		http:   &http.Client{Timeout: 10 * time.Second},
		essais: 1,
		log:    slog.New(slog.DiscardHandler),
		agent:  "golang-demystified/1.0",
	}
	var errs []error
	for _, o := range options {
		errs = append(errs, o(c))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return c, nil
}

// Delai, Essais et Agent exposent la configuration (pour les tests).
func (c *Client) Delai() time.Duration { return c.http.Timeout }
func (c *Client) Essais() int          { return c.essais }
func (c *Client) Agent() string        { return c.agent }
