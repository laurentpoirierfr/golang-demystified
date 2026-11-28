---
title: "I.4 Installation"
weight: 20
---

# Chapitre I.4 — Installation

## Installer Go

Les distributions officielles sont disponibles sur
[go.dev/dl](https://go.dev/dl/). Une installation de Go est autonome : elle
contient le compilateur, la bibliothèque standard et tous les outils.

### Linux

Téléchargez l'archive, puis extrayez-la dans `/usr/local` en remplaçant toute
installation précédente :

```bash
curl -LO https://go.dev/dl/go1.27.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.27.4.linux-amd64.tar.gz
```

Ajoutez ensuite le répertoire des binaires de Go, ainsi que celui des outils
que vous installerez, à votre `PATH` (dans `~/.bashrc`, `~/.zshrc`…) :

```bash
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
```

Pour fish, la commande équivalente est
`fish_add_path /usr/local/go/bin $HOME/go/bin`.

### macOS

Téléchargez et lancez l'installeur `.pkg` depuis
[go.dev/dl](https://go.dev/dl/), ou utilisez [Homebrew](https://brew.sh) :

```bash
brew install go
```

> [!NOTE]
> **Go 1.27** nécessite macOS 13 Ventura ou une version plus récente.

### Windows

Lancez l'installeur `.msi`, ou utilisez [winget](https://learn.microsoft.com/windows/package-manager/winget/) :

```powershell
winget install GoLang.Go
```

L'installeur configure le `PATH` automatiquement. Ouvrez un nouveau terminal
pour qu'il soit pris en compte.

## Vérifier l'installation

```bash
$ go version
go version go1.27.4 linux/amd64
```

La commande `go env` affiche la configuration de l'outil. Les variables les plus
utiles sont décrites dans le chapitre sur l'outil `go`.

> [!TIP]
> Il n'est plus nécessaire de définir `GOROOT` ni `GOPATH`. Depuis les modules
> (Go 1.11) et leur activation par défaut (Go 1.16), un projet peut se trouver
> n'importe où sur le disque. `GOPATH` vaut `$HOME/go` par défaut et sert
> uniquement de cache et de répertoire d'installation des outils.

## Gérer plusieurs versions

Depuis **Go 1.21**, la commande `go` sait télécharger et utiliser la version
de Go dont un projet a besoin. Le fichier `go.mod` d'un module déclare la
version minimale requise :

```go
module example.com/monprojet

go 1.27
```

Si la version installée est plus ancienne que celle demandée, `go` télécharge
automatiquement la bonne chaîne de compilation (*toolchain*) et l'utilise. Ce
comportement est contrôlé par la variable `GOTOOLCHAIN` :

| Valeur                | Effet                                                         |
|-----------------------|---------------------------------------------------------------|
| `auto` (défaut)       | Utilise la version locale, ou télécharge celle demandée       |
| `local`               | Utilise toujours la version installée, sans téléchargement    |
| `go1.26.3`            | Force une version précise                                     |
| `go1.26.3+auto`       | Utilise au minimum cette version, et plus récente si besoin   |

Pour tester un projet avec une autre version de Go :

```bash
GOTOOLCHAIN=go1.26.3 go test ./...
```

## Configurer un éditeur

Go fournit un serveur de langage officiel, **[gopls](https://pkg.go.dev/golang.org/x/tools/gopls)**, utilisé par la plupart
des éditeurs : complétion, navigation dans le code, renommage, diagnostics et
corrections automatiques.

- **[Visual Studio Code](https://code.visualstudio.com)** : installez l'extension **Go** (éditée par l'équipe
  Go). À l'ouverture du premier fichier `.go`, elle propose d'installer gopls et
  les outils nécessaires.
- **[GoLand](https://www.jetbrains.com/go/)** (JetBrains) : un IDE dédié à Go, prêt à l'emploi.
- **Neovim, Helix, Zed, Emacs…** : configurez le client LSP pour utiliser
  `gopls`, que vous pouvez installer avec :

```bash
go install golang.org/x/tools/gopls@latest
```

Pensez à activer le formatage automatique à l'enregistrement : avec Go, le
formatage n'est pas une question de goût, c'est celui de `gofmt`.
