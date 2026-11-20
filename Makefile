QUARTO_IMAGE := ghcr.io/quarto-dev/quarto:1.11.9
QUARTO_DIR   := book/build/quarto
BOOK_PDF     := book/golang-demystified.pdf
BOOK_EPUB    := book/golang-demystified.epub
AIDE_DIR     := book/build/aide-memoire
AIDE_PDF     := book/aide-memoire-go.pdf
# Podman rootless : keep-id conserve l'UID de l'utilisateur dans le conteneur.
CONTAINER_USER := $(if $(shell docker --version 2>/dev/null | grep -i podman),--userns=keep-id,--user $(shell id -u):$(shell id -g))
QUARTO = docker run --rm --volume "$(CURDIR):/data" --workdir /data/$(QUARTO_DIR) \
	$(CONTAINER_USER) -e HOME=/tmp $(QUARTO_IMAGE)

# Diapositives : l'image Marp démarre en root et prend l'identité MARP_USER.
# Avec Podman rootless, keep-id fait correspondre cette identité à la nôtre.
SLIDES_DIR := slides/build
MARP_IMAGE := docker.io/marpteam/marp-cli:v4.5.1
MARP_USERNS := $(if $(shell docker --version 2>/dev/null | grep -i podman),--userns=keep-id,)
MARP = docker run --rm $(MARP_USERNS) -e MARP_USER="$(shell id -u):$(shell id -g)" \
	--volume "$(CURDIR)/$(SLIDES_DIR):/home/marp/app" \
	--volume "$(CURDIR)/slides/theme:/theme:ro" \
	--volume "$(CURDIR)/book/quarto/fonts:/home/marp/.fonts:ro" \
	$(MARP_IMAGE)

.PHONY: all site publication serve book aide-memoire pdf slides slides-verif check vuln lint test-integration sqlc buf wasm wasm-tinygo rust-et-go numerote generate clean

all: check book slides publication

## site : génère le site statique dans site/public
site:
	cd site && hugo --minify --cleanDestinationDir

## publication : le site, avec le PDF, l'EPUB et les diapositives (après make
## book et make slides)
publication: site
	cp $(BOOK_PDF) $(BOOK_EPUB) $(AIDE_PDF) site/public/
	cp -r $(SLIDES_DIR)/site site/public/diapositives

## slides : un diaporama Marp par rubrique et par chapitre, en HTML et en PDF
slides:
	rm -rf $(SLIDES_DIR)
	go run ./tools/mkslides -content site/content/docs -out $(SLIDES_DIR)/md
	mkdir -p $(SLIDES_DIR)/md/polices
	cp book/quarto/fonts/SourceSans3-*.otf book/quarto/fonts/Go-Mono.ttf book/quarto/fonts/Go-Mono-Bold.ttf $(SLIDES_DIR)/md/polices/
	$(MARP) --input-dir md --output site --allow-local-files --theme-set /theme/golang.css
	$(MARP) --input-dir md --output site --allow-local-files --pdf --theme-set /theme/golang.css
	cp -r $(SLIDES_DIR)/md/polices $(SLIDES_DIR)/md/index.html $(SLIDES_DIR)/site/

## slides-verif : mesure le rendu de chaque diapositive dans Chromium, et
## échoue si un contenu déborde (après make slides)
PUPPETEER_IMAGE := docker.io/zenika/alpine-chrome:124-with-puppeteer
slides-verif:
	docker run --rm --volume "$(CURDIR)/slides/verif:/usr/src/app/test:ro" \
		--volume "$(CURDIR)/$(SLIDES_DIR)/site:/diapos:ro" $(PUPPETEER_IMAGE) node test/debordements.js

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
	$(MAKE) aide-memoire

## aide-memoire : le PDF séparé de l'aide-mémoire (annexe), en deux colonnes
aide-memoire:
	go run ./tools/mkbook -content site/content/docs -static site/static \
		-config book/quarto/_quarto.yml -out $(QUARTO_DIR) >/dev/null
	rm -rf $(AIDE_DIR) && mkdir -p $(AIDE_DIR)
	cp -r book/quarto/fonts book/aide-memoire/aide-memoire.typ $(AIDE_DIR)/
	# Le contenu du chapitre, sans son titre ni son introduction.
	{ cat book/aide-memoire/entete.yml; sed '1,/^## /{/^## /!d}' $(QUARTO_DIR)/*-aide-memoire.qmd; } \
		> $(AIDE_DIR)/aide-memoire-go.qmd
	docker run --rm --volume "$(CURDIR):/data" --workdir /data/$(AIDE_DIR) \
		$(CONTAINER_USER) -e HOME=/tmp $(QUARTO_IMAGE) quarto render aide-memoire-go.qmd --to typst
	cp $(AIDE_DIR)/aide-memoire-go.pdf $(AIDE_PDF)

## pdf : alias de book
pdf: book

## check : vérifie et teste les outils et les exemples du livre (sans conteneur)
check:
	test -z "$$(gofmt -l tools examples)"
	go run ./tools/numerote -verifie
	go vet ./...
	go test -short ./...

## lint : golangci-lint, par son image Docker (configuration : .golangci.yml)
GOLANGCI_IMAGE := docker.io/golangci/golangci-lint:v2.15.0
lint:
	docker run --rm --volume "$(CURDIR):/app" --workdir /app $(CONTAINER_USER) -e HOME=/tmp \
		$(GOLANGCI_IMAGE) golangci-lint run ./...

## vuln : recherche les vulnérabilités connues dans le code et ses dépendances
vuln:
	go tool govulncheck ./...

# Avec Podman, Testcontainers passe par l'API compatible Docker de Podman
# (systemctl --user start podman.socket), sans le conteneur « Ryuk ».
PODMAN_SOCKET := $(XDG_RUNTIME_DIR)/podman/podman.sock
TC_ENV := $(if $(shell docker --version 2>/dev/null | grep -i podman),DOCKER_HOST=unix://$(PODMAN_SOCKET) TESTCONTAINERS_RYUK_DISABLED=true)

## test-integration : tests d'intégration avec Testcontainers (Docker ou Podman)
test-integration:
	$(TC_ENV) go test -count=1 -run Postgres -v ./examples/...

## sqlc : régénère le code d'accès aux données de l'exemple « catalogue »
SQLC_IMAGE := docker.io/sqlc/sqlc:1.31.1
sqlc:
	docker run --rm --volume "$(CURDIR)/examples/approfondissements/catalogue:/src" --workdir /src \
		$(CONTAINER_USER) $(SQLC_IMAGE) generate

## buf : vérifie le contrat Protobuf et régénère le code gRPC/Connect
BUF_IMAGE := docker.io/bufbuild/buf:1.75.0
BUF := docker run --rm --volume "$(CURDIR)/examples/approfondissements/grpc:/workspace" --workdir /workspace --env BUF_CACHE_DIR=/tmp/buf \
	$(CONTAINER_USER) $(BUF_IMAGE)
buf:
	$(BUF) lint
	$(BUF) generate

## wasm : la démonstration WebAssembly dans examples/approfondissements/wasm/web
##   (puis : cd examples/approfondissements/wasm && go run ./cmd/site)
wasm:
	cd examples/approfondissements/wasm && GOOS=js GOARCH=wasm go build -o web/analyse.wasm ./cmd/navigateur
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" examples/approfondissements/wasm/web/

## wasm-tinygo : le greffon WASI compilé par TinyGo, puis essayé par l'hôte wazero
TINYGO_IMAGE := docker.io/tinygo/tinygo:0.44.0
TINYGO_SORTIE := examples/approfondissements/wasm/build
TINYGO = docker run --rm --volume "$(CURDIR):/src" --volume "$(shell go env GOMODCACHE):/go/pkg/mod:ro" \
	--workdir /src $(CONTAINER_USER) -e HOME=/tmp -e GOCACHE=/tmp/cache -e GOMODCACHE=/go/pkg/mod \
	-e GOFLAGS="-mod=mod -buildvcs=false" $(TINYGO_IMAGE)
wasm-tinygo:
	mkdir -p $(TINYGO_SORTIE)
	$(TINYGO) tinygo build -target=wasip1 -buildmode=c-shared -no-debug \
		-o $(TINYGO_SORTIE)/greffon-tinygo.wasm ./examples/approfondissements/wasm/cmd/greffon
	GREFFON_TINYGO=$(CURDIR)/$(TINYGO_SORTIE)/greffon-tinygo.wasm \
		go test -count=1 -run TinyGo -v ./examples/approfondissements/wasm/internal/hote

## rust-et-go : compile et teste la version Rust de l'article « Rust et Go »,
##   et vérifie que la course de données y est refusée à la compilation
RUST_IMAGE := docker.io/library/rust:1.99.1-slim
rust-et-go:
	go test -count=1 ./examples/decouvrir/rust-et-go/...
	docker run --rm $(MARP_USERNS) --user "$(shell id -u):$(shell id -g)" -e HOME=/tmp -e CARGO_HOME=/tmp/cargo \
		-e CARGO_TARGET_DIR=/tmp/cible --volume "$(CURDIR)/examples/decouvrir/rust-et-go/rust:/orig:ro" \
		$(RUST_IMAGE) sh -c 'cp -r /orig /tmp/p && cd /tmp/p/mots && cargo test --release && \
		if rustc --edition 2024 -o /tmp/course ../course/main.rs 2>/dev/null; \
		then echo "la course aurait dû être refusée"; exit 1; \
		else echo "course refusée par le compilateur Rust, comme attendu"; fi'

## numerote : numérote les parties (I, II…) et les chapitres du menu du site,
##   d'après leur poids (weight)
numerote:
	go run ./tools/numerote

## generate : régénère tout le code généré (oapi-codegen, sqlc, buf)
generate: sqlc buf
	go generate ./...

clean:
	rm -rf book/build site/public site/resources site/.hugo_build.lock
