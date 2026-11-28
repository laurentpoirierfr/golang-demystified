---
title: "I.2 Les grands projets écrits en Go"
weight: 15
---

# Chapitre I.2 — Les grands projets écrits en Go

Go s'est imposé comme **le langage de l'infrastructure** : une grande partie
des outils qui font fonctionner le cloud, les conteneurs et l'observabilité
est écrite en Go. Ses qualités y sont décisives : des exécutables statiques
faciles à déployer, une concurrence intégrée pour les services réseau, des
performances proches du C pour une fraction de sa complexité, et une
compilation croisée triviale.

Ce chapitre présente quelques-uns de ces projets. Les lire est une excellente
façon d'apprendre : leur code est ouvert, et souvent exemplaire.

## Conteneurs et orchestration

| Projet | Rôle |
|--------|------|
| [Kubernetes](https://kubernetes.io) | L'orchestrateur de conteneurs devenu le standard du cloud |
| [Docker](https://www.docker.com) ([Moby](https://mobyproject.org)) | Le moteur de conteneurs qui a popularisé Go dès 2013 |
| [containerd](https://containerd.io) | L'environnement d'exécution de conteneurs utilisé par Docker et Kubernetes |
| [Podman](https://podman.io) | Moteur de conteneurs sans démon, compatible avec Docker |
| [Helm](https://helm.sh) | Le gestionnaire de paquets de Kubernetes |
| [kind](https://kind.sigs.k8s.io) | Des grappes Kubernetes locales, dans des conteneurs |
| [k9s](https://k9scli.io) | Une interface en terminal pour piloter Kubernetes |
| [Nomad](https://developer.hashicorp.com/nomad) | Orchestrateur de conteneurs et d'applications de HashiCorp |
| [Dapr](https://dapr.io) | Briques réutilisables pour les applications distribuées |

## Réseau, maillage de services et sécurité

| Projet | Rôle |
|--------|------|
| [Traefik](https://traefik.io/traefik/) | Proxy inverse et répartiteur de charge, configuré automatiquement |
| [Caddy](https://caddyserver.com) | Serveur web avec HTTPS automatique |
| [CoreDNS](https://coredns.io) | Le serveur DNS de Kubernetes |
| [Consul](https://developer.hashicorp.com/consul) | Découverte de services et maillage de services |
| [Istio](https://istio.io) | Maillage de services : son plan de contrôle est en Go, son proxy (Envoy) en C++ |
| [Linkerd](https://linkerd.io) | Maillage de services : plan de contrôle en Go, proxy en Rust |
| [Cilium](https://cilium.io) | Réseau et sécurité pour Kubernetes, avec eBPF |
| [Vault](https://developer.hashicorp.com/vault) | Gestion des secrets et chiffrement |

## Infrastructure et déploiement

| Projet | Rôle |
|--------|------|
| [Terraform](https://developer.hashicorp.com/terraform) | L'infrastructure décrite par du code |
| [Argo CD](https://argo-cd.readthedocs.io) | Déploiement continu « GitOps » sur Kubernetes |
| [Flux](https://fluxcd.io) | Déploiement continu « GitOps » sur Kubernetes |
| [etcd](https://etcd.io) | Base clé-valeur distribuée, cœur de Kubernetes |

## Observabilité

| Projet | Rôle |
|--------|------|
| [Prometheus](https://prometheus.io) | Collecte et stockage de métriques, alertes |
| [Grafana](https://grafana.com/oss/grafana/) | Tableaux de bord : serveur en Go, interface en TypeScript |
| [Loki](https://grafana.com/oss/loki/) | Agrégation de journaux |
| [Thanos](https://thanos.io) | Stockage à long terme et haute disponibilité pour Prometheus |
| [Jaeger](https://www.jaegertracing.io) | Traçage distribué |
| [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) | Collecte et transformation des traces, métriques et journaux |

## Bases de données et stockage

| Projet | Rôle |
|--------|------|
| [CockroachDB](https://www.cockroachlabs.com) | Base SQL distribuée, compatible avec PostgreSQL |
| [TiDB](https://github.com/pingcap/tidb) | Base SQL distribuée, compatible avec MySQL |
| [Vitess](https://vitess.io) | Mise à l'échelle horizontale de MySQL, née chez YouTube |
| [MinIO](https://min.io) | Stockage d'objets compatible avec Amazon S3 |
| [NATS](https://nats.io) | Messagerie et flux d'événements |
| [Temporal](https://temporal.io) | Exécution durable de processus métier (*workflows*) |

## Outils du quotidien

| Projet | Rôle |
|--------|------|
| [Hugo](https://gohugo.io) | Générateur de sites statiques ; il produit le site de ce livre |
| [GitHub CLI](https://cli.github.com) | `gh`, la ligne de commande de GitHub |
| [Gitea](https://about.gitea.com) | Forge Git légère, auto-hébergée |
| [Mattermost](https://mattermost.com) | Messagerie d'équipe : serveur en Go, interface en TypeScript |
| [Ollama](https://ollama.com) | Exécution locale de modèles de langage |
| [Syncthing](https://syncthing.net) | Synchronisation de fichiers de pair à pair |
| [rclone](https://rclone.org) | Synchronisation avec des dizaines de stockages en ligne |
| [restic](https://restic.net) | Sauvegardes chiffrées et dédupliquées |
| [fzf](https://github.com/junegunn/fzf) | Recherche floue interactive dans le terminal |
| [lazygit](https://github.com/jesseduffield/lazygit) | Interface Git en terminal |

## Ce que ces projets ont en commun

Ces projets illustrent les points forts de Go mieux qu'un long discours :

- **ce sont souvent des services réseau** (proxys, serveurs DNS, bases
  distribuées) : les goroutines et la bibliothèque `net/http` y font
  merveille ;
- **ils se distribuent en un seul exécutable**, sans machine virtuelle ni
  bibliothèque partagée : `kubectl`, `terraform` ou `hugo` s'installent en
  copiant un fichier ;
- **ils sont compilés pour toutes les plateformes** depuis une même machine,
  grâce à la compilation croisée ;
- **ils sont développés par des équipes nombreuses**, parfois des milliers de
  contributeurs : la simplicité du langage et l'outillage commun (`gofmt`,
  `go vet`, `go test`) rendent leur code lisible par tous.

Plusieurs chapitres de ce livre s'inspirent directement de ces projets : la
ligne de commande avec cobra (utilisé par Kubernetes, Hugo et GitHub CLI), les
tests avec Testcontainers, ou les opérateurs Kubernetes.
