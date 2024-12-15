
# aiyou.cli

aiyou.cli est une interface en ligne de commande pour interagir avec les assistants AI.YOU directement depuis votre terminal, compatible avec Windows, Linux et macOS.

## Caractéristiques

Chat avec les assistants AI.YOU :
- Mode interactif pour des conversations continues
- Mode message unique pour des requêtes rapides
- Streaming des réponses en temps réel avec affichage progressif
- Paramètres avancés pour contrôler la génération :
  - Temperature (0.0-1.0) pour ajuster la créativité
  - Top-p (0.0-1.0) pour contrôler la diversité
  - Maximum de tokens pour limiter la longueur des réponses
- Sélection flexible des assistants
- Cache intelligent des réponses fréquentes
- Gestion des sessions et de l'authentification
- Mode debug pour un logging détaillé
- Mode silencieux pour minimiser les sorties
- Configuration flexible via fichier ou variables d'environnement

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

### Chat avec un Assistant

1. Message unique :
   ```bash
   # Utilise l'assistant par défaut
   aiyou chat "Votre message"
   
   # Assistant spécifique
   aiyou chat -a asst_xLMDUf2cWAKaU8UBFFp1LsLA "Votre message"
   ```

2. Mode interactif :
   ```bash
   # Démarre une session interactive
   aiyou chat -i
   
   # Avec un assistant spécifique
   aiyou chat -i -a asst_xLMDUf2cWAKaU8UBFFp1LsLA
   ```

3. Streaming des réponses :
   ```bash
   # Affiche la réponse en temps réel
   aiyou chat -s "Votre message"
   
   # Combine streaming et mode interactif
   aiyou chat -i -s
   ```

4. Paramètres avancés :
   ```bash
   # Ajuste la créativité (0.0-1.0)
   aiyou chat --temperature 0.7 "Votre message"
   
   # Contrôle la diversité des réponses (0.0-1.0)
   aiyou chat --top-p 0.9 "Votre message"
   
   # Limite la longueur de la réponse
   aiyou chat --max-tokens 100 "Votre message"
   
   # Combine plusieurs paramètres
   aiyou chat --temperature 0.8 --top-p 0.9 --max-tokens 150 "Votre message"
   ```

## Exemples d'Utilisation

1. Chat simple avec l'assistant par défaut :
   ```bash
   aiyou chat "Quelle est la capitale de la France ?"
   ```

2. Session interactive avec streaming :
   ```bash
   aiyou chat -i -s
   > Explique-moi la relativité
   [Assistant répond en temps réel...]
   > Peux-tu simplifier ?
   [Assistant répond...]
   > exit
   ```

3. Utilisation créative avec paramètres avancés :
   ```bash
   # Génération créative avec température élevée
   aiyou chat --temperature 0.9 "Écris un poème sur l'automne"
   
   # Réponse concise avec limite de tokens
   aiyou chat --max-tokens 50 "Résume la Révolution française"
   
   # Combinaison de paramètres pour une réponse équilibrée
   aiyou chat --temperature 0.7 --top-p 0.9 --max-tokens 200 "Génère une histoire courte"
   ```

4. Streaming avec paramètres avancés :
   ```bash
   # Réponse créative en temps réel
   aiyou chat -s --temperature 0.8 "Invente une histoire"
   ```

## Paramètres Avancés

### Temperature (--temperature, -t)
- Contrôle la créativité des réponses
- Plage : 0.0 à 1.0
- Valeur basse (0.1-0.3) : Réponses plus concentrées et déterministes
- Valeur haute (0.7-0.9) : Réponses plus créatives et variées
- Défaut : 0.7

### Top-p (--top-p)
- Contrôle la diversité des tokens sélectionnés
- Plage : 0.0 à 1.0
- Valeur basse : Réponses plus focalisées
- Valeur haute : Plus grande diversité de vocabulaire
- Défaut : 1.0

### Max Tokens (--max-tokens)
- Limite la longueur de la réponse
- 0 : Pas de limite
- Valeurs typiques : 50-2000
- Défaut : 0 (pas de limite)

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
