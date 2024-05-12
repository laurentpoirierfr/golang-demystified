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

Avec ces configurations, vous pourrez utiliser Go et ses outils associés de manière fluide, en ayant accès aux binaires des packages installés dans **$HOME/Community/gopath/bin**.

￼
￼
￼
￼
