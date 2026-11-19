PANDOC_IMAGE := docker.io/pandoc/extra:3.14
QUARTO_IMAGE := ghcr.io/quarto-dev/quarto:1.11.9
QUARTO_DIR   := book/build/quarto
BOOK_MD      := book/build/book.md
BOOK_PDF     := book/golang-demystified.pdf
# Podman rootless : keep-id conserve l'UID de l'utilisateur dans le conteneur.
CONTAINER_USER := $(if $(shell docker --version 2>/dev/null | grep -i podman),--userns=keep-id,--user $(shell id -u):$(shell id -g))

.PHONY: all site serve pdf book-md quarto check clean

all: check site pdf

## site : génère le site statique dans docs/ (GitHub Pages)
site:
	cd site && hugo --minify --cleanDestinationDir

## serve : prévisualise le site sur http://localhost:1313/golang-demystified/
serve:
	cd site && hugo server --buildDrafts --renderToMemory

## book-md : assemble les pages en un seul Markdown, dans l'ordre du site
book-md:
	go run ./tools/mkbook -content site/content/docs -static site/static -out $(BOOK_MD)

## pdf : génère le livre PDF avec pandoc + eisvogel (via Docker)
pdf: book-md
	docker run --rm --volume "$(CURDIR):/data" --workdir /data $(CONTAINER_USER) \
		$(PANDOC_IMAGE) $(BOOK_MD) --from gfm --metadata-file book/metadata.yaml --lua-filter book/callouts.lua \
		--template eisvogel --syntax-highlighting=idiomatic --pdf-engine=xelatex --top-level-division=part \
		--output $(BOOK_PDF)

## quarto : génère le livre avec Quarto (PDF via Typst, et EPUB) dans book/build/
quarto:
	go run ./tools/mkbook -format quarto -config book/quarto/_quarto.yml -out $(QUARTO_DIR)
	cp -r book/assets book/quarto/typst-show.typ $(QUARTO_DIR)/
	docker run --rm --volume "$(CURDIR):/data" --workdir /data/$(QUARTO_DIR) $(CONTAINER_USER) -e HOME=/tmp \
		$(QUARTO_IMAGE) quarto render
	cp $(QUARTO_DIR)/_book/Golang-Demystified.pdf book/build/golang-demystified-quarto.pdf
	cp $(QUARTO_DIR)/_book/Golang-Demystified.epub book/build/golang-demystified.epub

## check : vérifie et teste les outils et les exemples du livre
check:
	test -z "$$(gofmt -l tools examples)"
	go vet ./...
	go test ./...

clean:
	rm -rf book/build site/public site/resources site/.hugo_build.lock
