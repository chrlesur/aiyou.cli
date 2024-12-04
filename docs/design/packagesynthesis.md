
# aiyou.golib Package Overview

## Core Components

### Client

The `Client` struct is the main entry point for interacting with the AI.YOU API.

```go
type Client struct {
    baseURL      string
    httpClient   *http.Client
    auth         Authenticator
    maxRetries   int
    initialDelay time.Duration
    logger       Logger
    safeLog      func(level LogLevel, format string, args ...interface{})
    rateLimiter  *RateLimiter
}

func NewClient(email, password string, options ...ClientOption) (*Client, error)
```

#### Fields Explanation:

- `baseURL`: The base URL for the AI.YOU API.
- `httpClient`: An HTTP client for making requests.
- `auth`: An Authenticator interface implementation for handling authentication.
- `maxRetries`: Maximum number of retries for failed requests.
- `initialDelay`: Initial delay for retry backoff.
- `logger`: A Logger interface for logging operations.
- `safeLog`: A function for safe logging that masks sensitive information.
- `rateLimiter`: A RateLimiter for controlling request rates.

#### Client Options:

Client behavior can be customized using functional options:

```go
func WithLogger(logger Logger) ClientOption
func WithBaseURL(url string) ClientOption
func WithRateLimiter(config RateLimiterConfig) ClientOption
func WithRetry(maxRetries int, initialDelay time.Duration) ClientOption
```

#### Key Methods

```go
func (c *Client) CreateChatCompletion(ctx context.Context, messages []Message, assistantID string) (*ChatCompletionResponse, error)
func (c *Client) CreateChatCompletionStream(ctx context.Context, messages []Message, assistantID string) (*StreamReader, error)
func (c *Client) GetUserAssistants(ctx context.Context) (*AssistantsResponse, error)
func (c *Client) TranscribeAudioFile(ctx context.Context, filePath string, opts *AudioTranscriptionRequest) (*AudioTranscriptionResponse, error)
func (c *Client) SaveConversation(ctx context.Context, req SaveConversationRequest) (*SaveConversationResponse, error)
func (c *Client) GetConversation(ctx context.Context, threadID string) (*ConversationThread, error)
func (c *Client) GetUserThreads(ctx context.Context, params *UserThreadsParams) (*UserThreadsOutput, error)
func (c *Client) DeleteThread(ctx context.Context, threadID string) error
func (c *Client) CreateModel(ctx context.Context, req ModelRequest) (*ModelResponse, error)
func (c *Client) GetModels(ctx context.Context) (*ModelsResponse, error)
```

### Authentication

```go
type JWTAuthenticator struct {
    email    string
    password string
    token    string
    expiry   time.Time
    client   *http.Client
    baseURL  string
    logger   Logger
}

func NewJWTAuthenticator(email, password, baseURL string, client *http.Client, logger Logger) *JWTAuthenticator
```

### Message Handling

```go
type Message struct {
    Role    string        `json:"role"`
    Content []ContentPart `json:"content"`
}

type ContentPart struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

type MessageBuilder struct {
    message Message
    logger  Logger
}

func NewMessageBuilder(role string, logger Logger) *MessageBuilder
func (mb *MessageBuilder) AddText(text string) *MessageBuilder
func (mb *MessageBuilder) AddImage(imageURL string) *MessageBuilder
func (mb *MessageBuilder) Build() Message
```

### Streaming

```go
type StreamReader struct {
    reader *bufio.Reader
    closer io.Closer
    logger Logger
}

func (sr *StreamReader) ReadChunk() (*ChatCompletionResponse, error)
func (sr *StreamReader) Close() error
```

### Rate Limiting

```go
type RateLimiter struct {
    tokens     float64
    capacity   float64
    refillRate float64
    lastRefill time.Time
    mutex      sync.Mutex
    logger     Logger
}

type RateLimiterConfig struct {
    RequestsPerSecond float64
    BurstSize         int
    WaitTimeout       time.Duration
}

func NewRateLimiter(config RateLimiterConfig, logger Logger) *RateLimiter
```

## Key Structures

### Requests

```go
type ChatCompletionRequest struct {
    Messages     []Message `json:"messages"`
    AssistantID  string    `json:"assistantId"`
    Temperature  int       `json:"temperature"`
    TopP         float64   `json:"top_p"`
    Stream       bool      `json:"stream"`
    PromptSystem string    `json:"promptSystem,omitempty"`
    Form         string    `json:"form,omitempty"`
    Stop         []string  `json:"stop,omitempty"`
    ThreadId     string    `json:"threadId,omitempty"`
    MaxTokens    *int      `json:"max_tokens,omitempty"`
}

type AudioTranscriptionRequest struct {
    FileName string `json:"fileName"`
    Language string `json:"language,omitempty"`
    Format   string `json:"format,omitempty"`
}

type SaveConversationRequest struct {
    AssistantID    string `json:"assistantId"`
    Conversation   string `json:"conversation"`
    ThreadID       string `json:"threadId,omitempty"`
    FirstMessage   string `json:"firstMessage"`
    ContentJson    string `json:"contentJson"`
    ModelName      string `json:"modelName"`
    IsNewAppThread bool   `json:"isNewAppThread"`
}
```

### Responses

```go
type ChatCompletionResponse struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   *Usage   `json:"usage,omitempty"`
}

type AudioTranscriptionResponse struct {
    Transcription string `json:"transcription"`
}

type AssistantsResponse struct {
    Context    string      `json:"@context"`
    ID         string      `json:"@id"`
    Type       string      `json:"@type"`
    TotalItems int         `json:"hydra:totalItems"`
    Members    []Assistant `json:"hydra:member"`
}
```

## Error Handling

The package uses custom error types:

```go
type APIError struct {
    StatusCode int
    Message    string
}

type AuthenticationError struct {
    Message string
}

type RateLimitError struct {
    RetryAfter   int
    IsClientSide bool
}

type NetworkError struct {
    Err error
}
```

## Logging

```go
type Logger interface {
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
    SetLevel(level LogLevel)
}

func NewDefaultLogger(w io.Writer) *defaultLogger
```

## Important Go Concepts: context.Context

`context.Context` is a key Go interface used extensively in aiyou.golib for managing API requests. It serves several critical purposes:

1. **Request Cancellation**: Allows for graceful cancellation of long-running operations.
2. **Deadline Management**: Enables setting timeouts for API calls.
3. **Value Propagation**: Can carry request-scoped values across API boundaries and between goroutines.

### Usage in aiyou.golib

Most methods in the `Client` struct accept a `context.Context` as their first parameter:

```go
func (c *Client) CreateChatCompletion(ctx context.Context, messages []Message, assistantID string) (*ChatCompletionResponse, error)
```

### Best Practices

1. **Always pass a context**: Even if you don't need cancellation, use `context.Background()`.
2. **Set timeouts**: Use `context.WithTimeout()` to prevent long-running requests:

   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()
   response, err := client.CreateChatCompletion(ctx, messages, assistantID)
   ```

3. **Propagate contexts**: Pass the context down the call stack to ensure proper cancellation.
4. **Check for cancellation**: In long-running operations, periodically check `ctx.Done()`.

## Important Constants and Variables

```go
const (
    DEBUG LogLevel = iota
    INFO
    WARN
    ERROR
)

var SupportedFormats = []SupportedAudioFormat{
    {Extension: ".mp3", MimeTypes: []string{"audio/mpeg"}, MaxSize: 25 * 1024 * 1024},
    {Extension: ".wav", MimeTypes: []string{"audio/wav", "audio/x-wav"}, MaxSize: 25 * 1024 * 1024},
    {Extension: ".m4a", MimeTypes: []string{"audio/mp4", "audio/x-m4a"}, MaxSize: 25 * 1024 * 1024},
}
```

## AI Model Structures

```go
type Model struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Version     string          `json:"version"`
    CreatedAt   time.Time       `json:"createdAt"`
    UpdatedAt   time.Time       `json:"updatedAt"`
    Properties  ModelProperties `json:"properties"`
}

type ModelProperties struct {
    MaxTokens    int      `json:"maxTokens"`
    Temperature  float64  `json:"temperature"`
    Provider     string   `json:"provider"`
    Capabilities []string `json:"capabilities"`
}

type ModelRequest struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    Properties  ModelProperties `json:"properties"`
}
```

## Conversation Thread Management

```go
type ConversationThread struct {
    ID                   string    `json:"id"`
    ThreadIdParam        int       `json:"threadIdParam"`
    Content              string    `json:"content"`
    AssistantContentJson string    `json:"assistantContentJson"`
    AssistantName        string    `json:"assistantName"`
    AssistantModel       *string   `json:"assistantModel"`
    AssistantId          int       `json:"assistantId"`
    AssistantIdOpenAi    string    `json:"assistantIdOpenAi"`
    FirstMessage         string    `json:"firstMessage"`
    CreatedAt            time.Time `json:"createdAt"`
    UpdatedAt            time.Time `json:"updatedAt"`
    IsNewAppThread       bool      `json:"isNewAppThread"`
}

type UserThreadsParams struct {
    Page         int    `json:"page,omitempty"`
    ItemsPerPage int    `json:"itemsPerPage,omitempty"`
    Search       string `json:"search,omitempty"`
}

// UserThreadsOutput represents the response containing user threads.
type UserThreadsOutput struct {
	Threads      []ConversationThread `json:"threads"`
	TotalItems   int                  `json:"totalItems"`
	ItemsPerPage int                  `json:"itemsPerPage"`
	CurrentPage  int                  `json:"currentPage"`
}
```

## Utility Functions

```go
func NewTextMessage(role, text string) Message
func NewImageMessage(role, imageURL string) Message
func validateAudioFile(file *os.File, filePath string) error
func isRetryableError(err error) bool
func retryOperation(ctx context.Context, logger Logger, maxRetries int, initialDelay time.Duration, operation func() error) error
func MaskSensitiveInfo(input string) string
func SafeLog(logger Logger) func(level LogLevel, format string, args ...interface{})
```

## Usage Examples

### Creating a Chat Completion

```go
messages := []aiyou.Message{
    aiyou.NewTextMessage("user", "Hello, AI!"),
}
response, err := client.CreateChatCompletion(ctx, messages, "assistant-id")
if err != nil {
    // Handle error
}
fmt.Println(response.Choices[0].Message.Content)
```

### Streaming Chat Completion

```go
stream, err := client.CreateChatCompletionStream(ctx, messages, "assistant-id")
if err != nil {
    // Handle error
}
defer stream.Close()

for {
    chunk, err := stream.ReadChunk()
    if err == io.EOF {
        break
    }
    if err != nil {
        // Handle error
    }
    // Process chunk
}
```

### Audio Transcription

```go
request := &aiyou.AudioTranscriptionRequest{
    Language: "en",
    Format:   "text",
}
response, err := client.TranscribeAudioFile(ctx, "path/to/audio.mp3", request)
if err != nil {
    // Handle error
}
fmt.Println(response.Transcription)
```

## Best Practices

1. Always use context for timeouts and cancellation.
2. Close streams after use.
3. Handle rate limiting errors by implementing appropriate backoff strategies.
4. Use the logger option for better debugging.
5. Securely store and manage authentication credentials.
6. Use the MessageBuilder for complex message construction.
7. Implement proper error handling, especially for custom error types.
8. Utilize the utility functions for common tasks like creating messages or masking sensitive information.
