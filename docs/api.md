
# Module API

Le module API gère les interactions avec l'API AI.YOU.

## Structures principales

### AIYOUClient

```go
type AIYOUClient struct {
    apiCaller   APICaller
    AssistantID string
    Debug       bool
    Timeout     time.Duration
}
```

Cette structure représente un client pour interagir avec l'API AI.YOU.

### AssistantInfo

```go
type AssistantInfo struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    Description  string `json:"description"`
    Image        string `json:"image"`
    AssistantID  string `json:"assistantId"`
    Model        string `json:"model"`
    ActiveScript bool   `json:"activeScript"`
    Tools        []Tool `json:"tools"`
    Voice        string `json:"voice"`
}
```

Cette structure contient les informations détaillées d'un assistant AI.YOU.

## Fonctions principales

### NewAIYOUClient

```go
func NewAIYOUClient(assistantID string, debug bool) *AIYOUClient
```

Crée une nouvelle instance de AIYOUClient.

### Login

```go
func (c *AIYOUClient) Login(email, password string) error
```

Authentifie le client auprès de l'API AI.YOU.

### Chat

```go
func (c *AIYOUClient) Chat(input, additionalInstruction string) (string, error)
```

Envoie un message à l'assistant et retourne sa réponse.

### GetAssistantInfo

```go
func (c *AIYOUClient) GetAssistantInfo() (*AssistantInfo, error)
```

Récupère les informations détaillées de l'assistant.

### LoadInstructionFromFile

```go
func (c *AIYOUClient) LoadInstructionFromFile(filename string) (string, error)
```

Charge des instructions supplémentaires à partir d'un fichier.

## Utilisation

Exemple d'utilisation du client AI.YOU :

```go
client := api.NewAIYOUClient("votre_assistant_id", true)
err := client.Login("email@example.com", "password")
if err != nil {
    log.Fatal(err)
}

response, err := client.Chat("Bonjour, comment ça va ?", "")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response)
```

Ce module est au cœur de l'interaction avec l'API AI.YOU, gérant l'authentification, les requêtes de chat et la récupération d'informations sur l'assistant.