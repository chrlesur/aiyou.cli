// Package api provides core functionality for interacting with the AI.YOU API.
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
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

// Error definitions for common chat-related errors
var (
	ErrNoActiveSession   = errors.New("no active chat session")
	ErrInvalidAssistant  = errors.New("invalid assistant ID")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrMessageTooLong    = errors.New("message exceeds maximum length")
)

// ChatParameters represents advanced parameters for chat completion
type ChatParameters struct {
	Temperature float32 // Controls randomness (0.0 to 1.0)
	TopP        float32 // Controls diversity (0.0 to 1.0)
	MaxTokens   int     // Maximum length of the generated response
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
	AssistantID  string          // ID of the AI assistant
	ThreadID     string          // Optional thread ID for conversation tracking
	StartTime    time.Time       // When the session started
	LastActivity time.Time       // Last message timestamp
	Messages     []aiyou.Message // Conversation history
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
	logger *logger.Logger

	activeSession *ChatSession
	mu            sync.RWMutex
	rateLimiter   *RateLimiter
}

// ChatManagerConfig holds configuration for creating a new ChatManager
type ChatManagerConfig struct {
	Client interfaces.AIClient
	Cache  cache.Cache
	Config *config.Config
}

// NewChatManager creates a new instance of ChatManager
func NewChatManager(cfg ChatManagerConfig) (*ChatManager, error) {
	log := logger.GetLogger()
	log.Debug("Initializing ChatManager")

	if cfg.Client == nil {
		log.Error("Failed to create ChatManager: client is required")
		return nil, errors.New("client is required")
	}
	if cfg.Cache == nil {
		log.Error("Failed to create ChatManager: cache is required")
		return nil, errors.New("cache is required")
	}
	if cfg.Config == nil {
		log.Error("Failed to create ChatManager: config is required")
		return nil, errors.New("config is required")
	}

	cm := &ChatManager{
		client: cfg.Client,
		cache:  cfg.Cache,
		config: cfg.Config,
		logger: log,
		rateLimiter: &RateLimiter{
			requestsPerMinute: 60,
			burstSize:         5,
			tokens:            5,
			lastRefill:        time.Now(),
		},
	}

	log.Info("ChatManager initialized successfully with rate limit: %d requests/minute", cm.rateLimiter.requestsPerMinute)
	return cm, nil
}

// SendMessageWithParams sends a message with advanced parameters to an AI assistant
func (cm *ChatManager) SendMessageWithParams(ctx context.Context, message string, assistantID string, params ChatParameters) (*aiyou.ChatCompletionResponse, error) {
	cm.logger.Debug("Sending message to assistant %s with parameters: temperature=%.2f, topP=%.2f, maxTokens=%d",
		assistantID, params.Temperature, params.TopP, params.MaxTokens)

	if err := params.Validate(); err != nil {
		cm.logger.Error("Invalid chat parameters: %v", err)
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if err := cm.checkAuthentication(); err != nil {
		cm.logger.Error("Authentication check failed before sending message")
		return nil, err
	}

	if err := cm.validateMessage(message); err != nil {
		cm.logger.Error("Message validation failed: %v", err)
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

	// Add conversation context if available
	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
		cm.logger.Debug("Added conversation context: %d previous messages", len(cm.activeSession.Messages))
	}
	cm.mu.RUnlock()

	// Check rate limiting
	if err := cm.rateLimiter.Wait(ctx); err != nil {
		cm.logger.Warning("Rate limit exceeded, request delayed")
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	// Set request timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cm.logger.Debug("Sending request to API with %d total messages", len(messages))
	response, err := cm.client.CreateChatCompletion(timeoutCtx, messages, assistantID)
	if err != nil {
		cm.logger.Error("API request failed: %v", err)
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Update session with new messages
	cm.mu.Lock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		cm.activeSession.Messages = append(
			cm.activeSession.Messages,
			messages[len(messages)-1],
			response.Choices[0].Message,
		)
		cm.activeSession.LastActivity = time.Now()
		cm.logger.Debug("Updated session with new messages, total messages: %d", len(cm.activeSession.Messages))
	}
	cm.mu.Unlock()

	cm.logger.Info("Successfully received response from assistant %s", assistantID)
	return response, nil
}

// SendMessageStream sends a message and returns a stream of responses
func (cm *ChatManager) SendMessageStream(ctx context.Context, message string, assistantID string) (*aiyou.StreamReader, error) {
	cm.logger.Debug("Initiating streaming message to assistant: %s", assistantID)

	if err := ctx.Err(); err != nil {
		cm.logger.Error("Context error before starting stream: %v", err)
		return nil, fmt.Errorf("context error: %w", err)
	}

	if err := cm.checkAuthentication(); err != nil {
		cm.logger.Error("Authentication check failed before streaming")
		return nil, err
	}

	if err := cm.validateMessage(message); err != nil {
		cm.logger.Error("Message validation failed for streaming: %v", err)
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

	// Add conversation context if available
	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
		cm.logger.Debug("Added conversation context to stream: %d previous messages", len(cm.activeSession.Messages))
	}
	cm.mu.RUnlock()

	if err := cm.rateLimiter.Wait(ctx); err != nil {
		cm.logger.Warning("Rate limit exceeded for stream request")
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	go func() {
		<-ctx.Done()
		cm.logger.Debug("Stream context cancelled")
		cancel()
	}()

	cm.logger.Debug("Establishing stream connection with API")
	stream, err := cm.client.CreateChatCompletionStream(timeoutCtx, messages, assistantID)
	if err != nil {
		cm.logger.Error("Failed to establish stream connection: %v", err)
		cancel()
		return nil, fmt.Errorf("failed to start message stream: %w", err)
	}

	// Update session with new user message
	cm.mu.Lock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		cm.activeSession.Messages = append(cm.activeSession.Messages, messages[len(messages)-1])
		cm.activeSession.LastActivity = time.Now()
		cm.logger.Debug("Updated session with new user message")
	}
	cm.mu.Unlock()

	cm.logger.Info("Successfully established streaming connection with assistant %s", assistantID)
	return stream, nil
}

// GetAssistants retrieves the list of available assistants
func (cm *ChatManager) GetAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	cm.logger.Debug("Retrieving list of available assistants")

	if err := cm.checkAuthentication(); err != nil {
		cm.logger.Error("Authentication check failed while retrieving assistants")
		return nil, err
	}

	// Check cache first
	cacheKey := "assistants:list"
	if entry, exists := cm.cache.Get(ctx, cacheKey); exists {
		if response, ok := entry.Value.(*aiyou.AssistantsResponse); ok {
			cm.logger.Debug(`Retrieved assistants list from cache`)
			return response, nil
		}
		cm.logger.Warning("Cache entry exists but type assertion failed")
	}

	// Fetch from API if not in cache
	cm.logger.Debug("Fetching assistants list from API")
	response, err := cm.client.GetUserAssistants(ctx)
	if err != nil {
		cm.logger.Error("Failed to fetch assistants from API: %v", err)
		return nil, fmt.Errorf("failed to get assistants: %w", err)
	}

	// Cache the response
	err = cm.cache.Set(ctx, cacheKey, response, cache.AssistantCache)
	if err != nil {
		cm.logger.Warning("Failed to cache assistants list: %v", err)
	} else {
		cm.logger.Debug("Successfully cached assistants list")
	}

	cm.logger.Info("Successfully retrieved %d assistants", len(response.Members))
	return response, nil
}

// StartConversation initiates a new chat session with an assistant
func (cm *ChatManager) StartConversation(ctx context.Context, assistantID string) error {
	cm.logger.Debug("Attempting to start new conversation with assistant: %s", assistantID)

	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Cannot start conversation: client not authenticated")
		return ErrNotAuthenticated
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession != nil {
		cm.logger.Warning("Cannot start new conversation: active session exists with assistant %s",
			cm.activeSession.AssistantID)
		return errors.New("another conversation is already active")
	}

	cm.activeSession = &ChatSession{
		AssistantID:  assistantID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Messages:     make([]aiyou.Message, 0, 10), // Pre-allocate space for 10 messages
	}

	cm.logger.Info("Successfully started new conversation with assistant %s", assistantID)
	return nil
}

// EndConversation ends the current chat session and cleans up resources
func (cm *ChatManager) EndConversation(ctx context.Context) error {
	cm.logger.Debug("Attempting to end current conversation")

	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Cannot end conversation: client not authenticated")
		return ErrNotAuthenticated
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession == nil {
		cm.logger.Warning("Cannot end conversation: no active session found")
		return ErrNoActiveSession
	}

	assistantID := cm.activeSession.AssistantID
	messageCount := len(cm.activeSession.Messages)
	duration := time.Since(cm.activeSession.StartTime)

	cm.activeSession = nil

	cm.logger.Info("Successfully ended conversation with assistant %s (duration: %v, messages: %d)",
		assistantID, duration.Round(time.Second), messageCount)
	return nil
}

// validateMessage checks if a message meets the required criteria
func (cm *ChatManager) validateMessage(message string) error {
	cm.logger.Debug("Validating message length: %d characters", len(message))

	if len(message) == 0 {
		cm.logger.Warning("Message validation failed: empty message")
		return errors.New("message cannot be empty")
	}

	if len(message) > 4000 {
		cm.logger.Warning("Message validation failed: message too long (%d characters)", len(message))
		return fmt.Errorf("%w: message length is %d characters", ErrMessageTooLong, len(message))
	}

	return nil
}

// checkAuthentication verifies if the client is authenticated
func (cm *ChatManager) checkAuthentication() error {
	cm.logger.Debug("Checking authentication status")

	if !cm.client.IsAuthenticated() {
		cm.logger.Warning("Authentication check failed: client not authenticated")
		return ErrNotAuthenticated
	}

	return nil
}

// Wait implements rate limiting for API requests
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

// GetConversationStats returns current conversation statistics
func (cm *ChatManager) GetConversationStats() *ConversationStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.activeSession == nil {
		return nil
	}

	stats := &ConversationStats{
		AssistantID:  cm.activeSession.AssistantID,
		MessageCount: len(cm.activeSession.Messages),
		Duration:     time.Since(cm.activeSession.StartTime),
		LastActivity: time.Since(cm.activeSession.LastActivity),
	}

	cm.logger.Debug("Conversation stats - Assistant: %s, Messages: %d, Duration: %v, Last Activity: %v",
		stats.AssistantID, stats.MessageCount,
		stats.Duration.Round(time.Second),
		stats.LastActivity.Round(time.Second))

	return stats
}

// ConversationStats holds statistics about the current conversation
type ConversationStats struct {
	AssistantID  string
	MessageCount int
	Duration     time.Duration
	LastActivity time.Duration
}

// SetTestClient is used for testing purposes only
func (cm *ChatManager) SetTestClient(client interfaces.AIClient) {
	cm.logger.Debug("Setting test client")
	cm.client = client
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Close performs cleanup when the ChatManager is no longer needed
func (cm *ChatManager) Close() error {
	cm.logger.Debug("Cleaning up ChatManager resources")

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession != nil {
		cm.logger.Warning("Closing ChatManager with active session, forcing session end")
		cm.activeSession = nil
	}

	cm.logger.Info("ChatManager successfully closed")
	return nil
}

// ValidateAssistantID checks if an assistant ID is valid
func (cm *ChatManager) ValidateAssistantID(ctx context.Context, assistantID string) error {
	cm.logger.Debug("Validating assistant ID: %s", assistantID)

	if assistantID == "" {
		cm.logger.Warning("Invalid assistant ID: empty string")
		return ErrInvalidAssistant
	}

	assistants, err := cm.GetAssistants(ctx)
	if err != nil {
		cm.logger.Error("Failed to validate assistant ID: %v", err)
		return fmt.Errorf("failed to validate assistant ID: %w", err)
	}

	for _, assistant := range assistants.Members {
		if assistant.ID == assistantID {
			cm.logger.Debug("Assistant ID validated successfully")
			return nil
		}
	}

	cm.logger.Warning("Invalid assistant ID: %s not found", assistantID)
	return ErrInvalidAssistant
}
