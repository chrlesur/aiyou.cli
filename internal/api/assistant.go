// Package api provides the core API interaction layer for the AI.YOU CLI.
package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/internal/interfaces"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/sirupsen/logrus"
)

var (
	ErrAssistantNotFound  = errors.New("assistant not found")
	ErrInvalidAssistantID = errors.New("invalid assistant ID")
)

// AssistantManager handles operations related to AI assistants.
type AssistantManager struct {
	client interfaces.AIClient
	cache  cache.Cache
	config *config.Config
	logger *logrus.Logger
	mu     sync.RWMutex

	// Cache des assistants sélectionnés par l'utilisateur
	selectedAssistant string
}

// AssistantManagerConfig holds configuration for creating a new AssistantManager.
type AssistantManagerConfig struct {
	Client interfaces.AIClient
	Cache  cache.Cache
	Config *config.Config
	Logger *logrus.Logger
}

// NewAssistantManager creates a new instance of AssistantManager.
func NewAssistantManager(cfg AssistantManagerConfig) (*AssistantManager, error) {
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

	return &AssistantManager{
		client: cfg.Client,
		cache:  cfg.Cache,
		config: cfg.Config,
		logger: cfg.Logger,
	}, nil
}

// ListAssistants retrieves the list of available assistants.
func (am *AssistantManager) ListAssistants(ctx context.Context) ([]aiyou.Assistant, error) {
	// Vérifier l'authentification
	if !am.client.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Vérifier le cache
	cacheKey := "assistants:list"
	if entry, exists := am.cache.Get(ctx, cacheKey); exists {
		if assistants, ok := entry.Value.([]aiyou.Assistant); ok {
			am.logger.Debug("Using cached assistants list")
			return assistants, nil
		}
	}

	// Appeler l'API
	response, err := am.client.GetUserAssistants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistants: %w", err)
	}

	// Mettre en cache
	err = am.cache.Set(ctx, cacheKey, response.Members, cache.AssistantCache)
	if err != nil {
		am.logger.WithError(err).Warn("Failed to cache assistants list")
	}

	return response.Members, nil
}

// GetAssistant retrieves information about a specific assistant.
func (am *AssistantManager) GetAssistant(ctx context.Context, assistantID string) (*aiyou.Assistant, error) {
	if assistantID == "" {
		return nil, ErrInvalidAssistantID
	}

	assistants, err := am.ListAssistants(ctx)
	if err != nil {
		return nil, err
	}

	for _, assistant := range assistants {
		if assistant.ID == assistantID {
			return &assistant, nil
		}
	}

	return nil, ErrAssistantNotFound
}

// SelectAssistant sets the specified assistant as the currently selected one.
func (am *AssistantManager) SelectAssistant(ctx context.Context, assistantID string) error {
	assistant, err := am.GetAssistant(ctx, assistantID)
	if err != nil {
		return err
	}

	am.mu.Lock()
	am.selectedAssistant = assistant.ID
	am.mu.Unlock()

	// Sauvegarder la préférence dans le cache
	err = am.cache.Set(ctx, "selected_assistant", assistant.ID, cache.UserCache)
	if err != nil {
		am.logger.WithError(err).Warn("Failed to cache selected assistant")
	}

	return nil
}

// GetSelectedAssistant returns the currently selected assistant.
func (am *AssistantManager) GetSelectedAssistant(ctx context.Context) (*aiyou.Assistant, error) {
	am.mu.RLock()
	assistantID := am.selectedAssistant
	am.mu.RUnlock()

	if assistantID == "" {
		// Essayer de récupérer depuis le cache
		if entry, exists := am.cache.Get(ctx, "selected_assistant"); exists {
			if id, ok := entry.Value.(string); ok {
				assistantID = id
			}
		}
	}

	if assistantID == "" {
		return nil, errors.New("no assistant selected")
	}

	return am.GetAssistant(ctx, assistantID)
}
