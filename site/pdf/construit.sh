#!/usr/bin/env bash
# Construit le livre PDF au rendu du site :
#   1. Hugo, dans l'environnement « pdf » (config/pdf/hugo.yaml), produit tout
#      le livre sur une seule page, /livre/ (layouts/home.livre.html) ;
#   2. Paged.js la pagine : couverture, numéros de page, titres courants,
#      table des matières et renvois numérotés (assets/styles/livre.css) ;
#   3. Chromium l'imprime (pdf/imprime.js, dans un conteneur).
#
# Usage, depuis n'importe où :
#   pdf/construit.sh public/mon-livre.pdf
#
# Docker ou Podman requis. Le site Hugo est le répertoire parent de pdf/,
# sauf si SITE en désigne un autre.
set -euo pipefail

ici=$(cd "$(dirname "$0")" && pwd)
site=$(cd "${SITE:-$ici/..}" && pwd)
sortie=$(realpath -m "${1:?usage : pdf/construit.sh <fichier.pdf>}")

image=docker.io/zenika/alpine-chrome:124-with-puppeteer
paged_version=0.4.3
paged_sha256=f44e35a6d5106d819c3415fd38591de04d3834343c14e50a562b84257e204692

docker=$(command -v docker || command -v podman) || {
	echo "échec : ni docker ni podman" >&2
	exit 1
}

# Paged.js, téléchargé une fois et vérifié par sa somme de contrôle.
cache=${XDG_CACHE_HOME:-$HOME/.cache}/livre-pdf
paged=$cache/paged-$paged_version.polyfill.min.js
if [ ! -f "$paged" ]; then
	mkdir -p "$cache"
	curl -fsSL -o "$paged.tmp" "https://unpkg.com/pagedjs@$paged_version/dist/paged.polyfill.min.js"
	echo "$paged_sha256  $paged.tmp" | sha256sum --check --quiet
	mv "$paged.tmp" "$paged"
fi

travail=$(mktemp -d)
trap 'rm -rf "$travail"' EXIT

# Le site, servi par imprime.js sur localhost:8123, avec ses polices et Paged.js.
version=$(git -C "$site" describe --tags --always 2>/dev/null || true)
(cd "$site" && HUGO_PARAMS_VERSION="$version" hugo --environment pdf --minify --quiet \
	--baseURL http://localhost:8123/ --destination "$travail/site")
cp -r "$ici/polices" "$travail/site/polices"
cp "$paged" "$travail/site/paged.polyfill.min.js"

# Le conteneur écrit le PDF sous son propre utilisateur.
mkdir -p "$travail/sortie" && chmod 777 "$travail/sortie"
"$docker" run --rm -e HOME=/tmp -e FONTCONFIG_FILE=/usr/src/app/livre/fonts.conf \
	--volume "$travail/site:/site:ro" \
	--volume "$ici:/usr/src/app/livre:ro" \
	--volume "$travail/sortie:/sortie" \
	"$image" node livre/imprime.js

mkdir -p "$(dirname "$sortie")"
cp "$travail/sortie/livre.pdf" "$sortie"
echo "écrit : $sortie"
