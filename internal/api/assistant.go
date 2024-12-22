package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/internal/interfaces"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

var (
	ErrAssistantNotFound = errors.New("assistant not found")
	ErrInvalidAssistantID = errors.New("invalid assistant ID")
)

// AssistantManager handles operations related to AI assistants.
type AssistantManager struct {
	client interfaces.AIClient
	cache cache.Cache
	config *config.Config
	logger *logger.Logger
	mu sync.RWMutex

	selectedAssistant string
}

// AssistantManagerConfig holds configuration for creating a new AssistantManager.
type AssistantManagerConfig struct {
	Client interfaces.AIClient
	Cache cache.Cache
	Config *config.Config
}

// NewAssistantManager creates a new instance of AssistantManager.
func NewAssistantManager(cfg AssistantManagerConfig) (*AssistantManager, error) {
	log := logger.GetLogger()

	log.Debug("Initializing AssistantManager")

	if cfg.Client == nil {
		log.Error("Failed to create AssistantManager: client is required")
		return nil, errors.New("client is required")
	}
	if cfg.Cache == nil {
		log.Error("Failed to create AssistantManager: cache is required")
		return nil, errors.New("cache is required")
	}
	if cfg.Config == nil {
		log.Error("Failed to create AssistantManager: config is required")
		return nil, errors.New("config is required")
	}

	am := &AssistantManager{
		client: cfg.Client,
		cache: cfg.Cache,
		config: cfg.Config,
		logger: log,
	}

	log.Info("AssistantManager initialized successfully")
	return am, nil
}

// ListAssistants retrieves the list of available assistants.
func (am *AssistantManager) ListAssistants(ctx context.Context) ([]aiyou.Assistant, error) {
	am.logger.Debug("Attempting to retrieve assistants list")

	if !am.client.IsAuthenticated() {
		am.logger.Error("Authentication required for listing assistants")
		return nil, ErrNotAuthenticated
	}

	cacheKey := "assistants:list"
	if entry, exists := am.cache.Get(ctx, cacheKey); exists {
		if assistants, ok := entry.Value.([]aiyou.Assistant); ok {
			am.logger.Debug("Retrieved assistants list from cache, count: %d", len(assistants))
			return assistants, nil
		}
		am.logger.Warning("Cache entry exists but type assertion failed for assistants list")
	}

	am.logger.Debug("Fetching assistants list from API")
	response, err := am.client.GetUserAssistants(ctx)
	if err != nil {
		am.logger.Error("Failed to get assistants from API: %v", err)
		return nil, fmt.Errorf("failed to get assistants: %w", err)
	}

	err = am.cache.Set(ctx, cacheKey, response.Members, cache.AssistantCache)
	if err != nil {
		am.logger.Warning("Failed to cache assistants list: %v", err)
	} else {
		am.logger.Debug("Successfully cached %d assistants", len(response.Members))
	}

	am.logger.Info("Successfully retrieved %d assistants", len(response.Members))
	return response.Members, nil
}

// GetAssistant retrieves information about a specific assistant.
func (am *AssistantManager) GetAssistant(ctx context.Context, assistantID string) (*aiyou.Assistant, error) {
	am.logger.Debug("Attempting to retrieve assistant with ID: %s", assistantID)

	if assistantID == "" {
		am.logger.Error("Invalid assistant ID provided: empty string")
		return nil, ErrInvalidAssistantID
	}

	assistants, err := am.ListAssistants(ctx)
	if err != nil {
		am.logger.Error("Failed to retrieve assistants list while looking for ID %s: %v", assistantID, err)
		return nil, err
	}

	for _, assistant := range assistants {
		if assistant.ID == assistantID {
			am.logger.Debug("Found assistant: %s", assistant.ID)
			return &assistant, nil
		}
	}

	am.logger.Warning("Assistant not found with ID: %s", assistantID)
	return nil, ErrAssistantNotFound
}

// SelectAssistant sets the specified assistant as the currently selected one.
func (am *AssistantManager) SelectAssistant(ctx context.Context, assistantID string) error {
	am.logger.Debug("Attempting to select assistant: %s", assistantID)

	assistant, err := am.GetAssistant(ctx, assistantID)
	if err != nil {
		am.logger.Error("Failed to select assistant %s: %v", assistantID, err)
		return err
	}

	am.mu.Lock()
	am.selectedAssistant = assistant.ID
	am.mu.Unlock()

	err = am.cache.Set(ctx, "selected_assistant", assistant.ID, cache.UserCache)
	if err != nil {
		am.logger.Warning("Failed to cache selected assistant %s: %v", assistant.ID, err)
	} else {
		am.logger.Debug("Successfully cached selected assistant: %s", assistant.ID)
	}

	am.logger.Info("Successfully selected assistant: %s", assistant.ID)
	return nil
}

// GetSelectedAssistant returns the currently selected assistant.
func (am *AssistantManager) GetSelectedAssistant(ctx context.Context) (*aiyou.Assistant, error) {
	am.logger.Debug("Retrieving currently selected assistant")

	am.mu.RLock()
	assistantID := am.selectedAssistant
	am.mu.RUnlock()

	if assistantID == "" {
		am.logger.Debug("No assistant selected in memory, checking cache")
		if entry, exists := am.cache.Get(ctx, "selected_assistant"); exists {
			if id, ok := entry.Value.(string); ok {
				assistantID = id
				am.logger.Debug("Found selected assistant in cache: %s", id)
			} else {
				am.logger.Warning("Cache entry exists but type assertion failed for selected assistant")
			}
		}
	}

	if assistantID == "" {
		am.logger.Warning("No assistant currently selected")
		return nil, errors.New("no assistant selected")
	}

	assistant, err := am.GetAssistant(ctx, assistantID)
	if err != nil {
		am.logger.Error("Failed to retrieve selected assistant %s: %v", assistantID, err)
		return nil, err
	}

	am.logger.Debug("Successfully retrieved selected assistant: %s", assistant.ID)
	return assistant, nil
}