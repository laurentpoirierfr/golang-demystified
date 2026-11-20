// Package slog contient les solutions des exercices du chapitre
// « Journalisation : log/slog ».
package slog

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// Exercice 1 ★ : un journal JSON.

// Nouveau renvoie un journal JSON écrit dans w, qui ignore les messages sous
// le niveau donné, et ajoute à chaque message l'attribut service.
func Nouveau(w io.Writer, niveau slog.Level, service string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: niveau})).With("service", service)
}

// Exercice 2 ★★ : masquer un secret.

// Secret est une valeur qui ne doit jamais apparaître dans un journal.
type Secret string

// LogValue satisfait slog.LogValuer : slog l'appelle au lieu d'afficher la
// valeur, quel que soit le handler.
func (Secret) LogValue() slog.Value { return slog.StringValue("***") }

// Exercice 3 ★★★ : un handler qui compte.

// Compteur compte les messages par niveau, puis les transmet à un autre
// handler. Les compteurs sont partagés par les handlers dérivés (With,
// WithGroup).
type Compteur struct {
	suivant slog.Handler
	etat    *etat
}

type etat struct {
	mu        sync.Mutex
	parNiveau map[slog.Level]int
}

// NouveauCompteur enveloppe un handler.
func NouveauCompteur(suivant slog.Handler) *Compteur {
	return &Compteur{suivant, &etat{parNiveau: map[slog.Level]int{}}}
}

func (c *Compteur) Enabled(ctx context.Context, l slog.Level) bool {
	return c.suivant.Enabled(ctx, l)
}

func (c *Compteur) Handle(ctx context.Context, r slog.Record) error {
	c.etat.mu.Lock()
	c.etat.parNiveau[r.Level]++
	c.etat.mu.Unlock()
	return c.suivant.Handle(ctx, r)
}

func (c *Compteur) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Compteur{c.suivant.WithAttrs(attrs), c.etat}
}

func (c *Compteur) WithGroup(nom string) slog.Handler {
	return &Compteur{c.suivant.WithGroup(nom), c.etat}
}

// Nombre renvoie le nombre de messages d'un niveau.
func (c *Compteur) Nombre(l slog.Level) int {
	c.etat.mu.Lock()
	defer c.etat.mu.Unlock()
	return c.etat.parNiveau[l]
}
