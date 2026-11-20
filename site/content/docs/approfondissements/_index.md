---
title: "Approfondissements"
weight: 50
---

Des chapitres construits autour d'exemples complets et testés, qui mettent en
œuvre ensemble ce que les parties précédentes ont présenté séparément. Ils
suivent un parcours : on construit un service, on le rend fiable, on le relie
à d'autres, on le livre, puis on emmène Go hors du serveur. Chaque chapitre
peut aussi se lire seul.

**Les fondations pratiques**

1. [Concurrence en pratique](concurrence/) : pipelines, pools de workers,
   `errgroup`, limitation de débit, traitement de flux.
2. [Templates](templates/) : générer du texte et du HTML, en toute sécurité.

**Construire un service**

3. [Une API web complète](api-web/) : contrat OpenAPI, routage, validation,
   stockage, tests.
4. [Ligne de commande](cli-cobra-viper/) : le client de cette API, avec
   cobra et viper.
5. [Accès aux données](sqlc-migrate/) : SQL typé avec sqlc, migrations avec
   golang-migrate.
6. [Architecture hexagonale](architecture-hexagonale/) : séparer le domaine
   de l'infrastructure quand la logique métier grandit.

**Rendre le service fiable, et comprendre le runtime**

7. [Sécurité](securite/) : OpenID Connect et JWT, TLS, secrets, validation
   des entrées.
8. [Observabilité](observabilite/) : traces, métriques et journaux avec
   OpenTelemetry.
9. [Profilage et performance](performance/) : mesurer avant d'optimiser.
10. [Les coulisses de Go](coulisses/) : réflexion, `unsafe`, cgo,
    l'ordonnanceur et le ramasse-miettes.

**Relier les services**

11. [gRPC et Connect](grpc-connect/) : des API à contrat Protobuf.
12. [Temps réel](temps-reel/) : WebSocket et Server-Sent Events.
13. [Architecture événementielle](architecture-evenementielle/) : Watermill,
    NATS, outbox et saga.

**Livrer et exploiter**

14. [CI/CD](ci-cd/) : intégration continue, analyse, publication.
15. [Kubernetes](kubernetes/) : écrire un opérateur.

**Au-delà du serveur**

16. [WebAssembly et TinyGo](webassembly-tinygo/) : le même code dans le
    navigateur, dans un greffon et sur un microcontrôleur.
17. [Go et l'IA](go-ia/) : modèles de langage, outils et serveur MCP.
