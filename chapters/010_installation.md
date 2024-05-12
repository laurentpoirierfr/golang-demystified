---
marp: true
theme: gaia
markdown.marp.enableHtml: true
paginate: true
---
# Installation

---
### Téléchargement de Go :

- Rendez-vous sur le site officiel de Go à l'adresse https://golang.org/dl/.
- Téléchargez la version appropriée pour votre système d'exploitation (Linux, macOS, Windows, etc.).
- Une fois le téléchargement terminé, extrayez l'archive dans le répertoire **$HOME/Community/go**.
---
### Configuration des variables d'environnement :

- Ouvrez ou créez le fichier de profil de votre shell (**~/.bash_profile**, **~/.bashrc**, **~/.zshrc**, etc.).
- Ajoutez les lignes suivantes pour définir les variables d'environnement **GOROOT**, **GOPATH**, et mettre les binaires de **GOPATH/bin** dans le **PATH** :

```bash
export GOROOT=$HOME/Community/go
export GOPATH=$HOME/Community/gopath
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

---
### Vérification de l'installation :

- Ouvrez un nouveau terminal.
- Tapez **go version** pour vérifier que Go est correctement installé et configuré.

```bash
elfeo@elfeo:~/Project/golang-demystified$ go version
go version go1.22.2 linux/amd64
```
Avec ces configurations, vous pourrez utiliser Go et ses outils associés de manière fluide, en ayant accès aux binaires des packages installés dans **$HOME/Community/gopath/bin**.

---
### Téléchargement de Visual Studio Code :

- Rendez-vous sur le site officiel de Visual Studio Code à l'adresse https://code.visualstudio.com/.
- Téléchargez la version appropriée pour votre système d'exploitation (Linux, macOS, Windows, etc.).
- Suivez les instructions pour installer Visual Studio Code sur votre système.

---
### Installation d'extensions pour Go :

![help](assets/010_installation_000.png)

- Recherchez "Go" dans la barre de recherche des extensions.
- Installez l'extension **"Go"** proposée par Microsoft, en cliquant sur le bouton **"Installer"**.

---
### Configuration de l'environnement Go :

- Ouvrez les paramètres de Visual Studio Code en appuyant sur Ctrl+, ou en sélectionnant **"Fichier" > "Préférences" > "Paramètres"**.
- Recherchez "go.gopath" dans la barre de recherche des paramètres.
- Spécifiez le chemin du répertoire gopath que vous avez configuré précédemment, par exemple **"$HOME/Community/gopath"**.
- Enregistrez les modifications.

---
### Vérification de l'installation :

- Redémarrez Visual Studio Code pour appliquer les paramètres.
- Créez un nouveau fichier Go ou ouvrez un projet existant.
- Visual Studio Code devrait automatiquement détecter que c'est un projet Go et proposer des suggestions d'installation d'outils supplémentaires si nécessaire.
- Vous êtes maintenant prêt à coder en Go avec Visual Studio Code !

Avec Visual Studio Code et l'extension Go, vous bénéficierez d'un environnement de développement puissant et convivial pour vos projets en Go.








