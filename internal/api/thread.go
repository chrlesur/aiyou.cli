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
	ErrThreadNotFound      = errors.New("thread not found")
	ErrInvalidThreadID     = errors.New("invalid thread ID")
	ErrThreadLimitExceeded = errors.New("thread limit exceeded")
)

// ThreadManager handles operations related to conversation threads
type ThreadManager struct {
	client interfaces.AIClient
	cache  cache.Cache
	config *config.Config
	logger *logrus.Logger
	mu     sync.RWMutex
}

// ThreadManagerConfig holds configuration for creating a new ThreadManager
type ThreadManagerConfig struct {
	Client interfaces.AIClient
	Cache  cache.Cache
	Config *config.Config
	Logger *logrus.Logger
}

// NewThreadManager creates a new instance of ThreadManager
func NewThreadManager(cfg ThreadManagerConfig) (*ThreadManager, error) {
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

	return &ThreadManager{
		client: cfg.Client,
		cache:  cfg.Cache,
		config: cfg.Config,
		logger: cfg.Logger,
	}, nil
}

// CreateThread creates a new conversation thread
func (tm *ThreadManager) CreateThread(ctx context.Context) (*aiyou.ConversationThread, error) {
    if !tm.client.IsAuthenticated() {
        return nil, ErrNotAuthenticated
    }

    // Vérifier la limite des threads
    threadsOutput, err := tm.client.GetUserThreads(ctx, &aiyou.UserThreadsParams{})
    if err != nil {
        return nil, fmt.Errorf("failed to check thread limit: %w", err)
    }

    // Vérifier si la limite de threads est atteinte
    if len(threadsOutput.Threads) >= tm.config.MaxThreads {
        return nil, ErrThreadLimitExceeded
    }

    // Le reste de la logique de création de thread...
    req := aiyou.SaveConversationRequest{
        IsNewAppThread: true,
    }
    resp, err := tm.client.SaveConversation(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create thread: %w", err)
    }

    // Récupérer le thread créé
    thread, err := tm.client.GetConversation(ctx, resp.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get created thread: %w", err)
    }

    // Mettre à jour le cache
    cacheKey := fmt.Sprintf("thread:%s", thread.ID)
    err = tm.cache.Set(ctx, cacheKey, thread, cache.QueryCache)
    if err != nil {
        tm.logger.WithError(err).Warn("Failed to cache thread")
    }

    return thread, nil
}

// GetThread retrieves a specific conversation thread
func (tm *ThreadManager) GetThread(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	if threadID == "" {
		return nil, ErrInvalidThreadID
	}

	if !tm.client.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Vérifier le cache
	cacheKey := fmt.Sprintf("thread:%s", threadID)
	if entry, exists := tm.cache.Get(ctx, cacheKey); exists {
		if thread, ok := entry.Value.(*aiyou.ConversationThread); ok {
			return thread, nil
		}
	}

	// Récupérer depuis l'API
	thread, err := tm.client.GetConversation(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	// Mettre en cache
	err = tm.cache.Set(ctx, cacheKey, thread, cache.QueryCache)
	if err != nil {
		tm.logger.WithError(err).Warn("Failed to cache thread")
	}

	return thread, nil
}

// ListThreads retrieves a list of conversation threads with pagination
func (tm *ThreadManager) ListThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	if !tm.client.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Vérifier le cache uniquement pour la première page sans recherche
	if params.Page == 1 && params.Search == "" {
		cacheKey := "threads:list"
		if entry, exists := tm.cache.Get(ctx, cacheKey); exists {
			if output, ok := entry.Value.(*aiyou.UserThreadsOutput); ok {
				return output, nil
			}
		}
	}

	// Récupérer depuis l'API
	output, err := tm.client.GetUserThreads(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	// Mettre en cache uniquement la première page sans recherche
	if params.Page == 1 && params.Search == "" {
		err = tm.cache.Set(ctx, "threads:list", output, cache.QueryCache)
		if err != nil {
			tm.logger.WithError(err).Warn("Failed to cache threads list")
		}
	}

	return output, nil
}

// DeleteThread deletes a conversation thread
func (tm *ThreadManager) DeleteThread(ctx context.Context, threadID string) error {
	if threadID == "" {
		return ErrInvalidThreadID
	}

	if !tm.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	// Supprimer via l'API
	err := tm.client.DeleteThread(ctx, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	// Supprimer du cache
	cacheKey := fmt.Sprintf("thread:%s", threadID)
	err = tm.cache.Delete(ctx, cacheKey)
	if err != nil {
		tm.logger.WithError(err).Warn("Failed to remove thread from cache")
	}

	// Invalider le cache de la liste des threads
	err = tm.cache.Delete(ctx, "threads:list")
	if err != nil {
		tm.logger.WithError(err).Warn("Failed to invalidate threads list cache")
	}

	return nil
}

// Pour les tests uniquement
func (tm *ThreadManager) SetTestClient(client interfaces.AIClient) {
	tm.client = client
}
