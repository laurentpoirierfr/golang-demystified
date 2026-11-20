// Package motdepasse stocke les mots de passe sous forme d'empreintes
// argon2id, l'algorithme recommandé par l'OWASP et la RFC 9106.
//
// Une empreinte se présente au format PHC, qui contient tout ce qu'il faut
// pour la vérifier :
//
//	$argon2id$v=19$m=65536,t=3,p=4$<sel>$<empreinte>
//
// Les paramètres voyagent avec l'empreinte : on peut les renforcer plus tard
// sans invalider les mots de passe existants (voir ARenouveler).
package motdepasse

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parametres règle le coût du calcul. Plus il est élevé, plus une attaque par
// force brute sur une base volée est lente, mais plus chaque connexion coûte.
type Parametres struct {
	Memoire     uint32 // en Kio
	Iterations  uint32
	Parallelism uint8
	TailleSel   uint32
	TailleCle   uint32
}

// Defaut suit la deuxième recommandation de la RFC 9106 : 64 Mio de mémoire,
// 3 passes, 4 fils.
var Defaut = Parametres{Memoire: 64 * 1024, Iterations: 3, Parallelism: 4, TailleSel: 16, TailleCle: 32}

// ErrFormat signale une empreinte illisible.
var ErrFormat = errors.New("empreinte au format invalide")

var b64 = base64.RawStdEncoding

// Hache calcule l'empreinte d'un mot de passe, avec un sel aléatoire : deux
// utilisateurs au même mot de passe ont des empreintes différentes.
func (p Parametres) Hache(motDePasse string) string {
	sel := make([]byte, p.TailleSel)
	rand.Read(sel) // ne renvoie jamais d'erreur depuis Go 1.24
	cle := argon2.IDKey([]byte(motDePasse), sel, p.Iterations, p.Memoire, p.Parallelism, p.TailleCle)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memoire, p.Iterations, p.Parallelism, b64.EncodeToString(sel), b64.EncodeToString(cle))
}

// Hache calcule l'empreinte avec les paramètres par défaut.
func Hache(motDePasse string) string { return Defaut.Hache(motDePasse) }

// Verifie indique si le mot de passe correspond à l'empreinte.
func Verifie(motDePasse, empreinte string) (bool, error) {
	p, sel, cle, err := decode(empreinte)
	if err != nil {
		return false, err
	}
	calculee := argon2.IDKey([]byte(motDePasse), sel, p.Iterations, p.Memoire, p.Parallelism, uint32(len(cle)))
	// Comparaison en temps constant.
	return subtle.ConstantTimeCompare(calculee, cle) == 1, nil
}

// ARenouveler indique si l'empreinte a été calculée avec des paramètres plus
// faibles que p : après une connexion réussie, on recalcule alors l'empreinte
// avec les nouveaux paramètres.
func (p Parametres) ARenouveler(empreinte string) bool {
	q, _, _, err := decode(empreinte)
	return err != nil || q.Memoire < p.Memoire || q.Iterations < p.Iterations || q.Parallelism < p.Parallelism
}

func decode(empreinte string) (p Parametres, sel, cle []byte, err error) {
	morceaux := strings.Split(empreinte, "$")
	if len(morceaux) != 6 || morceaux[1] != "argon2id" {
		return p, nil, nil, ErrFormat
	}
	var version int
	if _, err := fmt.Sscanf(morceaux[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, ErrFormat
	}
	if _, err := fmt.Sscanf(morceaux[3], "m=%d,t=%d,p=%d", &p.Memoire, &p.Iterations, &p.Parallelism); err != nil {
		return p, nil, nil, ErrFormat
	}
	// Une empreinte venue de la base pourrait avoir été altérée : on borne les
	// paramètres pour qu'elle ne puisse pas épuiser la mémoire du serveur.
	if p.Memoire == 0 || p.Memoire > 1024*1024 || p.Iterations == 0 || p.Iterations > 64 || p.Parallelism == 0 {
		return p, nil, nil, ErrFormat
	}
	if sel, err = b64.DecodeString(morceaux[4]); err != nil {
		return p, nil, nil, ErrFormat
	}
	if cle, err = b64.DecodeString(morceaux[5]); err != nil || len(cle) < 16 {
		return p, nil, nil, ErrFormat
	}
	return p, sel, cle, nil
}
