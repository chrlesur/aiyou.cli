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
	"github.com/chrlesur/aiyou.golib"
	"github.com/sirupsen/logrus"
)

// Définition des erreurs
var (
	ErrNoActiveSession   = errors.New("no active chat session")
	ErrInvalidAssistant  = errors.New("invalid assistant ID")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrMessageTooLong    = errors.New("message exceeds maximum length")
)

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

// RateLimiterConfig contains configuration for rate limiting
type RateLimiterConfig struct {
	RequestsPerMinute int
	BurstSize         int
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

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		requestsPerMinute: config.RequestsPerMinute,
		burstSize:         config.BurstSize,
		tokens:            config.BurstSize,
		lastRefill:        time.Now(),
	}
}

// SaveConversationRequest represents the data needed to save a conversation
type SaveConversationRequest struct {
	AssistantID    string
	ThreadID       string
	FirstMessage   string
	IsNewAppThread bool
}

// Wait blocks until a token is available or returns an error
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens <= 0 {
		return ErrRateLimitExceeded
	}

	rl.tokens--
	return nil
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	tokensToAdd := int(elapsed.Minutes()) * rl.requestsPerMinute
	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.burstSize)
		rl.lastRefill = now
	}
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
		rateLimiter: NewRateLimiter(RateLimiterConfig{
			RequestsPerMinute: 60,
			BurstSize:         5,
		}),
	}

	cm.logger.Debug("ChatManager initialized successfully")
	return cm, nil
}

// SendMessage sends a single message to an AI assistant and returns the response.
func (cm *ChatManager) SendMessage(ctx context.Context, message string, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	// Vérifier l'authentification
	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Authentication required")
		return nil, ErrNotAuthenticated
	}

	// Valider le message
	if err := cm.validateMessage(message); err != nil {
		cm.logger.WithError(err).Error("Message validation failed")
		return nil, err
	}

	// Vérifier le rate limiting
	if err := cm.rateLimiter.Wait(ctx); err != nil {
		cm.logger.WithError(err).Warn("Rate limit exceeded")
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	// Vérifier le contexte
	if err := ctx.Err(); err != nil {
		cm.logger.WithError(err).Error("Context cancelled")
		return nil, err
	}

	// Chercher dans le cache
	cacheKey := fmt.Sprintf("chat:%s:%s", assistantID, message)
	if entry, exists := cm.cache.Get(ctx, cacheKey); exists {
		if response, ok := entry.Value.(*aiyou.ChatCompletionResponse); ok {
			cm.logger.Debug("Returning cached response")
			return response, nil
		}
	}

	// Préparer le message
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

	// Ajouter le contexte de conversation si une session est active
	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
		cm.logger.WithField("message_count", len(messages)).Debug("Added conversation context")
	}
	cm.mu.RUnlock()

	// Créer un timeout context pour l'appel API
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Envoyer la requête à l'API
	cm.logger.Debug("Sending request to API")
	response, err := cm.client.CreateChatCompletion(timeoutCtx, messages, assistantID)
	if err != nil {
		cm.logger.WithError(err).Error("API request failed")
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Mettre en cache la réponse
	if err := cm.cache.Set(ctx, cacheKey, response, cache.QueryCache); err != nil {
		cm.logger.WithError(err).Warn("Failed to cache response")
	}

	// Mettre à jour la session si active
	cm.mu.Lock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		// Ajouter le message utilisateur et la réponse à l'historique
		cm.activeSession.Messages = append(cm.activeSession.Messages,
			messages[len(messages)-1],   // Message utilisateur
			response.Choices[0].Message, // Réponse de l'assistant
		)
		cm.activeSession.LastActivity = time.Now()
		cm.logger.Debug("Updated conversation history")
	}
	cm.mu.Unlock()

	return response, nil
}

// StartConversation initiates a new chat session
func (cm *ChatManager) StartConversation(ctx context.Context, assistantID string) error {
	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Authentication required")
		return ErrNotAuthenticated
	}

	cm.logger.WithField("assistant_id", assistantID).Debug("Starting new conversation")

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.activeSession != nil {
		cm.logger.Warn("Another conversation is already active")
		return errors.New("another conversation is already active, please end it first")
	}

	// Vérifier la validité de l'assistant
	cacheKey := fmt.Sprintf("assistant:%s", assistantID)
	exists := false
	if _, ok := cm.cache.Get(ctx, cacheKey); !ok {
		cm.logger.Debug("Validating assistant with API")

		// Ici, vous devriez implémenter la vérification réelle avec l'API
		// Pour l'exemple, nous supposons que l'assistant est valide
		valid := true // À remplacer par la vérification réelle

		if !valid {
			cm.logger.WithField("assistant_id", assistantID).Error("Invalid assistant ID")
			return ErrInvalidAssistant
		}

		// Mettre en cache le résultat de la validation
		cm.cache.Set(ctx, cacheKey, true, cache.AssistantCache)
	} else {
		exists = true
	}

	cm.logger.WithField("cached", exists).Debug("Assistant validation complete")

	// Créer une nouvelle session
	cm.activeSession = &ChatSession{
		AssistantID:  assistantID,
		ThreadID:     "", // Sera défini lors de la première interaction
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Messages:     make([]aiyou.Message, 0, 10),
	}

	cm.logger.WithFields(logrus.Fields{
		"assistant_id": assistantID,
		"start_time":   cm.activeSession.StartTime,
	}).Info("New conversation started successfully")

	return nil
}

// EndConversation terminates the current chat session and saves the conversation history.
func (cm *ChatManager) EndConversation(ctx context.Context) error {
	// Vérifier l'authentification
	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Authentication required")
		return ErrNotAuthenticated
	}

	cm.logger.Debug("Attempting to end conversation")

	// Vérifier le contexte
	if err := ctx.Err(); err != nil {
		cm.logger.WithError(err).Error("Context cancelled")
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Vérifier s'il y a une session active
	if cm.activeSession == nil {
		cm.logger.Warn("No active conversation to end")
		return ErrNoActiveSession
	}

	// Log des statistiques de session
	cm.logger.WithFields(logrus.Fields{
		"assistant_id":  cm.activeSession.AssistantID,
		"duration":      time.Since(cm.activeSession.StartTime),
		"message_count": len(cm.activeSession.Messages),
		"last_activity": cm.activeSession.LastActivity,
	}).Info("Ending conversation session")

	// Sauvegarder la conversation si elle contient des messages
	if len(cm.activeSession.Messages) > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		// Préparer la requête de sauvegarde
		req := &aiyou.SaveConversationRequest{
			AssistantID:    cm.activeSession.AssistantID,
			ThreadID:       cm.activeSession.ThreadID,
			FirstMessage:   cm.activeSession.Messages[0].Content[0].Text,
			IsNewAppThread: true,
		}

		cm.logger.Debug("Saving conversation history")
		err := cm.saveConversation(timeoutCtx, req)
		if err != nil {
			cm.logger.WithError(err).Error("Failed to save conversation")
			return fmt.Errorf("failed to save conversation: %w", err)
		}
	}

	// Nettoyer la session
	cm.activeSession = nil
	cm.logger.Info("Conversation ended successfully")

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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// saveConversation persists the conversation history to the server
func (cm *ChatManager) saveConversation(ctx context.Context, req *aiyou.SaveConversationRequest) error {
	cm.logger.Debug("Saving conversation to server")

	resp, err := cm.client.SaveConversation(ctx, *req)
	if err != nil {
		cm.logger.WithError(err).Error("Failed to save conversation")
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	if resp == nil {
		cm.logger.Error("Received nil response from server")
		return errors.New("received nil response from server")
	}

	cm.logger.WithFields(logrus.Fields{
		"assistant_id": req.AssistantID,
		"thread_id":    req.ThreadID,
	}).Debug("Conversation saved successfully")

	return nil
}

// SendMessageStream envoie un message et retourne un flux de réponses
func (cm *ChatManager) SendMessageStream(ctx context.Context, message string, assistantID string) (*aiyou.StreamReader, error) {
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

	// Ajouter le contexte de la conversation si disponible
	cm.mu.RLock()
	if cm.activeSession != nil && cm.activeSession.AssistantID == assistantID {
		messages = append(cm.activeSession.Messages, messages...)
	}
	cm.mu.RUnlock()

	return cm.client.CreateChatCompletionStream(ctx, messages, assistantID)
}

// GetAssistants récupère la liste des assistants disponibles
func (cm *ChatManager) GetAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if err := cm.checkAuthentication(); err != nil {
		return nil, err
	}

	// Vérifier le cache
	cacheKey := "assistants:list"
	if entry, exists := cm.cache.Get(ctx, cacheKey); exists {
		if response, ok := entry.Value.(*aiyou.AssistantsResponse); ok {
			return response, nil
		}
	}

	// Appeler l'API
	response, err := cm.client.GetUserAssistants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistants: %w", err)
	}

	// Mettre en cache
	cm.cache.Set(ctx, cacheKey, response, cache.AssistantCache)

	return response, nil
}

// checkAuthentication vérifie si le client est authentifié
func (cm *ChatManager) checkAuthentication() error {
	if !cm.client.IsAuthenticated() {
		cm.logger.Error("Authentication required")
		return ErrNotAuthenticated
	}
	return nil
}

// Pour les tests uniquement
func (cm *ChatManager) SetTestClient(client interfaces.AIClient) {
	cm.client = client
}
