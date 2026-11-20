# Évolutions prévues

Liste des évolutions du livre, par ordre de priorité. Chaque élément est
développé sur une branche `feature/*` (git flow), puis publié par une release.

Légende : `[ ]` à faire, `[~]` en cours, `[x]` fait.

## 1. Qualité et publication

- [x] **Intégration continue du dépôt** (GitHub Actions)
  - vérifications à chaque push : `gofmt`, `go vet`, `go test -race`, `govulncheck`
  - tests d'intégration avec Testcontainers (Docker est disponible sur les machines GitHub)
  - vérification que le code généré est à jour (sqlc, oapi-codegen, controller-gen)
- [x] **Publication par l'intégration continue** : site, PDF et EPUB construits
  à chaque tag, au lieu d'être versionnés dans `docs/` et `book/`
- [x] **Chapitre « CI/CD d'un projet Go »** : GitHub Actions, golangci-lint,
  govulncheck, GoReleaser, images multi-architectures, SBOM

## 2. Nouveaux chapitres d'approfondissement

- [x] **Observabilité** : OpenTelemetry (traces et métriques), Prometheus,
  corrélation des journaux `slog` avec les traces, appliqués à l'API des tâches
- [x] **gRPC et Connect** : Protocol Buffers, Buf, connect-go, communication
  entre services
- [x] **Sécurité** : authentification OIDC et JWT, TLS, gestion des secrets,
  validation des entrées, bonnes pratiques avec `crypto`
- [x] **Architecture événementielle** : Kafka ou NATS avec Watermill (suite de
  la section sur le traitement de flux)
- [ ] **Temps réel** : WebSockets et Server-Sent Events
- [ ] **Go et l'IA** : appeler des modèles de langage, écrire un serveur MCP en
  Go, utiliser Ollama en local
- [ ] **WebAssembly et TinyGo** : Go dans le navigateur, `go:wasmexport`,
  microcontrôleurs
- [ ] **Les coulisses de Go** : réflexion, `unsafe`, cgo, fonctionnement de
  l'ordonnanceur et du ramasse-miettes
- [ ] **Go idiomatique** : motifs récurrents (options fonctionnelles,
  middlewares, injection de dépendances), anti-motifs fréquents, grille de
  relecture de code

## 3. Aides pédagogiques

- [ ] **Exercices corrigés** à la fin de chaque chapitre des parties I à III,
  avec les solutions testées dans `examples/`
- [ ] **Aide-mémoire** : syntaxe, idiomes courants, commandes `go` ; en annexe
  et en PDF séparé
- [ ] **Glossaire français-anglais** (goroutine, *receiver*, *embedding*…)
- [ ] **Chapitre « Go pour les développeurs Java, Python ou JavaScript »** :
  comparaison des concepts

## 4. Diffusion

- [ ] **Éditions versionnées** : une édition par version de Go, avec une page
  « nouveautés de l'édition »
- [ ] **Traduction anglaise**

## Fait

- [x] Nouveau thème Hextra, site dans `docs/`
- [x] Livre PDF et EPUB avec Quarto et Typst, couverture personnalisée
- [x] Parties I à IV réécrites pour Go 1.27
- [x] Approfondissements : concurrence, templates, API web, ligne de commande,
  sqlc et golang-migrate, architecture hexagonale, Kubernetes, performance
- [x] Tests d'intégration avec Testcontainers, Docker et Podman
- [x] Panorama des grands projets écrits en Go
- [x] Liens vers les projets cités et vers le code des exemples
