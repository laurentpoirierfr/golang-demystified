---
title: "III. La bibliothèque standard"
bookCollapseSection: true
weight: 30
---

# Partie III — La bibliothèque standard

La bibliothèque standard de Go couvre l'essentiel des besoins d'un programme :
texte, collections, fichiers, temps, JSON, HTTP, journalisation, annulation et
tests. On peut construire un service web complet sans aucune dépendance
externe.

<!-- sommaire -->

- **III.1** [Texte](texte/) : formater avec fmt, manipuler avec strings, convertir avec strconv, et l'UTF-8.
- **III.2** [Collections](collections/) : trier, chercher et modifier avec slices et maps, itérateurs et cmp.
- **III.3** [Fichiers et flux](fichiers/) : io.Reader et io.Writer, fichiers, lecture ligne par ligne, io/fs, embed et os.Root.
- **III.4** [Le temps](temps/) : instants, durées, formats, fuseaux horaires, timers et tickers.
- **III.5** [JSON](json/) : encoder, décoder, personnaliser, traiter des flux, et encoding/json/v2.
- **III.6** [HTTP](http/) : serveur, routage, middlewares, client et tests avec net/http seul.
- **III.7** [Journalisation](slog/) : des journaux structurés, contextualisés, sans données sensibles, au niveau réglable à chaud.
- **III.8** [Annulation et délais](context/) : délais, annulation et valeurs transmis d'un appel à l'autre.
- **III.9** [Tests](testing/) : tests de table, exemples, benchmarks, fuzzing, synctest et Testcontainers.
- **III.10** [Tour d'horizon](divers/) : uuid, hasard, cryptographie, CSV, regexp, flag, sync.Once et database/sql.

<!-- /sommaire -->
