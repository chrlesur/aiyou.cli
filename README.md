
# aiyou.cli

aiyou.cli est une interface en ligne de commande pour interagir avec les assistants AI.YOU directement depuis votre terminal, compatible avec Windows, Linux et macOS.

## Caractéristiques

- Chat en mode interactif avec les assistants AI.YOU
- Envoi de messages uniques à l'assistant
- Support pour des instructions supplémentaires
- Mode debug pour un logging détaillé
- Mode silencieux pour minimiser les sorties
- Affichage des informations de l'assistant

## Installation

### Prérequis

Assurez-vous d'avoir Go 1.16 ou une version ultérieure installée sur votre système.

### Étapes d'installation

1. Clonez ce dépôt :
   ```
   git clone https://github.com/chrlesur/aiyou.cli.git
   ```
2. Naviguez vers le répertoire du projet :
   ```
   cd aiyou.cli
   ```
3. Construisez le projet :

   - Pour Windows :
   ```
   go build -o aiyou.cli.exe ./cmd/aiyou.cli
   ```

   - Pour Linux et macOS :
   ```
   go build -o aiyou.cli ./cmd/aiyou.cli
   ```

### Configuration spécifique à chaque système d'exploitation

#### Windows

Assurez-vous que votre `%GOPATH%\bin` est dans votre PATH. Vous pouvez l'ajouter en exécutant cette commande dans PowerShell :
```
$env:Path += ";$env:GOPATH\bin"
```

#### Linux et macOS

Ajoutez la ligne suivante à votre fichier `.bashrc`, `.zshrc` ou équivalent :
```
export PATH=$PATH:$(go env GOPATH)/bin
```
Puis rechargez votre configuration de shell :
```
source ~/.bashrc  # ou ~/.zshrc, selon votre shell
```

## Configuration

Avant d'utiliser aiyou.cli, assurez-vous de configurer vos identifiants AI.YOU dans un fichier `.env` à la racine du projet :

```
AIYOU_EMAIL=votre_email@exemple.com
AIYOU_PASSWORD=votre_mot_de_passe
```

## Utilisation

Pour démarrer une session de chat interactive :

```
./aiyou.cli interactive -a votre_id_assistant
```

Pour envoyer un message unique :

```
./aiyou.cli chat "Votre message" -a votre_id_assistant
```

Vous pouvez utiliser l'assistant LIA Mini par exemple avec l'id "asst_xLMDUf2cWAKaU8UBFFp1LsLA"

Pour plus d'options, utilisez la commande d'aide :

```
./aiyou.cli --help
```

## Documentation

Pour une documentation plus détaillée sur chaque module, veuillez consulter les fichiers suivants :

- [API](docs/api.md)
- [CLI](docs/cli.md)
- [Logger](docs/logger.md)
- [Main](docs/main.md)

## Licence

Distribué sous la licence GPL-3.0. Voir [LICENSE](LICENSE) pour plus d'informations.

## Contact

Christophe Lesur - christophe.lesur@cloud-temple.com

Lien du projet : [https://github.com/chrlesur/aiyou.cli](https://github.com/chrlesur/aiyou.cli)
