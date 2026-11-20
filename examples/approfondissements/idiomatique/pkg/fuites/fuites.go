// Package fuites montre une fuite de goroutines classique, et sa correction.
//
// La fonction interroge plusieurs sources en parallèle et garde la première
// réponse. Dans la version fautive, les sources plus lentes envoient leur
// réponse dans un canal que plus personne ne lit : elles restent bloquées
// pour toujours, avec leur mémoire.
package fuites

import "context"

// Source répond à une question.
type Source func(ctx context.Context) string

// PremierFuyant renvoie la première réponse. Il fuit : le canal n'a pas de
// tampon, et les sources perdantes bloquent sur leur envoi.
func PremierFuyant(ctx context.Context, sources ...Source) string {
	reponses := make(chan string) // sans tampon
	for _, s := range sources {
		go func() { reponses <- s(ctx) }()
	}
	return <-reponses
}

// Premier renvoie la première réponse, sans fuite : le canal a une place par
// source, chaque envoi réussit même si personne ne le lit, et le contexte
// annulé prévient les sources que leur réponse n'intéresse plus personne.
func Premier(ctx context.Context, sources ...Source) string {
	ctx, annule := context.WithCancel(ctx)
	defer annule()
	reponses := make(chan string, len(sources))
	for _, s := range sources {
		go func() { reponses <- s(ctx) }()
	}
	return <-reponses
}
