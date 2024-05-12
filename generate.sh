#!/bin/bash

# Répertoire contenant les fichiers Markdown
repertoire="./chapters"

# Répertoire de destination pour les fichiers HTML
cible="./docs"

# Vérifier si le répertoire de destination existe, sinon le créer
mkdir -p "$cible"

# Parcours des fichiers Markdown dans le répertoire spécifié
for fichier_md in "$repertoire"/*.md; do
    # Vérification si le fichier est un fichier Markdown
    if [ -f "$fichier_md" ]; then
        # Extraire le nom du fichier (sans l'extension)
        nom_fichier=$(basename "$fichier_md" .md)
        
        # Appliquer la commande marp pour convertir en HTML
        # marp-cli --html "$fichier_md" -o "$cible/$nom_fichier.html"
        # echo "Conversion de $nom_fichier.md en $cible/$nom_fichier.html terminée."

        docker run -e MARP_USER="$(id -u):$(id -g)" --rm -v $PWD:/home/marp/app/ -e LANG=$LANG marpteam/marp-cli $fichier_md --html
        
    fi
done

mv $repertoire/*.html $cible

echo "Conversion de tous les fichiers Markdown terminée."
