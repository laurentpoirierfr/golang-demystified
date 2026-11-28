---
title: "VI. Exercices corrigés"
weight: 60
---

# Partie VI — Exercices corrigés

Les chapitres des parties II et III se terminent par trois exercices, de
difficulté croissante (★, ★★, ★★★). Chacun n'utilise que ce que son chapitre
enseigne : on peut les faire au fil de la lecture.

Les solutions sont dans le dépôt, dans
[`examples/exercices`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/exercices),
une par chapitre, avec leurs tests. Les tests servent de **spécification** :

```bash
mkdir mes-exercices && cd mes-exercices
go mod init exemple.fr/exercices
cp chemin/du/depot/examples/exercices/langage/controle/exercices_test.go .
# écrivez les fonctions demandées dans un fichier .go, package controle
go test
```

Tant que les tests échouent, leurs messages disent ce qui ne va pas. Les
solutions sont vérifiées à chaque modification du dépôt, avec le détecteur de
courses.

## Le langage

- **II.1** [Variables, constantes et types](../langage/bases/#exercices) : Celsius et Fahrenheit ★ · Des unités avec iota ★★ · Détecter un dépassement ★★★
- **II.2** [Les opérateurs](../langage/operateurs/#exercices) : Pair ou impair ★ · Des droits en bits ★★ · Compter les bits ★★★
- **II.3** [Contrôle de flux](../langage/controle/#exercices) : FizzBuzz ★ · Une mention ★★ · La suite de Syracuse ★★★
- **II.4** [Tableaux, slices et maps](../langage/collections/#exercices) : Inverser une slice ★ · Compter des mots ★★ · Dédoublonner en gardant l'ordre ★★★
- **II.5** [Fonctions](../langage/fonctions/#exercices) : Une moyenne variadique ★ · Un compteur ★★ · Mémoïser ★★★
- **II.6** [Structures et méthodes](../langage/structures/#exercices) : Un rectangle ★ · Une pile ★★ · Un compte avec historique ★★★
- **II.7** [Interfaces](../langage/interfaces/#exercices) : Des formes ★ · Une durée lisible ★★ · Un lecteur en majuscules ★★★
- **II.8** [Gestion des erreurs](../langage/erreurs/#exercices) : Envelopper une erreur ★ · Un type d'erreur ★★ · Une panique devenue erreur ★★★
- **II.9** [Génériques](../langage/generiques/#exercices) : Un maximum ★ · Transformer et filtrer ★★ · Un ensemble générique ★★★
- **II.10** [Pointeurs et mémoire](../langage/pointeurs/#exercices) : Échanger deux variables ★ · Inverser une liste chaînée ★★ · Un arbre binaire de recherche ★★★
- **II.11** [Goroutines et channels](../langage/concurrence/#exercices) : Une somme en parallèle ★ · Un pipeline ★★ · Un groupe de travailleurs ★★★
- **II.12** [Packages et modules](../langage/packages/#exercices) : Encapsuler ★ · Un registre ★★ · Une version fixée à la compilation ★★★

## La bibliothèque standard

- **III.1** [Texte](../bibliotheque/texte/#exercices) : Des initiales ★ · Un palindrome ★★ · Aligner des colonnes ★★★
- **III.2** [Collections](../bibliotheque/collections/#exercices) : Trier sur deux critères ★ · Regrouper ★★ · Combiner des itérateurs ★★★
- **III.3** [Fichiers et flux](../bibliotheque/fichiers/#exercices) : Compter les lignes ★ · Parcourir un système de fichiers ★★ · Écrire un fichier de façon atomique ★★★
- **III.4** [Le temps](../bibliotheque/temps/#exercices) : Un âge ★ · Des jours ouvrés ★★ · Une date française, en UTC ★★★
- **III.5** [JSON](../bibliotheque/json/#exercices) : Des tags ★ · Décoder strictement ★★ · Un encodage personnalisé ★★★
- **III.6** [HTTP](../bibliotheque/http/#exercices) : Une route avec un paramètre ★ · Une petite API JSON ★★ · Un middleware d'authentification ★★★
- **III.7** [Journalisation](../bibliotheque/slog/#exercices) : Un journal JSON ★ · Masquer un secret ★★ · Un handler qui compte ★★★
- **III.8** [Annulation et délais](../bibliotheque/context/#exercices) : Une attente annulable ★ · Une valeur typée ★★ · Le premier qui réussit ★★★
- **III.9** [Tests](../bibliotheque/testing/#exercices) : Un test de table ★ · Un test de fuzzing ★★ · Tester le temps ★★★
- **III.10** [Tour d'horizon](../bibliotheque/divers/#exercices) : Extraire des adresses ★ · Totaliser un CSV ★★ · Signer un message ★★★
