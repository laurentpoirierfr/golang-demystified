---
title: "Sécurité : authentification, TLS et secrets"
linkTitle: "8. Sécurité"
weight: 80
---

Go a la réputation d'un langage sûr : pas de dépassement de tampon, une
bibliothèque `crypto` maintenue par des spécialistes, des valeurs par défaut
prudentes. Mais un service se compromet rarement par le langage. Il se
compromet par un jeton mal vérifié, un chemin de fichier venu du client, un
secret écrit dans un journal ou un corps de requête sans limite de taille.

Ce chapitre construit une petite API de notes, et la protège couche par couche :

| Menace | Protection | Outil |
|--------|------------|-------|
| Usurpation d'identité | jetons JWT d'un fournisseur OpenID Connect | `go-oidc` |
| Accès non autorisé | portées (scopes) et propriété des données | middleware |
| Écoute, falsification | TLS 1.3, et mTLS entre services | `crypto/tls` |
| Fuite de secrets | type `Secret` masqué, fichiers montés | `slog.LogValuer` |
| Mots de passe volés | empreintes argon2id | `x/crypto/argon2` |
| Entrées malveillantes | liste blanche, limites de taille | `encoding/json`, `MaxBytesReader` |
| Traversée de répertoires | accès confiné | `os.Root` (Go 1.24) |
| CSRF | origine des requêtes | `http.CrossOriginProtection` (Go 1.25) |
| Dépendances vulnérables | analyse du code appelé | `govulncheck` |

Le code est dans [`examples/approfondissements/securite`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/securite) :

```
securite/
├── cmd/serveur/          le programme : HTTPS, configuration, assemblage
├── internal/auth/        vérification des jetons, portées
├── internal/oidctest/    un faux fournisseur d'identité pour les tests
├── internal/web/         l'API : validation, fichiers, en-têtes
├── pkg/secret/           un type qui ne fuit jamais
├── pkg/motdepasse/       empreintes argon2id
├── pkg/pki/              une autorité de certification de test
└── pkg/signature/        signatures post-quantiques (Go 1.27)
```

## Authentifier : OpenID Connect et JWT

Une application ne devrait pas gérer elle-même les mots de passe de ses
utilisateurs. Elle délègue l'authentification à un **fournisseur d'identité**
([Keycloak](https://www.keycloak.org), [Dex](https://dexidp.io),
[Zitadel](https://zitadel.com), Auth0, Entra ID, Google…), qui délivre des
**jetons** signés. L'API n'a plus qu'à vérifier ces jetons, sans jamais voir un
mot de passe.

Ces jetons sont des **JWT** (JSON Web Tokens) : trois parties en base64,
séparées par des points (en-tête, revendications, signature).

```json
{"sub":"facturation","aud":"api-notes","scope":"notes:lire notes:ecrire",
 "iss":"http://localhost:18080/default","exp":1795122597,"iat":1795118997}
```

Vérifier un jeton, c'est contrôler **cinq** choses, et en oublier une suffit
à ouvrir une brèche :

1. la **signature**, avec la clé publique du fournisseur ;
2. l'**algorithme**, imposé par le serveur et non choisi par le jeton ;
3. l'**émetteur** (`iss`), le fournisseur attendu ;
4. l'**audience** (`aud`), cette API et pas une autre ;
5. l'**expiration** (`exp`).

[go-oidc](https://github.com/coreos/go-oidc) fait tout cela. Il lit la
configuration du fournisseur à l'adresse standard
`/.well-known/openid-configuration`, télécharge ses clés publiques (JWKS), les
garde en cache et les recharge quand le fournisseur en change :

```go
func NouveauVerificateur(ctx context.Context, emetteur, audience string) (*oidc.IDTokenVerifier, error) {
	fournisseur, err := oidc.NewProvider(ctx, emetteur)
	if err != nil {
		return nil, err
	}
	return fournisseur.Verifier(&oidc.Config{ClientID: audience}), nil
}
```

Le middleware extrait le jeton de l'en-tête `Authorization`, le vérifie, et
place l'identité de l'appelant dans le contexte :

```go
func Authentifie(v Verificateur, log *slog.Logger) func(http.Handler) http.Handler {
	return func(suivant http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			brut, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || brut == "" {
				refuse(w, `Bearer`)
				return
			}
			jeton, err := v.Verify(r.Context(), brut)
			if err != nil {
				// La raison est journalisée, jamais le jeton ; le client, lui,
				// n'apprend rien qui l'aiderait à forger un jeton.
				log.WarnContext(r.Context(), "jeton refusé", "raison", err.Error())
				refuse(w, `Bearer error="invalid_token"`)
				return
			}
			var revendications struct {
				Scope string `json:"scope"`
			}
			if err := jeton.Claims(&revendications); err != nil {
				refuse(w, `Bearer error="invalid_token"`)
				return
			}
			id := Identite{Sujet: jeton.Subject, Portees: strings.Fields(revendications.Scope)}
			suivant.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), cleIdentite{}, id)))
		})
	}
}
```

> [!CAUTION]
> N'écrivez pas votre propre vérification de JWT. L'attaque classique,
> `"alg": "none"`, présente un jeton **sans signature** à un vérificateur qui
> lit l'algorithme dans le jeton lui-même. Une autre consiste à signer un jeton
> avec la clé *publique* du serveur, utilisée comme secret HMAC. go-oidc
> n'accepte que les algorithmes annoncés par le fournisseur. Les tests de ce
> chapitre essaient les deux attaques.

### Autoriser : portées et propriété

L'**authentification** répond à « qui êtes-vous ? » (statut 401).
L'**autorisation** répond à « en avez-vous le droit ? » (statut 403) :

```go
prive.Handle("GET /notes", auth.Exige("notes:lire", http.HandlerFunc(s.listeNotes)))
prive.Handle("POST /notes", auth.Exige("notes:ecrire", http.HandlerFunc(s.creeNote)))
```

Mais une portée ne suffit pas : `notes:lire` autorise à lire **ses** notes,
pas celles des autres. L'autorisation porte aussi sur les **données** :

```go
for _, n := range s.notes {
	if n.Auteur == id.Sujet {
		notes = append(notes, n)
	}
}
```

Oublier ce filtre est la première vulnérabilité du classement de l'OWASP
pour les API (« Broken Object Level Authorization ») : un client change
l'identifiant dans l'URL, et lit les données d'un autre.

### Tester sans fournisseur réel

Le package `internal/oidctest` simule un fournisseur : un `httptest.Server`
publie la configuration et les clés publiques, et des méthodes signent des
jetons à la demande, y compris des jetons défectueux :

```go
emetteur := oidctest.Nouveau(t)
v, _ := auth.NouveauVerificateur(t.Context(), emetteur.URL, "api-notes")

valide := emetteur.Jeton(t, "alice", "api-notes", "notes:lire notes:ecrire")
expire := emetteur.Signe(t, oidctest.Revendications{
	Sujet: "alice", Audience: "api-notes", Expire: time.Now().Add(-time.Minute),
}, nil)
autreCle := emetteur.Signe(t, oidctest.Revendications{…}, cleInconnue)
```

Un tableau de tests couvre chaque cas : jeton valide, absent, d'un autre
schéma, illisible, expiré, d'une autre audience, d'un autre émetteur, signé par
une autre clé, `alg: none`, portée insuffisante. Chaque cas vérifie le statut,
l'en-tête `WWW-Authenticate` de la RFC 6750, la raison journalisée, et que le
jeton n'apparaît **jamais** dans le journal.

### Essayer avec un vrai fournisseur

[mock-oauth2-server](https://github.com/navikt/mock-oauth2-server) est un vrai
serveur OAuth2/OIDC, conçu pour le développement. On le configure comme un
Keycloak : l'audience `api-notes`, et les portées dans `scope`.

```bash
docker run -d --name oidc -p 18080:8080 -e JSON_CONFIG='{
  "interactiveLogin": false,
  "tokenCallbacks": [{"issuerId": "default", "requestMappings": [
    {"requestParam": "client_id", "match": "facturation",
     "claims": {"sub": "facturation", "aud": ["api-notes"], "scope": "notes:lire notes:ecrire"}},
    {"requestParam": "client_id", "match": "lecteur",
     "claims": {"sub": "lecteur", "aud": ["api-notes"], "scope": "notes:lire"}}
  ]}]}' ghcr.io/navikt/mock-oauth2-server:6.1.5

OIDC_EMETTEUR=http://localhost:18080/default go run ./cmd/serveur
```

Un service obtient un jeton par le flux `client_credentials`, puis appelle
l'API :

```console
$ JETON=$(curl -s localhost:18080/default/token -d grant_type=client_credentials \
    -d client_id=facturation -d client_secret=x -d scope=notes | jq -r .access_token)

$ curl -k -H "Authorization: Bearer $JETON" https://localhost:8443/moi
{"portees":["notes:lire","notes:ecrire"],"sujet":"facturation"}

$ curl -k -H "Authorization: Bearer $JETON_LECTEUR" -d '{"titre":"x"}' https://localhost:8443/notes
autorisation insuffisante                                          ← 403
```

> [!NOTE]
> En essayant de falsifier un jeton, on remplace volontiers son dernier
> caractère. Le premier essai de ce chapitre a ainsi obtenu un « 200 »
> inquiétant… parce que le jeton se terminait déjà par la lettre choisie : le
> jeton n'avait pas changé. Toute modification réelle de la signature ou du
> contenu (une portée `admin` ajoutée, par exemple) est bien refusée. Un test
> de sécurité doit d'abord vérifier qu'il a vraiment modifié ce qu'il croit
> modifier.

## Chiffrer : TLS

### Le serveur

Go fournit une implémentation complète de TLS, sans OpenSSL. Ses valeurs par
défaut sont sûres : seules des suites de chiffrement robustes, et, depuis
Go 1.24, un **échange de clés hybride post-quantique** (X25519MLKEM768), qui
protège dès aujourd'hui contre le déchiffrement futur de communications
enregistrées (« récolter maintenant, déchiffrer plus tard »).

```go
conf := &tls.Config{
	// TLS 1.3 seulement : des suites de chiffrement toutes sûres, et un
	// échange de clés hybride post-quantique par défaut depuis Go 1.24.
	MinVersion:   tls.VersionTLS13,
	Certificates: []tls.Certificate{cert},
}
srv := &http.Server{
	Handler:           handler,
	TLSConfig:         conf,
	ReadHeaderTimeout: 5 * time.Second, // contre les clients trop lents (Slowloris)
	ReadTimeout:       10 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       2 * time.Minute,
	MaxHeaderBytes:    16 << 10,
}
srv.ServeTLS(ecoute, "", "") // certificats dans TLSConfig
```

Les **délais** font partie de la sécurité : sans `ReadHeaderTimeout`, un
client qui envoie ses en-têtes un octet par minute occupe une connexion
indéfiniment, et quelques milliers de tels clients épuisent le serveur.

La négociation observée avec `openssl s_client` :

```console
$ openssl s_client -connect localhost:8443
Protocol: TLSv1.3
Negotiated TLS1.3 group: X25519MLKEM768

$ curl -k --tls-max 1.2 https://localhost:8443/sante
curl: (35) … protocol version
```

Et le journal du serveur garde la trace du client refusé :

```json
{"level":"WARN","msg":"http: TLS handshake error from 127.0.0.1:48158: tls: client offered only unsupported versions: [303 302 301]"}
```

### mTLS : authentifier les services entre eux

Dans le TLS ordinaire, seul le serveur présente un certificat. Avec le **TLS
mutuel** (mTLS), le client en présente un aussi : le serveur sait quel service
l'appelle avant même la première requête. C'est ce que font les maillages de
services comme Istio ou Linkerd.

```go
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(pemAC)
conf.ClientCAs = pool
conf.ClientAuth = tls.RequireAndVerifyClientCert
```

Le certificat du client est ensuite lisible dans la requête :

```go
if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
	reponse["certificat"] = r.TLS.PeerCertificates[0].Subject.CommonName
}
```

### Une autorité de certification pour les tests

Le package `pkg/pki` crée en quelques lignes une autorité, des certificats de
serveur et des certificats de clients (ECDSA P-256), avec la bibliothèque
standard seule :

```go
ac, _ := pki.NouvelleAC("AC de test")
serveur, _ := ac.Serveur("localhost", "127.0.0.1")
client, _ := ac.Client("facturation")

c := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
	RootCAs:      ac.Pool(), // cette autorité, et elle seule
	Certificates: []tls.Certificate{certClient},
}}}
```

Les tests vérifient qu'un client qui ne connaît pas l'autorité refuse le
serveur, que TLS 1.2 est refusé, que l'échange de clés est bien
`X25519MLKEM768`, qu'un client sans certificat ou avec le certificat d'une
autre autorité est rejeté, et que le bon client est identifié.

> [!WARNING]
> `InsecureSkipVerify: true` désactive toute vérification du certificat : la
> connexion est chiffrée, mais avec n'importe qui, y compris un attaquant
> placé au milieu. Il n'apparaît nulle part dans ce chapitre, même dans les
> tests : `RootCAs` avec l'autorité de test est aussi simple, et vérifie
> vraiment.

## Protéger les secrets

### Un type qui ne fuit jamais

Une clé d'API finit souvent dans un journal par accident : un
`log.Info("config", "cfg", cfg)`, une erreur formatée avec `%v`, une
structure renvoyée en JSON. Un type dédié masque toutes ses représentations :

```go
type Secret struct {
	valeur string
}

func (s Secret) Valeur() string             { return s.valeur } // explicite
func (s Secret) String() string             { return "[masqué]" } // fmt, erreurs
func (s Secret) GoString() string           { return "[masqué]" } // %#v
func (s Secret) LogValue() slog.Value       { return slog.StringValue("[masqué]") }
func (s Secret) MarshalText() ([]byte, error) { return []byte("[masqué]"), nil } // JSON

// Egal compare en temps constant.
func (s Secret) Egal(autre string) bool {
	return subtle.ConstantTimeCompare([]byte(s.valeur), []byte(autre)) == 1
}
```

L'exemple testé le prouve :

```go
cfg := Config{Base: "taches", MotDePasse: secret.Nouveau("hunter2")}
fmt.Printf("%+v\n", cfg)                 // {Base:taches MotDePasse:[masqué]}
json.Marshal(cfg)                        // {"Base":"taches","MotDePasse":"[masqué]"}
log.Info("connexion", "mot_de_passe", cfg.MotDePasse) // mot_de_passe=[masqué]
cfg.MotDePasse.Valeur()                  // hunter2, explicitement
```

Le nom `Valeur()` est voulu : il se repère à la relecture. Le serveur peut
alors journaliser sa configuration sans crainte :

```json
{"level":"INFO","msg":"configuration","emetteur":"http://localhost:18080/default","audience":"api-notes","cle_admin":"[masqué]"}
```

### D'où viennent les secrets

Une variable d'environnement est visible dans `docker inspect`, dans
`/proc/<pid>/environ` et dans les rapports d'erreur de certains outils.
Docker et Kubernetes préfèrent monter les secrets comme **fichiers**. La
convention `NOM_FILE` accepte les deux :

```go
func Lit(getenv func(string) string, nom string) (Secret, error) {
	if fichier := getenv(nom + "_FILE"); fichier != "" {
		contenu, err := os.ReadFile(fichier)
		if err != nil {
			// L'erreur cite le fichier, jamais son contenu.
			return Secret{}, fmt.Errorf("secret %s : %w", nom, err)
		}
		return Nouveau(strings.TrimRight(string(contenu), "\r\n")), nil
	}
	return Nouveau(getenv(nom)), nil
}
```

En production, un gestionnaire de secrets ([Vault](https://www.vaultproject.io),
[OpenBao](https://openbao.org), les coffres des fournisseurs de cloud) les
délivre, les renouvelle et trace leurs accès. Et pour fabriquer un jeton ou
une clé aléatoire, `crypto/rand.Text()` (Go 1.24) renvoie une chaîne d'au
moins 128 bits d'aléa :

```go
jeton := rand.Text() // par exemple « 4EZKLSOVQ2DAHKAXRYXQU7NZ5I »
```

> [!CAUTION]
> Jamais `math/rand` pour un secret : ses valeurs sont prévisibles. Et jamais
> de secret dans le dépôt Git, même « temporairement » : l'historique le
> conserve. [gitleaks](https://github.com/gitleaks/gitleaks) le détecte avant
> le commit ou dans l'intégration continue.

## Stocker les mots de passe

Si l'application gère tout de même des mots de passe, elle n'en stocke que
des **empreintes**, avec une fonction **lente par conception** : argon2id,
recommandée par l'OWASP et la RFC 9106. Un SHA-256, conçu pour être rapide,
se teste par milliards à la seconde sur une carte graphique.

```go
func (p Parametres) Hache(motDePasse string) string {
	sel := make([]byte, p.TailleSel)
	rand.Read(sel) // ne renvoie jamais d'erreur depuis Go 1.24
	cle := argon2.IDKey([]byte(motDePasse), sel, p.Iterations, p.Memoire, p.Parallelism, p.TailleCle)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memoire, p.Iterations, p.Parallelism, b64.EncodeToString(sel), b64.EncodeToString(cle))
}
```

L'empreinte, au format PHC, contient le sel et les paramètres :

```
$argon2id$v=19$m=65536,t=3,p=4$<sel>$<empreinte>
```

- le **sel** aléatoire donne deux empreintes différentes à deux utilisateurs
  qui ont le même mot de passe, et rend inutiles les tables précalculées ;
- les **paramètres** voyagent avec l'empreinte : on peut les renforcer plus
  tard, et recalculer l'empreinte à la connexion suivante (`ARenouveler`) ;
- la **vérification** relit les paramètres, recalcule, et compare en temps
  constant. Elle borne aussi les paramètres lus : une empreinte altérée dans
  la base ne doit pas pouvoir demander des gigaoctets de mémoire.

Avec 64 Mio et 3 passes, un calcul prend environ 45 ms sur un processeur
récent (`BenchmarkHache`) : imperceptible pour un utilisateur, ruineux pour
qui teste des milliards de candidats.

## Valider les entrées

Toute donnée qui vient du client est hostile jusqu'à preuve du contraire.

```go
func (s *Serveur) creeNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, TailleMax) // 16 Kio au plus
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // « auteur » ou « id » sont refusés, pas ignorés
	var n nouvelleNote
	if err := dec.Decode(&n); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "requête trop volumineuse", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "JSON invalide : "+err.Error(), http.StatusBadRequest)
		return
	}
	if dec.More() {
		http.Error(w, "JSON invalide : un seul objet attendu", http.StatusBadRequest)
		return
	}
	if err := n.valide(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	…
}
```

Chaque ligne a sa raison :

- **`MaxBytesReader`** : sans limite, un client envoie un corps de plusieurs
  gigaoctets et épuise la mémoire ;
- **un type dédié** (`nouvelleNote`, sans `ID` ni `Auteur`) et
  **`DisallowUnknownFields`** : un client ne peut pas choisir l'auteur de la
  note en ajoutant `"auteur": "bob"` (« mass assignment ») ;
- **`dec.More()`** : un seul objet JSON, pas une suite ;
- **une liste blanche** : longueur maximale, pas de caractère de contrôle. On
  décrit ce qui est permis, plutôt que de chercher à énumérer ce qui est
  dangereux.

En sortie, `encoding/json` échappe `<`, `>` et `&` : un contenu
`<script>alert(1)</script>` ressort `\u003cscript\u003e…`, et ne peut pas être
interprété comme du HTML. `html/template` (chapitre « Templates ») échappe
selon le contexte, et les paramètres de `database/sql` (`?`, `$1`) empêchent
l'injection SQL : on ne construit **jamais** une requête par concaténation.

### Confiner l'accès aux fichiers : os.Root

Servir un fichier dont le nom vient du client est un piège classique : le nom
`../../etc/passwd`, encodé `%2e%2e%2f`, passe le routeur (qui ne nettoie que le
chemin brut), et `r.PathValue` le décode. `filepath.Join` puis une
vérification de préfixe est fragile : liens symboliques, chemins absolus,
particularités de Windows.

`os.Root` (Go 1.24) ouvre un répertoire et garantit qu'aucune opération ne
peut en sortir, liens symboliques compris :

```go
fichiers, err := os.OpenRoot("fichiers")
…
f, err := s.fichiers.Open(r.PathValue("nom"))
if err != nil {
	s.log.WarnContext(r.Context(), "fichier refusé", "nom", nom, "raison", err.Error())
	http.NotFound(w, r) // même réponse que pour un fichier absent
	return
}
```

```json
{"level":"WARN","msg":"fichier refusé","nom":"../../etc/passwd","raison":"openat ../../etc/passwd: path escapes from parent"}
```

Le fichier est servi en `application/octet-stream` avec
`Content-Disposition: attachment` : un fichier HTML déposé par un utilisateur
est téléchargé, jamais affiché, et ne peut pas exécuter de script.

## Les protections du navigateur

### CSRF : http.CrossOriginProtection

Une attaque CSRF fait envoyer par le navigateur de la victime une requête vers
votre site, depuis un site piégé, avec les **cookies** de la victime. Depuis
Go 1.25, `http.CrossOriginProtection` refuse les requêtes « non sûres »
(`POST`, `PUT`, `DELETE`…) venues d'une autre origine. Il s'appuie sur
l'en-tête `Sec-Fetch-Site` des navigateurs récents, ou à défaut sur `Origin` :

```go
csrf := http.NewCrossOriginProtection()
return entetes(csrf.Handler(mux))
```

Plus besoin de jetons CSRF cachés dans les formulaires. Une API authentifiée
par l'en-tête `Authorization`, comme celle-ci, n'est pas exposée au CSRF (le
navigateur n'ajoute pas cet en-tête de lui-même). La protection ne coûte rien
pour autant, et couvre le jour où l'application ajoutera une session par
cookie.

### Les en-têtes de sécurité

```go
h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
h.Set("X-Content-Type-Options", "nosniff")
h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
h.Set("Referrer-Policy", "no-referrer")
h.Set("Cache-Control", "no-store")
```

| En-tête | Effet |
|---------|-------|
| `Strict-Transport-Security` | le navigateur n'utilise plus que HTTPS pour ce domaine |
| `X-Content-Type-Options: nosniff` | le navigateur ne devine pas le type d'un contenu |
| `Content-Security-Policy` | aucune ressource chargée, aucun affichage dans un cadre |
| `Referrer-Policy` | l'URL de la page n'est pas transmise aux autres sites |
| `Cache-Control: no-store` | les réponses personnelles ne restent pas dans un cache |

## La cryptographie dans la bibliothèque standard

Go fait évoluer sa bibliothèque `crypto` à chaque version. Les ajouts récents :

| Version | Ajout |
|---------|-------|
| Go 1.24 | `crypto/mlkem` (échange de clés post-quantique), activé par défaut dans TLS |
| Go 1.24 | `crypto/rand.Text`, et `rand.Read` qui ne peut plus échouer |
| Go 1.24 | module cryptographique FIPS 140-3 (`GOFIPS140`, `GODEBUG=fips140=on`) |
| Go 1.24 | `crypto/pbkdf2`, `crypto/hkdf`, `crypto/sha3` dans la bibliothèque standard |
| Go 1.24 | `os.Root`, accès aux fichiers confiné |
| Go 1.25 | `http.CrossOriginProtection` |
| Go 1.26 | `crypto/hpke` (chiffrement hybride à clé publique, RFC 9180) |
| Go 1.26 | échanges de clés TLS `SecP256r1MLKEM768` et `SecP384r1MLKEM1024` |
| Go 1.27 | `crypto/mldsa` (signatures post-quantiques, FIPS 204) |

### Les signatures post-quantiques : crypto/mldsa

Un ordinateur quantique suffisamment puissant casserait RSA et ECDSA. ML-DSA,
normalisé par le NIST en 2024, y résiste. Go 1.27 l'intègre :

```go
// contexte distingue les signatures de cette application de toute autre
// signature faite avec la même clé : une signature de facture ne peut pas
// être rejouée comme signature de contrat.
var contexte = &mldsa.Options{Context: "golang-demystified/factures/v1"}

func Signe(cle *mldsa.PrivateKey, message []byte) ([]byte, error) {
	return cle.Sign(rand.Reader, message, contexte)
}

func Verifie(cle *mldsa.PublicKey, message, sig []byte) error {
	return mldsa.Verify(cle, message, sig, contexte)
}
```

```go
cle, _ := mldsa.GenerateKey(mldsa.MLDSA65())
sig, _ := signature.Signe(cle, facture)
// taille de la signature : 3309 octets
// taille de la clé publique : 1952 octets
// authentique : true
// falsifiée : false
```

Le prix de la résistance quantique est la **taille** : 3 309 octets de
signature, contre 64 pour ECDSA P-256. C'est pourquoi la transition commence
par l'échange de clés (déjà en place dans TLS) avant les certificats.

> [!TIP]
> La règle d'or reste la même : n'inventez pas de cryptographie, et préférez
> les constructions de haut niveau. Pour chiffrer des données, `crypto/cipher`
> avec AES-GCM (ou `crypto/hpke` pour un destinataire identifié par sa clé
> publique) ; pour signer, `crypto/ed25519` ou `crypto/mldsa` ; pour les mots de
> passe, argon2id. Et toujours `subtle.ConstantTimeCompare` pour comparer des
> secrets.

## Surveiller les dépendances

Une application Go embarque des dizaines de modules tiers. `govulncheck`,
présenté au chapitre « CI/CD », signale les vulnérabilités connues, mais
**seulement si le code vulnérable est réellement appelé** : peu de bruit, et
une alerte sérieuse quand il en émet une. Il tourne à chaque poussée dans
l'intégration continue du dépôt :

```console
$ go tool govulncheck ./examples/approfondissements/securite/...
No vulnerabilities found.
```

golangci-lint propose aussi `gosec`, qui repère les motifs risqués :
`InsecureSkipVerify`, `math/rand` pour un secret, permissions de fichiers
trop larges, requêtes SQL construites par concaténation.

## En résumé

- **Déléguez l'authentification** à un fournisseur OpenID Connect, et faites
  vérifier les jetons par une bibliothèque éprouvée : signature, algorithme,
  émetteur, audience, expiration.
- **Autorisez** par portée, puis par propriété des données.
- **TLS 1.3** par défaut, mTLS entre services, des délais sur le serveur, et
  jamais `InsecureSkipVerify`.
- **Les secrets** dans un type qui se masque, lus depuis des fichiers montés.
- **Les mots de passe** en argon2id, comparés en temps constant.
- **Les entrées** limitées en taille, décodées dans des types dédiés, validées
  par liste blanche ; **les fichiers** confinés par `os.Root`.
- **Testez les attaques** : jeton expiré, `alg: none`, `../`, champ inattendu,
  requête intersite. Et vérifiez que le test attaque vraiment.
