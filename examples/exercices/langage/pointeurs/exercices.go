// Package pointeurs contient les solutions des exercices du chapitre
// « Pointeurs et mémoire ».
package pointeurs

// Exercice 1 ★ : échanger deux variables.

// Echange échange les valeurs pointées par a et b.
func Echange(a, b *int) { *a, *b = *b, *a }

// Exercice 2 ★★ : une liste chaînée.

// Noeud est un maillon de liste chaînée.
type Noeud struct {
	Valeur  int
	Suivant *Noeud
}

// Liste construit une liste à partir de valeurs.
func Liste(valeurs ...int) *Noeud {
	var tete *Noeud
	for i := len(valeurs) - 1; i >= 0; i-- {
		tete = &Noeud{Valeur: valeurs[i], Suivant: tete}
	}
	return tete
}

// Valeurs renvoie les valeurs de la liste, dans l'ordre.
func (n *Noeud) Valeurs() []int {
	var r []int
	for ; n != nil; n = n.Suivant { // une méthode accepte un récepteur nil
		r = append(r, n.Valeur)
	}
	return r
}

// Inverse retourne la liste sur place, sans allouer, et renvoie la nouvelle
// tête.
func Inverse(tete *Noeud) *Noeud {
	var precedent *Noeud
	for tete != nil {
		tete.Suivant, precedent, tete = precedent, tete, tete.Suivant
	}
	return precedent
}

// Exercice 3 ★★★ : un arbre binaire de recherche.

// Arbre est un arbre binaire de recherche d'entiers. Un *Arbre nil est un
// arbre vide.
type Arbre struct {
	Valeur        int
	Gauche, Droit *Arbre
}

// Insere ajoute v, et renvoie la racine (nouvelle si l'arbre était vide).
func (a *Arbre) Insere(v int) *Arbre {
	if a == nil {
		return &Arbre{Valeur: v}
	}
	switch {
	case v < a.Valeur:
		a.Gauche = a.Gauche.Insere(v)
	case v > a.Valeur:
		a.Droit = a.Droit.Insere(v)
	}
	return a
}

// Contient indique si v est dans l'arbre.
func (a *Arbre) Contient(v int) bool {
	for a != nil {
		switch {
		case v < a.Valeur:
			a = a.Gauche
		case v > a.Valeur:
			a = a.Droit
		default:
			return true
		}
	}
	return false
}

// EnOrdre renvoie les valeurs dans l'ordre croissant.
func (a *Arbre) EnOrdre() []int {
	if a == nil {
		return nil
	}
	return append(append(a.Gauche.EnOrdre(), a.Valeur), a.Droit.EnOrdre()...)
}
