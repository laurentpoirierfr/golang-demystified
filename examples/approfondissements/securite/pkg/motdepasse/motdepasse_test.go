package motdepasse_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/securite/pkg/motdepasse"
)

func TestHacheEtVerifie(t *testing.T) {
	empreinte := motdepasse.Hache("correct horse battery staple")
	if !strings.HasPrefix(empreinte, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Errorf("format : %s", empreinte)
	}
	if ok, err := motdepasse.Verifie("correct horse battery staple", empreinte); !ok || err != nil {
		t.Errorf("bon mot de passe refusé : %v", err)
	}
	if ok, _ := motdepasse.Verifie("Correct horse battery staple", empreinte); ok {
		t.Error("mauvais mot de passe accepté")
	}
}

func TestSelAleatoire(t *testing.T) {
	a, b := motdepasse.Hache("1234"), motdepasse.Hache("1234")
	if a == b {
		t.Error("deux empreintes identiques : le sel n'est pas aléatoire")
	}
}

func TestEmpreintesInvalides(t *testing.T) {
	for _, e := range []string{
		"",
		"5f4dcc3b5aa765d61d8327deb882cf99", // un MD5 : jamais ça
		"$argon2i$v=19$m=65536,t=3,p=4$c2Vs$Y2xl",
		"$argon2id$v=19$m=99999999,t=3,p=4$c2VsLXNlbC1zZWw$Y2xlLWNsZS1jbGUtY2xlLWNsZQ", // mémoire démesurée
		"$argon2id$v=19$m=65536,t=3,p=4$!!!$Y2xl",
	} {
		if _, err := motdepasse.Verifie("x", e); !errors.Is(err, motdepasse.ErrFormat) {
			t.Errorf("%q : err = %v", e, err)
		}
	}
}

func TestARenouveler(t *testing.T) {
	faible := motdepasse.Parametres{Memoire: 19 * 1024, Iterations: 2, Parallelism: 1, TailleSel: 16, TailleCle: 32}
	ancienne := faible.Hache("secret")
	if !motdepasse.Defaut.ARenouveler(ancienne) {
		t.Error("une empreinte plus faible devrait être renouvelée")
	}
	if motdepasse.Defaut.ARenouveler(motdepasse.Hache("secret")) {
		t.Error("une empreinte à jour ne devrait pas être renouvelée")
	}
	// L'ancienne empreinte reste vérifiable : ses paramètres l'accompagnent.
	if ok, _ := motdepasse.Verifie("secret", ancienne); !ok {
		t.Error("ancienne empreinte refusée")
	}
}

func BenchmarkHache(b *testing.B) {
	for b.Loop() {
		motdepasse.Hache("correct horse battery staple")
	}
}
