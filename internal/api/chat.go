// Package api provides core functionality for interacting with the AI.YOU API
package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/internal/interfaces"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/sirupsen/logrus"
)

// Error definitions
var (
	ErrNoActiveSession   = errors.New("no active chat session")
	ErrInvalidAssistant  = errors.New("invalid assistant ID")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrMessageTooLong    = errors.New("message exceeds maximum length")
)

// ChatParameters represents advanced parameters for chat completion
type ChatParameters struct {
	Temperature float32
	TopP        float32
	MaxTokens   int
}

// DefaultChatParameters returns the default parameters for chat completion
func DefaultChatParameters() ChatParameters {
	return ChatParameters{
		Temperature: 0.7,
		TopP:        1.0,
		MaxTokens:   0, // 0 means no limit
	}
}

// Validate checks if the parameters are within valid ranges
func (p ChatParameters) Validate() error {
	if p.Temperature < 0.0 || p.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0")
	}
	if p.TopP < 0.0 || p.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0.0 and 1.0")
	}
	if p.MaxTokens < 0 {
		return fmt.Errorf("max_tokens cannot be negative")
	}
	return nil
}

// ChatSession represents an active chat conversation
type ChatSession struct {
	AssistantID  string
	ThreadID     string
	StartTime    time.Time
	LastActivity time.Time
	Messages     []aiyou.Message
}

// RateLimiter handles request rate limiting
type RateLimiter struct {
	requestsPerMinute int
	burstSize         int
	tokens            int
	lastRefill        time.Time
	mu                sync.Mutex
}

// ChatManager handles chat interactions with AI assistants
type ChatManager struct {
	client interfaces.AIClient
	cache  cache.Cache
	config *config.Config
	logger *logrus.Logger

	activeSession *ChatSession
	mu            sync.RWMutex
	rateLimiter   *RateLimiter
}

// ChatManagerConfig holds configuration for creating a new ChatManager
type ChatManagerConfig struct {
	Client interfaces.AIClient
	Cache  cache.Cache
	Config *config.Config
	Logger *logrus.Logger
}

// NewChatManager creates a new instance of ChatManager
func NewChatManager(cfg ChatManagerConfig) (*ChatManager, error) {
	if cfg.Client == nil {
		return nil, errors.New("client is required")
	}
	if cfg.Cache == nil {
		return nil, errors.New("cache is required")
	}
	if cfg.Config == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}

	cm := &ChatManager{
		client: cfg.Client,
		cache:  cfg.Cache,
		config: cfg.Config,
		logger: cfg.Logger,
		rateLimiter: &RateLimiter{
			requestsPerMinute: 60,
			burstSize:         5,
			tokens:            5,
			lastRefill:        time.Now(),
		},
	}

	cm.logger.Debug("ChatManager initialized successfully")
	return cm, nil
}

// SendMessageWithParams sends a message with advanced parameters
func (cm *ChatManager) SendMessageWithParams(ctx context.Context, message string, assistantID string, params ChatParameters) (*aiyou.ChatCompletionResponse, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if err := cm.checkAuthentication(); err != nil {
		return nil, err
	}

	if err := cm.validateMessage(message); err != nil {
		return nil, err
	}

	messages := []aiyou.Message{
		{
			Role: "user",
			Content: []aiyou.ContentPart{
				{
					Type: "text",
					Text: message,
				},
			},
		},
	}

	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
		cm.logger.Debug("Added conversation context")
	}
	cm.mu.RUnlock()

	if err := cm.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Convert TopP from float32 to float64 for API compatibility
	response, err := cm.client.CreateChatCompletion(timeoutCtx, messages, assistantID)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	cm.mu.Lock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		cm.activeSession.Messages = append(
			cm.activeSession.Messages,
			messages[len(messages)-1],
			response.Choices[0].Message,
		)
		cm.activeSession.LastActivity = time.Now()
	}
	cm.mu.Unlock()

	return response, nil
}

// SendMessageStream sends a message and returns a stream of responses
func (cm *ChatManager) SendMessageStream(ctx context.Context, message string, assistantID string) (*aiyou.StreamReader, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	if err := cm.checkAuthentication(); err != nil {
		return nil, err
	}

	if err := cm.validateMessage(message); err != nil {
		return nil, err
	}

	messages := []aiyou.Message{
		{
			Role: "user",
			Content: []aiyou.ContentPart{
				{
					Type: "text",
					Text: message,
				},
			},
		},
	}

	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
		cm.logger.Debug("Added conversation context to stream request")
	}
	cm.mu.RUnlock()

	if err := cm.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	stream, err := cm.client.CreateChatCompletionStream(timeoutCtx, messages, assistantID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start message stream: %w", err)
	}

	cm.mu.Lock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		cm.activeSession.Messages = append(cm.activeSession.Messages, messages[len(messages)-1])
		cm.activeSession.LastActivity = time.Now()
	}
	cm.mu.Unlock()

	cm.logger.Debug("Stream started successfully")
	return stream, nil
}

// StartConversation initiates a new chat session
func (cm *ChatManager) StartConversation(ctx context.Context, assistantID string) error {
	if !cm.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	cm.logger.WithField("assistant_id", assistantID).Debug("Starting new conversation")

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession != nil {
		return errors.New("another conversation is already active")
	}

	cm.activeSession = &ChatSession{
		AssistantID:  assistantID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Messages:     make([]aiyou.Message, 0),
	}

	return nil
}

// EndConversation ends the current chat session
func (cm *ChatManager) EndConversation(ctx context.Context) error {
	// Check authentication first
	if !cm.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession == nil {
		return ErrNoActiveSession
	}

	cm.activeSession = nil
	return nil
}

// validateMessage checks if a message meets the required criteria
func (cm *ChatManager) validateMessage(message string) error {
	if len(message) == 0 {
		return errors.New("message cannot be empty")
	}
	if len(message) > 4000 {
		return fmt.Errorf("%w: message length is %d characters", ErrMessageTooLong, len(message))
	}
	return nil
}

// checkAuthentication verifies if the client is authenticated
func (cm *ChatManager) checkAuthentication() error {
	if !cm.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}
	return nil
}

// Wait implements rate limiting
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed.Minutes()) * rl.requestsPerMinute

	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.burstSize)
		rl.lastRefill = now
	}

	if rl.tokens <= 0 {
		return ErrRateLimitExceeded
	}

	rl.tokens--
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetTestClient is used for testing purposes only
func (cm *ChatManager) SetTestClient(client interfaces.AIClient) {
	cm.client = client
}

// GetAssistants retrieves the list of available assistants
func (cm *ChatManager) GetAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if err := cm.checkAuthentication(); err != nil {
		return nil, err
	}

	// Check cache
	cacheKey := "assistants:list"
	if entry, exists := cm.cache.Get(ctx, cacheKey); exists {
		if response, ok := entry.Value.(*aiyou.AssistantsResponse); ok {
			return response, nil
		}
	}

	// Call API
	response, err := cm.client.GetUserAssistants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistants: %w", err)
	}

	// Cache the result
	cm.cache.Set(ctx, cacheKey, response, cache.AssistantCache)

	return response, nil
}
