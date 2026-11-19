// Exemples du chapitre « Tour d'horizon ».
package divers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	randv2 "math/rand/v2"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"uuid"
)

func Example_uuid() {
	id := uuid.NewV7()                                // triable par date de création
	fmt.Println(len(id.String()), id.String()[14:15]) // le 15e caractère est la version

	u, err := uuid.Parse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	fmt.Println(u, err)

	_, err = uuid.Parse("pas-un-uuid")
	fmt.Println(err != nil)
	// Output:
	// 36 7
	// f47ac10b-58cc-4372-a567-0e02b2c3d479 <nil>
	// true
}

func Example_hasard() {
	jeton := rand.Text() // aléatoire cryptographique, 26 caractères base32
	fmt.Println(len(jeton))

	de := randv2.IntN(6) + 1
	fmt.Println(de >= 1 && de <= 6)

	r := randv2.New(randv2.NewPCG(1, 2)) // générateur déterministe
	fmt.Println(r.IntN(100), r.IntN(100), randv2.N(10*time.Second) < 10*time.Second)
	// Output:
	// 26
	// true
	// 76 61 true
}

func Example_empreinte() {
	somme := sha256.Sum256([]byte("gopher"))
	fmt.Println(hex.EncodeToString(somme[:8]))
	// Output: 9cc1ee455a3363ff
}

func Example_regexp() {
	email := regexp.MustCompile(`^([a-z0-9._%+-]+)@([a-z0-9.-]+\.[a-z]{2,})$`)
	fmt.Println(email.MatchString("gopher@go.dev"), email.MatchString("pas un email"))
	fmt.Println(email.FindStringSubmatch("alice@example.com"))

	date := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	fmt.Println(date.ReplaceAllString("sortie le 2026-08-12", "$3/$2/$1"))
	// Output:
	// true false
	// [alice@example.com alice example.com]
	// sortie le 12/08/2026
}

func Example_csv() {
	r := csv.NewReader(strings.NewReader("nom,age\nAlice,31\n\"Bob, junior\",27\n"))
	lignes, err := r.ReadAll()
	fmt.Println(len(lignes), lignes[2][0], err)

	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"ville", "habitants"})
	w.Write([]string{"Saint-Étienne, Loire", "170000"})
	w.Flush()
	// Output:
	// 3 Bob, junior <nil>
	// ville,habitants
	// "Saint-Étienne, Loire",170000
}

func Example_flag() {
	fs := flag.NewFlagSet("serveur", flag.ContinueOnError)
	port := fs.Int("port", 8080, "port d'écoute")
	verbeux := fs.Bool("v", false, "mode verbeux")
	delai := fs.Duration("delai", 5*time.Second, "délai d'expiration")

	fs.Parse([]string{"-port", "9090", "-v", "fichier.txt"})
	fmt.Println(*port, *verbeux, *delai, fs.Args())
	// Output: 9090 true 5s [fichier.txt]
}

func Example_onceValue() {
	appels := 0
	config := sync.OnceValue(func() map[string]string {
		appels++
		return map[string]string{"env": "prod"}
	})
	fmt.Println(config()["env"], config()["env"], appels)
	// Output: prod prod 1
}
