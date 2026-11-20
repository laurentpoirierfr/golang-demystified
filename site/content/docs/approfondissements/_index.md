---
title: "Approfondissements"
linkTitle: "V - Approfondissements"
weight: 50
---

Des chapitres construits autour d'exemples complets et testés, qui mettent en
œuvre ensemble ce que les parties précédentes ont présenté séparément. Ils
suivent un parcours : on construit un service, on le rend fiable, on le relie
à d'autres, on le livre, puis on emmène Go hors du serveur. Chaque chapitre
peut aussi se lire seul.

**Les fondations pratiques**

1. [Go idiomatique](go-idiomatique/) : options fonctionnelles, middlewares,
   injection de dépendances, anti-motifs, et une grille de relecture.
2. [Concurrence en pratique](concurrence/) : pipelines, pools de workers,
   `errgroup`, limitation de débit, traitement de flux.
3. [Templates](templates/) : générer du texte et du HTML, en toute sécurité.

**Construire un service**

4. [Une API web complète](api-web/) : contrat OpenAPI, routage, validation,
   stockage, tests.
5. [Ligne de commande](cli-cobra-viper/) : le client de cette API, avec
   cobra et viper.
6. [Accès aux données](sqlc-migrate/) : SQL typé avec sqlc, migrations avec
   golang-migrate.
7. [Architecture hexagonale](architecture-hexagonale/) : séparer le domaine
   de l'infrastructure quand la logique métier grandit.

**Rendre le service fiable, et comprendre le runtime**

8. [Sécurité](securite/) : OpenID Connect et JWT, TLS, secrets, validation
   des entrées.
9. [Observabilité](observabilite/) : traces, métriques et journaux avec
   OpenTelemetry.
10. [Profilage et performance](performance/) : mesurer avant d'optimiser.
11. [Les coulisses de Go](coulisses/) : réflexion, `unsafe`, cgo,
    l'ordonnanceur et le ramasse-miettes.

**Relier les services**

12. [gRPC et Connect](grpc-connect/) : des API à contrat Protobuf.
13. [Temps réel](temps-reel/) : WebSocket et Server-Sent Events.
14. [Architecture événementielle](architecture-evenementielle/) : Watermill,
    NATS, outbox et saga.

**Livrer et exploiter**

15. [CI/CD](ci-cd/) : intégration continue, analyse, publication.
16. [Kubernetes](kubernetes/) : écrire un opérateur.

**Au-delà du serveur**

17. [WebAssembly et TinyGo](webassembly-tinygo/) : le même code dans le
    navigateur, dans un greffon et sur un microcontrôleur.
18. [Go et l'IA](go-ia/) : modèles de langage, outils et serveur MCP.
