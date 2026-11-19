---
title: "Architecture hexagonale"
weight: 40
---

Le chapitre précédent a construit un service web en couches simples : HTTP,
stockage, domaine. C'est suffisant tant que la logique métier reste mince.
Lorsqu'elle grandit, avec des règles, des calculs et des processus, il devient
précieux de la **protéger** de la technique : la base de données, le
protocole HTTP, le fournisseur d'e-mails ou l'heure système.

L'**architecture hexagonale**, ou *ports et adaptateurs*, proposée par Alistair
Cockburn, répond à ce besoin. La *Clean Architecture* de Robert C. Martin et
l'architecture en oignon en sont des variantes, qui reposent sur le même
principe.

Ce chapitre l'applique à un domaine plus riche que la liste de tâches : les
**emprunts d'une bibliothèque**. Le code est dans
`examples/approfondissements/hexagonale`.

## Le principe

L'application est organisée en cercles concentriques. **Les dépendances
pointent toujours vers le centre** :

```
  ┌───────────────────────────────────────────────┐
  │ adaptateurs : web, SQL, SMTP, horloge système │
  │   ┌───────────────────────────────────────┐   │
  │   │ application : cas d'usage et ports    │   │
  │   │   ┌───────────────────────────────┐   │   │
  │   │   │ domaine : règles métier       │   │   │
  │   │   └───────────────────────────────┘   │   │
  │   └───────────────────────────────────────┘   │
  └───────────────────────────────────────────────┘
```

- Le **domaine** contient les règles métier. Il n'importe que la bibliothèque
  standard.
- L'**application** orchestre les **cas d'usage** (emprunter, rendre,
  relancer) et déclare les **ports** : les interfaces dont elle a besoin pour
  parler au monde extérieur.
- Les **adaptateurs** implémentent ces ports avec une technologie précise. Les
  adaptateurs **primaires** (*driving*) appellent l'application : API HTTP,
  ligne de commande, consommateur de messages. Les adaptateurs **secondaires**
  (*driven*) sont appelés par elle : base de données, envoi d'e-mails,
  horloge.
- Le programme principal (`cmd/`) **assemble** le tout.

En Go, ce principe se traduit naturellement : les interfaces sont satisfaites
implicitement, et l'usage veut qu'elles soient définies **par celui qui les
utilise**. Les ports sont donc simplement des interfaces déclarées dans le
package `application`.

```
hexagonale/
├── domaine/                  règles d'emprunt, sans dépendance
├── application/
│   ├── ports.go              les interfaces vers l'extérieur
│   └── service.go            les cas d'usage
├── adaptateurs/
│   ├── web/                  primaire : HTTP
│   ├── memoire/              secondaire : stockage en mémoire
│   └── notification/         secondaire : notifications, horloge, UUID
├── cmd/bibliotheque/         assemblage
└── architecture_test.go      vérification des dépendances
```

## Le domaine

Le domaine exprime les règles de la bibliothèque dans le vocabulaire du
métier : un membre emprunte au plus trois livres, pour 21 jours, et paie 20
centimes par jour de retard, plafonnés à 5 euros. Tant qu'il a des pénalités
impayées, il ne peut plus emprunter.

```go
const (
	EmpruntsMax      = 3
	DureeEmprunt     = 21 * 24 * time.Hour
	PenaliteParJour  = 20 // en centimes
	PenaliteMaximale = 500
)

var (
	ErrLivreIndisponible = errors.New("livre déjà emprunté")
	ErrQuotaAtteint      = fmt.Errorf("%d emprunts au plus", EmpruntsMax)
	ErrMembreSuspendu    = errors.New("membre suspendu : pénalités impayées")
	ErrDejaRendu         = errors.New("emprunt déjà rendu")
	ErrIntrouvable       = errors.New("introuvable")
)

// PeutEmprunter applique les règles d'emprunt.
func PeutEmprunter(m Membre, empruntsEnCours int, livreDisponible bool) error {
	switch {
	case m.Penalites > 0:
		return ErrMembreSuspendu
	case empruntsEnCours >= EmpruntsMax:
		return ErrQuotaAtteint
	case !livreDisponible:
		return ErrLivreIndisponible
	}
	return nil
}

// Rend clôt l'emprunt à l'instant t et renvoie la pénalité due, en centimes.
func (e *Emprunt) Rend(t time.Time) (int, error) {
	if !e.EnCours() {
		return 0, ErrDejaRendu
	}
	e.Retour = t
	if !t.After(e.Echeance) {
		return 0, nil
	}
	jours := int((t.Sub(e.Echeance) + 24*time.Hour - 1) / (24 * time.Hour))
	return min(jours*PenaliteParJour, PenaliteMaximale), nil
}
```

Le domaine ne lit jamais l'heure lui-même : l'instant est un **paramètre**.
Ses règles se testent donc avec de simples tables de cas, sans base de
données ni attente :

```go
cas := []struct {
	nom     string
	retour  time.Duration // après l'échéance
	attendu int
}{
	{"pile à l'échéance", 0, 0},
	{"une heure de retard", time.Hour, 20},
	{"trois jours", 3 * 24 * time.Hour, 60},
	{"plafond", 100 * 24 * time.Hour, PenaliteMaximale},
}
```

## Les ports

L'application déclare ce dont elle a besoin, sans dire comment c'est fait :

```go
// Catalogue donne accès aux livres et aux membres.
type Catalogue interface {
	Livre(ctx context.Context, isbn string) (domaine.Livre, error)
	Membre(ctx context.Context, id string) (domaine.Membre, error)
	EnregistreMembre(ctx context.Context, m domaine.Membre) error
}

// Emprunts stocke les emprunts.
type Emprunts interface {
	Emprunt(ctx context.Context, id string) (domaine.Emprunt, error)
	EnCoursPourLivre(ctx context.Context, isbn string) (bool, error)
	EnCoursPourMembre(ctx context.Context, membre string) ([]domaine.Emprunt, error)
	EnCours(ctx context.Context) ([]domaine.Emprunt, error)
	Enregistre(ctx context.Context, e domaine.Emprunt) error
}

type Notifieur interface {
	Notifie(ctx context.Context, m domaine.Membre, message string) error
}

// Horloge donne l'heure : la remplacer rend le temps contrôlable en test.
type Horloge interface {
	Maintenant() time.Time
}

type Identifiants interface {
	Nouveau() string
}

// Transactions exécute une fonction de manière atomique.
type Transactions interface {
	Atomique(ctx context.Context, f func(ctx context.Context) error) error
}
```

Deux ports méritent une explication :

- **`Horloge` et `Identifiants`** : l'heure et le hasard sont des dépendances
  cachées. En faire des ports rend chaque cas d'usage **déterministe** en test.
- **`Transactions`** : un cas d'usage lit puis écrit plusieurs données (vérifier
  la disponibilité, puis enregistrer l'emprunt). Ces étapes doivent être
  atomiques, sinon deux membres pourraient emprunter le même livre au même
  instant. L'application exprime ce besoin ; l'adaptateur SQL l'implémenterait
  par une transaction (en plaçant le `*sql.Tx` dans le contexte), l'adaptateur
  en mémoire par un verrou.

## Les cas d'usage

Un cas d'usage enchaîne les étapes : charger les données par les ports,
appliquer les règles du domaine, enregistrer le résultat.

```go
func (s *Service) Emprunte(ctx context.Context, membreID, isbn string) (domaine.Emprunt, error) {
	var e domaine.Emprunt
	err := s.tx.Atomique(ctx, func(ctx context.Context) error {
		m, err := s.catalogue.Membre(ctx, membreID)
		if err != nil {
			return err
		}
		if _, err := s.catalogue.Livre(ctx, isbn); err != nil {
			return err
		}
		pris, err := s.emprunts.EnCoursPourLivre(ctx, isbn)
		if err != nil {
			return err
		}
		enCours, err := s.emprunts.EnCoursPourMembre(ctx, membreID)
		if err != nil {
			return err
		}
		if err := domaine.PeutEmprunter(m, len(enCours), !pris); err != nil {
			return err
		}
		e = domaine.NouvelEmprunt(s.identifiants.Nouveau(), isbn, membreID, s.horloge.Maintenant())
		return s.emprunts.Enregistre(ctx, e)
	})
	if err != nil {
		return domaine.Emprunt{}, fmt.Errorf("emprunt de %s par %s : %w", isbn, membreID, err)
	}
	return e, nil
}
```

Le cas d'usage ne contient **aucune règle métier** : la décision appartient à
`domaine.PeutEmprunter`. Il ne contient pas non plus de technique : ni SQL, ni
HTTP. Il se lit comme la description du processus.

## Les adaptateurs

### Primaire : HTTP

L'adaptateur web traduit les requêtes en appels, et les erreurs du domaine en
statuts HTTP. Il déclare lui aussi l'interface dont il a besoin :

```go
type CasDUsage interface {
	Emprunte(ctx context.Context, membre, isbn string) (domaine.Emprunt, error)
	Rend(ctx context.Context, emprunt string) (int, error)
}

func erreur(w http.ResponseWriter, err error) {
	statut := http.StatusInternalServerError
	switch {
	case errors.Is(err, domaine.ErrIntrouvable):
		statut = http.StatusNotFound
	case errors.Is(err, domaine.ErrLivreIndisponible),
		errors.Is(err, domaine.ErrQuotaAtteint),
		errors.Is(err, domaine.ErrMembreSuspendu),
		errors.Is(err, domaine.ErrDejaRendu):
		statut = http.StatusConflict
	}
	…
}
```

C'est le seul endroit qui connaît à la fois le métier et le protocole. Un
adaptateur en ligne de commande ou un consommateur de messages piloterait les
mêmes cas d'usage, sans en modifier une ligne.

### Secondaires : stockage, notifications, horloge

Chaque adaptateur secondaire implémente un ou plusieurs ports. L'adaptateur
en mémoire implémente le stockage et les transactions :

```go
type Depot struct {
	mu       sync.Mutex
	livres   map[string]domaine.Livre
	membres  map[string]domaine.Membre
	emprunts map[string]domaine.Emprunt
}

// Atomique exécute f en détenant le verrou du dépôt.
func (d *Depot) Atomique(ctx context.Context, f func(context.Context) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return f(ctx)
}
```

Les ports techniques ont des implémentations de quelques lignes :

```go
type HorlogeSysteme struct{}

func (HorlogeSysteme) Maintenant() time.Time { return time.Now() }

type UUID struct{}

func (UUID) Nouveau() string { return uuid.NewV7().String() }
```

Un adaptateur PostgreSQL, construit comme le stockage du chapitre précédent,
remplacerait le dépôt en mémoire **sans toucher** au domaine ni à
l'application.

## L'assemblage

Le programme principal est le seul endroit qui connaît à la fois les ports et
les adaptateurs concrets. Il les branche ensemble :

```go
app := application.NouveauService(application.Dependances{
	Catalogue:    depot,
	Emprunts:     depot,
	Transactions: depot,
	Notifieur:    notification.Journal{Log: journal},
	Horloge:      notification.HorlogeSysteme{},
	Identifiants: notification.UUID{},
	Journal:      journal,
})

srv := &http.Server{Addr: ":8080", Handler: web.Routes(app), ReadHeaderTimeout: 5 * time.Second}
```

Pas de framework d'injection de dépendances : en Go, une structure et un
constructeur suffisent, et le compilateur vérifie que chaque adaptateur
satisfait bien son port.

## Tester chaque cercle séparément

L'architecture hexagonale facilite surtout les **tests** :

| Cercle        | Ce que l'on teste             | Avec                                    |
|---------------|-------------------------------|-----------------------------------------|
| domaine       | les règles métier             | des tables de cas, sans dépendance      |
| application   | les cas d'usage               | l'adaptateur en mémoire et des doubles  |
| adaptateurs   | la traduction technique       | un faux cas d'usage, ou une vraie base  |
| architecture  | la direction des dépendances  | l'analyse des imports                   |

Les cas d'usage se testent avec une **horloge que le test fait avancer** et un
**espion** qui enregistre les notifications au lieu de les envoyer :

```go
type horlogeFixe struct{ t time.Time }

func (h *horlogeFixe) Maintenant() time.Time   { return h.t }
func (h *horlogeFixe) Avance(d time.Duration) { h.t = h.t.Add(d) }

func TestRetardPenaliteEtSuspension(t *testing.T) {
	b := nouveauBanc()
	ctx := t.Context()
	e, _ := b.service.Emprunte(ctx, "alice", "go-1")

	b.horloge.Avance(21*24*time.Hour + 2*24*time.Hour + time.Hour) // 2 jours et 1 heure de retard
	penalite, err := b.service.Rend(ctx, e.ID)
	if err != nil || penalite != 60 {
		t.Fatalf("Rend = %d, %v ; attendu 60 centimes", penalite, err)
	}
	if _, err := b.service.Emprunte(ctx, "alice", "go-2"); !errors.Is(err, domaine.ErrMembreSuspendu) {
		t.Errorf("erreur = %v, attendu ErrMembreSuspendu", err)
	}
}
```

Trois semaines de la vie de la bibliothèque s'écoulent en une
microseconde, de manière parfaitement reproductible.

### Vérifier l'architecture

Une architecture n'est utile que si elle est respectée. Un test analyse les
imports avec `go/parser` et échoue si une dépendance pointe dans le mauvais
sens :

```go
func TestDirectionDesDependances(t *testing.T) {
	regles := []struct {
		paquet    string
		autorises []string // packages du projet autorisés
	}{
		{"domaine", nil},
		{"application", []string{"domaine"}},
	}
	for _, r := range regles {
		for _, imp := range imports(t, r.paquet) {
			…
			if !autorise {
				t.Errorf("%s ne doit pas importer %s", r.paquet, interne)
			}
		}
	}
}
```

```
--- FAIL: TestDirectionDesDependances (0.00s)
    architecture_test.go:69: domaine ne doit pas importer adaptateurs/memoire
```

Go interdit déjà les cycles d'imports ; ce test va plus loin et protège
l'intention de l'architecture, même face à un nouveau venu dans l'équipe.

## Quand l'utiliser ?

L'architecture hexagonale a un coût : plus de packages, plus d'interfaces, et
une traduction des données entre les couches. Elle se justifie lorsque :

- la **logique métier est riche** et évolue souvent ;
- l'application a **plusieurs points d'entrée** (API, ligne de commande,
  messages) ou plusieurs technologies à brancher ;
- les **tests** du métier doivent être rapides et nombreux.

Pour un service qui se contente de lire et d'écrire des données, la structure
plus simple du chapitre précédent suffit. Rien n'empêche de commencer simple,
puis d'extraire le domaine et les ports le jour où la complexité métier le
demande : en Go, déplacer une interface vers son utilisateur est une
refactorisation légère.

> « A little copying is better than a little dependency. » — Go Proverbs
