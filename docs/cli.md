
# Module CLI

Le module CLI gère l'interface en ligne de commande de l'application aiyou.cli, en particulier le mode interactif.

## Fonction principale

### RunInteractiveMode

```go
func RunInteractiveMode(client *api.AIYOUClient)
```

Cette fonction lance le mode interactif, permettant à l'utilisateur de converser avec l'assistant AI.YOU en temps réel.

## Fonctionnalités

- Boucle de conversation interactive
- Gestion des commandes spéciales (comme '/quit' et '/save')
- Sauvegarde de l'historique des conversations

## Structure

### Message

```go
type Message struct {
    Role    string
    Content string
    Time    time.Time
}
```

Cette structure représente un message dans la conversation, stockant le rôle (utilisateur ou assistant), le contenu et l'horodatage.

## Utilisation

Le mode interactif est généralement lancé via la commande principale de l'application :

```
aiyou.cli interactive -a votre_id_assistant
```

Dans le mode interactif, l'utilisateur peut :
- Envoyer des messages à l'assistant
- Recevoir des réponses en temps réel
- Utiliser des commandes spéciales comme '/quit' pour quitter et '/save' pour sauvegarder la conversation

## Exemple de flux d'interaction

1. L'utilisateur lance le mode interactif
2. L'application affiche un message de bienvenue et des instructions
3. L'utilisateur entre un message
4. Le message est envoyé à l'assistant via le client API
5. La réponse de l'assistant est affichée
6. Le cycle se répète jusqu'à ce que l'utilisateur quitte

Ce module est crucial pour offrir une expérience utilisateur fluide et interactive avec l'assistant AI.YOU directement depuis le terminal.
```

Maintenant, passons à `docs/logger.md`:

```markdown
# Module Logger

Le module Logger fournit des fonctionnalités de journalisation pour l'application aiyou.cli.

## Variables globales

```go
var (
    debugMode   bool
    silentMode  bool
    logger      *log.Logger
)
```

Ces variables contrôlent le comportement global du logger.

## Fonctions principales

### SetDebugMode

```go
func SetDebugMode(debug bool)
```

Active ou désactive le mode debug.

### SetSilentMode

```go
func SetSilentMode(silent bool)
```

Active ou désactive le mode silencieux.

### Info, Debug, Error, Warning

```go
func Info(message string)
func Debug(message string)
func Error(message string)
func Warning(message string)
```

Ces fonctions permettent de logger des messages avec différents niveaux de gravité.

## Comportement

- En mode debug, tous les messages sont affichés.
- En mode silencieux, seuls les messages d'erreur sont affichés.
- Les messages sont formatés avec un horodatage et un niveau de gravité.

## Utilisation

```go
logger.SetDebugMode(true)
logger.Info("Application démarrée")
logger.Debug("Connexion à l'API")
logger.Error("Erreur de connexion")
```

Ce module permet une gestion flexible et efficace des logs de l'application, facilitant le débogage et le suivi des opérations.
```

Enfin, voici le contenu pour `docs/main.md`:

```markdown
# Module Main

Le module Main est le point d'entrée de l'application aiyou.cli. Il gère la configuration de l'application, le parsing des arguments de la ligne de commande, et l'exécution des commandes.

## Structures et variables principales

```go
var (
    debug            bool
    silent           bool
    aiyouAssistantID string
    instruction      string
    instructionFile  string
    showAssistantInfo bool
)
```

Ces variables globales sont utilisées pour stocker les options de la ligne de commande.

## Commandes

### rootCmd

La commande racine de l'application.

### versionCmd

Affiche la version de l'application.

### chatCmd

Envoie un message unique à l'assistant AI.YOU.

### interactiveCmd

Lance le mode de chat interactif.

## Fonctions principales

### init()

Initialise les drapeaux de la ligne de commande et configure les commandes.

### getAIYOUClient()

Crée et configure un client AI.YOU basé sur les paramètres fournis.

### main()

Point d'entrée de l'application, exécute la commande racine.

## Configuration

L'application utilise un fichier `.env` pour stocker les identifiants AI.YOU :

```
AIYOU_EMAIL=votre_email@exemple.com
AIYOU_PASSWORD=votre_mot_de_passe
```

## Utilisation

```
aiyou.cli [commande] [options]
```

Exemples :
- `aiyou.cli version`
- `aiyou.cli chat "Bonjour" -a votre_id_assistant`
- `aiyou.cli interactive -a votre_id_assistant --debug`

Ce module orchestre l'ensemble de l'application, gérant les interactions entre l'interface utilisateur, le client API, et le système de logging.
