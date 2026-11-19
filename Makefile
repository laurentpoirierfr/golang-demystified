QUARTO_IMAGE := ghcr.io/quarto-dev/quarto:1.11.9
QUARTO_DIR   := book/build/quarto
BOOK_PDF     := book/golang-demystified.pdf
BOOK_EPUB    := book/golang-demystified.epub
# Podman rootless : keep-id conserve l'UID de l'utilisateur dans le conteneur.
CONTAINER_USER := $(if $(shell docker --version 2>/dev/null | grep -i podman),--userns=keep-id,--user $(shell id -u):$(shell id -g))
QUARTO = docker run --rm --volume "$(CURDIR):/data" --workdir /data/$(QUARTO_DIR) \
	$(CONTAINER_USER) -e HOME=/tmp $(QUARTO_IMAGE)

.PHONY: all site serve book pdf check test-integration clean

all: check site book

## site : génère le site statique dans docs/ (GitHub Pages)
site:
	cd site && hugo --minify --cleanDestinationDir

## serve : prévisualise le site sur http://localhost:1313/golang-demystified/
serve:
	cd site && hugo server --buildDrafts --renderToMemory

## book : génère le livre en PDF (Quarto + Typst) et en EPUB
book:
	go run ./tools/mkbook -content site/content/docs -static site/static \
		-config book/quarto/_quarto.yml -out $(QUARTO_DIR)
	cp -r book/assets book/quarto/fonts book/quarto/*.typ $(QUARTO_DIR)/
	$(QUARTO) quarto typst compile couverture-epub.typ assets/couverture.png \
		--font-path fonts --ppi 110
	$(QUARTO) quarto render
	cp $(QUARTO_DIR)/_book/Golang-Demystified.pdf $(BOOK_PDF)
	cp $(QUARTO_DIR)/_book/Golang-Demystified.epub $(BOOK_EPUB)
	cp $(QUARTO_DIR)/assets/couverture.png site/static/images/couverture.png

## pdf : alias de book
pdf: book

## check : vérifie et teste les outils et les exemples du livre (sans conteneur)
check:
	test -z "$$(gofmt -l tools examples)"
	go vet ./...
	go test -short ./...

# Avec Podman, Testcontainers passe par l'API compatible Docker de Podman
# (systemctl --user start podman.socket), sans le conteneur « Ryuk ».
PODMAN_SOCKET := $(XDG_RUNTIME_DIR)/podman/podman.sock
TC_ENV := $(if $(shell docker --version 2>/dev/null | grep -i podman),DOCKER_HOST=unix://$(PODMAN_SOCKET) TESTCONTAINERS_RYUK_DISABLED=true)

## test-integration : tests d'intégration avec Testcontainers (Docker ou Podman)
test-integration:
	$(TC_ENV) go test -count=1 -run Postgres -v ./examples/...

clean:
	rm -rf book/build site/public site/resources site/.hugo_build.lock
