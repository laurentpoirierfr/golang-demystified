---
title: "I.3 Rust et Go"
description: "ramasse-miettes ou propriété, la même mesure dans les deux langages, et comment choisir."
weight: 17
---

# Chapitre I.3 — Rust et Go : deux réponses à des questions différentes

Rust et Go sont souvent comparés : deux langages compilés, récents, qui
produisent des binaires rapides et ont conquis l'infrastructure logicielle.
Ils répondent pourtant à des questions différentes. Go demande : **comment
une équipe peut-elle écrire, lire et maintenir simplement des services
fiables ?** Rust demande : **comment écrire du code système aussi rapide que
le C, sans ses erreurs de mémoire ?** Ce chapitre explique les choix de
chacun, les mesure sur un même programme, et propose une manière de choisir.

Le code des deux versions est dans [`examples/decouvrir/rust-et-go`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/decouvrir/rust-et-go).

## Deux histoires

| | Go | Rust |
|-|----|------|
| Origine | Google, 2007 ; Robert Griesemer, Rob Pike, Ken Thompson | Mozilla, à partir de 2009 ; projet personnel de Graydon Hoare dès 2006 |
| Version 1.0 | mars 2012 | mai 2015 |
| Problème de départ | compilations lentes et complexité du C++ dans les grands services de Google | erreurs de mémoire et de concurrence du C++ dans un navigateur (Firefox) |
| Gouvernance | l'équipe Go de Google, avec la communauté | la Rust Foundation (2021) et des équipes ouvertes |
| Évolution | promesse de compatibilité Go 1 : un programme de 2012 compile toujours | des « éditions » (2015, 2018, 2021, 2024) qui font évoluer le langage, compatibles entre elles |

## La mémoire : ramasse-miettes contre propriété

C'est la différence fondamentale, dont découlent presque toutes les autres.

**Go** libère la mémoire avec un **ramasse-miettes** concurrent (chapitre
« Les coulisses de Go ») : le développeur alloue, et ne libère jamais. Le prix
est un peu de processeur, un peu plus de mémoire, et des pauses très courtes
(de l'ordre de la dizaine de microsecondes).

**Rust** n'a pas de ramasse-miettes. Chaque valeur a un **propriétaire**
unique, et elle est libérée quand son propriétaire disparaît. On peut
l'**emprunter**, soit en lecture par plusieurs, soit en écriture par un seul,
jamais les deux à la fois. Le compilateur, par le *borrow checker*, vérifie
ces règles **avant** l'exécution. Le prix est payé par le développeur : il
faut convaincre le compilateur que le programme est correct, ce qui demande
un apprentissage réel.

Les deux langages sont **sûrs pour la mémoire** : pas de dépassement de
tampon, pas de pointeur libéré deux fois, en dehors de `unsafe`, que les deux
proposent. Ils diffèrent sur un point : les courses de données.

## La concurrence : vérifiée à l'exécution, ou à la compilation

Deux tâches qui modifient le même compteur sans synchronisation forment une
**course de données**. En Go, le compilateur accepte ce programme :

```go
compteur := 0
var wg sync.WaitGroup
for range 2 {
	wg.Go(func() {
		for range 100_000 {
			compteur++ // lecture et écriture concurrentes, sans verrou
		}
	})
}
wg.Wait()
fmt.Println(compteur)
```

Cinq exécutions donnent cinq résultats faux, et différents :

```console
115408 101550 111283 131794 100796
```

Le **détecteur de courses** de Go la trouve, mais à l'exécution, et seulement
si le chemin fautif est exercé (par les tests, par exemple) :

```console
$ go run -race ./course
==================
WARNING: DATA RACE
Read at 0x… by goroutine 8:
  main.main.func1()
      …/course/main.go:21
Previous write at 0x… by goroutine 9:
  main.main.func1()
      …/course/main.go:21
```

> [!NOTE]
> La première version de ce programme bouclait mille fois, et affichait
> toujours le bon résultat, 2 000 : la première goroutine finissait avant que
> la seconde ne démarre. Une course peut rester invisible longtemps. C'est
> pourquoi l'intégration continue de ce dépôt lance les tests avec `-race`.

En Rust, le même programme **ne compile pas** (message abrégé) :

```rust
let mut compteur = 0;
thread::scope(|s| {
    for _ in 0..2 {
        s.spawn(|| {
            for _ in 0..100_000 {
                compteur += 1;
            }
        });
    }
});
```

```console
error[E0499]: cannot borrow `compteur` as mutable more than once at a time
  --> main.rs:12:21
   |
12 |             s.spawn(|| {
   |             -       ^^ `compteur` was mutably borrowed here in the previous iteration of the loop
13 | |               for _ in 0..100_000 {
14 | |                   compteur += 1;
   | |                   -------- borrows occur due to use of `compteur` in closure
```

Pour partager le compteur, Rust impose un type qui le permet (`AtomicU64`,
`Mutex`) : la classe d'erreurs entière disparaît. C'est l'atout majeur de
Rust, que Go compense par son détecteur, ses canaux et sa discipline.

### Goroutines et async

Go a un **ordonnanceur intégré** : une goroutine coûte quelques kilo-octets,
on en lance des centaines de milliers, et le code reste séquentiel à lire. Les
entrées-sorties bloquantes sont gérées par le runtime, sans mot-clé
particulier.

Rust offre des fils du système et, pour les entrées-sorties massives,
`async`/`await` (stable depuis 2019). Mais le langage ne fournit pas
d'exécuteur : on en choisit un, le plus souvent [Tokio](https://tokio.rs).
C'est plus flexible, plus économe, et plus complexe : une fonction `async` ne
s'appelle pas comme une autre, et l'écosystème se partage entre code
synchrone et asynchrone.

## Le même programme, mesuré

Les deux versions comptent les mots les plus fréquents d'un texte. Elles sont
écrites simplement, chacune de façon idiomatique.

```go
func Compte(texte string, n int) []Frequence {
	compte := map[string]int{}
	for mot := range strings.FieldsFuncSeq(texte, func(r rune) bool { return !unicode.IsLetter(r) }) {
		compte[strings.ToLower(mot)]++
	}
	…
}
```

```rust
fn compte(texte: &str, n: usize) -> Vec<(String, usize)> {
    let mut compte: HashMap<String, usize> = HashMap::new();
    for mot in texte.split(|c: char| !c.is_alphabetic()).filter(|m| !m.is_empty()) {
        *compte.entry(mot.to_lowercase()).or_insert(0) += 1;
    }
    …
}
```

Les mesures ont été faites sur la même machine (Go 1.27.4, Rust 1.99.1, un
texte de 100 Mo, médiane de cinq exécutions). Les deux programmes donnent
exactement le même résultat.

| | Go | Rust |
|-|----|------|
| Binaire | 2,4 Mio (1,6 Mio sans symboles) | 406 Kio (326 Kio sans symboles) |
| Recompilation après une modification | 155 ms | 87 ms (debug), 2,7 s (release, optimisation à l'édition de liens) |
| Exécution, versions naïves | 0,71 s, 193 Mio | 0,87 s, 97 Mio |

Le binaire Go embarque son runtime (ordonnanceur, ramasse-miettes) ; celui de
Rust, non. Les deux sont autonomes et se déploient en copiant un fichier.

### Les écarts viennent des détails, pas du langage

Les versions naïves suggèrent que Go est plus rapide, et que Rust consomme
deux fois moins de mémoire. Deux variantes, hors du dépôt, ont cherché la
cause de chaque écart :

| Variante | Temps | Mémoire |
|----------|-------|---------|
| Go naïf | 0,71 s | 193 Mio |
| Go, sans copier le fichier en chaîne | 0,67 s | 100 Mio |
| Rust naïf | 0,87 s | 97 Mio |
| Rust, sans allouer de minuscules inutilement | 0,74 s | 97 Mio |
| Rust, avec une fonction de hachage rapide | 0,61 s | 97 Mio |

- La mémoire de Go doublait à cause d'une ligne : `string(texte)` **copie**
  les 100 Mo du fichier. Sans cette copie, Go consomme autant que Rust.
- Rust était plus lent pour deux raisons. `to_lowercase()` alloue une nouvelle
  chaîne pour chaque mot, quand `strings.ToLower` de Go renvoie la chaîne
  d'origine si elle est déjà en minuscules. Et la table de hachage de Rust
  utilise par défaut **SipHash**, une fonction résistante aux attaques par
  déni de service, mais plus lente que celle de Go. Avec une fonction plus
  rapide (`rustc-hash`), Rust passe devant.

La leçon dépasse ces deux langages : les **choix par défaut** des
bibliothèques et quelques lignes d'écriture pèsent plus que le langage. Rust
offre un plafond plus haut et des performances plus prévisibles (pas de
ramasse-miettes) ; Go atteint une bonne performance avec du code simple.

> [!WARNING]
> Ces chiffres valent pour ce programme, sur cette machine. Ils ne disent rien
> des grands projets : la compilation de Rust y est réputée bien plus lente
> que celle de Go, à cause de la monomorphisation des génériques et des
> macros. Mesurez votre propre cas.

## Les autres différences

| Thème | Go | Rust |
|-------|----|------|
| Erreurs | valeurs `error`, `if err != nil` | `Result<T, E>`, opérateur `?`, `Option<T>` au lieu de `nil` |
| Absence de valeur | `nil`, valeurs zéro | `Option<T>` : pas de `nil` |
| Génériques | simples (Go 1.18), contraintes par interfaces | riches : traits, types associés, monomorphisation |
| Abstraction | interfaces implicites | traits explicites, très expressifs |
| Macros | aucune ; génération de code (`go generate`) | macros déclaratives et procédurales |
| Outillage | `go` : construire, tester, formater, profiler, intégré | `cargo` : construire, tester, gérer les dépendances, excellent |
| Apprentissage | quelques jours pour être productif | plusieurs semaines pour être à l'aise avec l'emprunt |
| Lisibilité | délibérément limitée en fonctionnalités : un seul style | plus expressif, donc plus de façons d'écrire la même chose |
| Bibliothèque standard | large : HTTP, TLS, JSON, crypto, tests | réduite ; l'écosystème (crates) fournit le reste |
| Cibles sans système | TinyGo (microcontrôleurs, WebAssembly) | `no_std` : noyaux, micrologiciels, de premier plan |

## Où chacun excelle

**Go** domine l'**infrastructure cloud** : Kubernetes, Docker, Terraform,
Prometheus, etcd, Consul (chapitre « Les grands projets écrits en Go »). Il
convient aux services réseau, aux API, aux outils en ligne de commande, aux
équipes nombreuses dont les membres changent : le code se relit facilement,
et un nouveau venu devient productif vite.

**Rust** s'impose là où le contrôle de la mémoire et du temps compte : le
**noyau Linux** l'intègre depuis la version 6.1 (fin 2022), d'abord pour son
infrastructure, puis pour des pilotes ;
Android, Windows et Firefox en contiennent ; Firecracker (les micro-machines
virtuelles d'AWS Lambda) et Pingora (le proxy de Cloudflare) sont écrits en
Rust, comme de nombreux outils en ligne de commande (ripgrep) et moteurs
WebAssembly.

Deux décisions publiques éclairent le choix :

- en **2020**, Discord a décrit le passage d'un de ses services de Go à Rust :
  avec un très grand cache en mémoire, les cycles du ramasse-miettes de
  l'époque provoquaient des pics de latence réguliers. Le ramasse-miettes de
  Go a beaucoup progressé depuis, et `GOMEMLIMIT` répond en partie à ce cas,
  mais le principe demeure : sans ramasse-miettes, pas de pause du tout ;
- en **2025**, Microsoft a annoncé le portage du compilateur TypeScript en
  **Go**, plutôt qu'en Rust. L'équipe a expliqué qu'elle portait un code
  existant, plein de structures cycliques et de mémoire partagée, et que la
  structure du code et la gestion automatique de la mémoire de Go
  permettaient une traduction presque ligne à ligne.

## Choisir

| Si le projet… | Plutôt |
|---------------|--------|
| est un service réseau, une API, un outil d'infrastructure | Go |
| doit être écrit vite, et maintenu par une équipe qui change | Go |
| a des contraintes de latence strictes, sans aucune pause tolérée | Rust |
| tourne sans système d'exploitation, ou dans un noyau | Rust |
| doit tirer le maximum du matériel (moteur, codec, base de données) | Rust |
| est une bibliothèque appelée depuis d'autres langages (via l'ABI C) | Rust |
| porte un code existant à ramasse-miettes (Java, C#, TypeScript) | Go |

Les deux se combinent : un service Go peut appeler une bibliothèque Rust
compilée en bibliothèque C (`cdylib`), par cgo (chapitre « Les coulisses de
Go »), ou charger un module WebAssembly écrit en Rust par wazero (chapitre
« WebAssembly et TinyGo »).

## Reproduire les mesures

```bash
make rust-et-go   # tests Go et Rust, et vérification que la course ne compile pas en Rust
```

La version Go est compilée et testée avec le reste du dépôt. La version Rust
l'est dans le conteneur officiel `rust:1.99.1-slim`, sans rien installer.

## En résumé

- **Go** : un ramasse-miettes, un ordonnanceur intégré, peu de concepts. Il
  vise la simplicité d'écriture et de lecture, et la productivité d'une
  équipe.
- **Rust** : la propriété et l'emprunt, vérifiés par le compilateur. Il vise la
  performance et la sûreté maximales, au prix d'un apprentissage exigeant.
- **La sûreté** : les deux sont sûrs pour la mémoire ; Rust élimine aussi les
  courses de données à la compilation, Go les détecte à l'exécution.
- **La performance** : sur un même programme, les écarts viennent surtout des
  choix par défaut des bibliothèques. Rust a le plafond le plus haut et la
  latence la plus prévisible.
- **Le choix** dépend du problème : services et outils d'infrastructure pour
  Go, code système et contraintes de latence strictes pour Rust.
