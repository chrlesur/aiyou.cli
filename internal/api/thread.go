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
	ErrThreadNotFound      = errors.New("thread not found")
	ErrInvalidThreadID     = errors.New("invalid thread ID")
	ErrThreadLimitExceeded = errors.New("thread limit exceeded")
)

type ThreadManager struct {
	client interfaces.AIClient
	cache  cache.Cache
	config *config.Config
	logger *logger.Logger
	mu     sync.RWMutex
}

type ThreadManagerConfig struct {
	Client interfaces.AIClient
	Cache  cache.Cache
	Config *config.Config
}

func NewThreadManager(cfg ThreadManagerConfig) (*ThreadManager, error) {
	log := logger.GetLogger()
	log.Debug("Initializing ThreadManager")

	if cfg.Client == nil {
		log.Error("Failed to create ThreadManager: client is required")
		return nil, errors.New("client is required")
	}
	if cfg.Cache == nil {
		log.Error("Failed to create ThreadManager: cache is required")
		return nil, errors.New("cache is required")
	}
	if cfg.Config == nil {
		log.Error("Failed to create ThreadManager: config is required")
		return nil, errors.New("config is required")
	}

	tm := &ThreadManager{
		client: cfg.Client,
		cache:  cfg.Cache,
		config: cfg.Config,
		logger: log,
	}

	log.Info("ThreadManager initialized successfully")
	return tm, nil
}

func (tm *ThreadManager) CreateThread(ctx context.Context) (*aiyou.ConversationThread, error) {
	tm.logger.Debug("Attempting to create new thread")

	if !tm.client.IsAuthenticated() {
		tm.logger.Error("Cannot create thread: not authenticated")
		return nil, ErrNotAuthenticated
	}

	threadsOutput, err := tm.client.GetUserThreads(ctx, &aiyou.UserThreadsParams{})
	if err != nil {
		tm.logger.Error("Failed to check thread limit: %v", err)
		return nil, fmt.Errorf("failed to check thread limit: %w", err)
	}

	if len(threadsOutput.Threads) >= tm.config.MaxThreads {
		tm.logger.Warning("Thread limit exceeded: %d/%d", len(threadsOutput.Threads), tm.config.MaxThreads)
		return nil, ErrThreadLimitExceeded
	}

	tm.logger.Debug("Creating new thread via API")
	req := aiyou.SaveConversationRequest{
		IsNewAppThread: true,
	}
	resp, err := tm.client.SaveConversation(ctx, req)
	if err != nil {
		tm.logger.Error("Failed to create thread: %v", err)
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	tm.logger.Debug("Retrieving created thread details")
	thread, err := tm.client.GetConversation(ctx, resp.ID)
	if err != nil {
		tm.logger.Error("Failed to get created thread: %v", err)
		return nil, fmt.Errorf("failed to get created thread: %w", err)
	}

	cacheKey := fmt.Sprintf("thread:%s", thread.ID)
	if err := tm.cache.Set(ctx, cacheKey, thread, cache.QueryCache); err != nil {
		tm.logger.Warning("Failed to cache thread: %v", err)
	} else {
		tm.logger.Debug("Thread cached successfully")
	}

	tm.logger.Info("Successfully created new thread with ID: %s", thread.ID)
	return thread, nil
}

func (tm *ThreadManager) GetThread(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	tm.logger.Debug("Retrieving thread: %s", threadID)

	if threadID == "" {
		tm.logger.Error("Invalid thread ID: empty string")
		return nil, ErrInvalidThreadID
	}

	if !tm.client.IsAuthenticated() {
		tm.logger.Error("Cannot get thread: not authenticated")
		return nil, ErrNotAuthenticated
	}

	cacheKey := fmt.Sprintf("thread:%s", threadID)
	if entry, exists := tm.cache.Get(ctx, cacheKey); exists {
		if thread, ok := entry.Value.(*aiyou.ConversationThread); ok {
			tm.logger.Debug("Thread retrieved from cache: %s", threadID)
			return thread, nil
		}
		tm.logger.Warning("Cache entry exists but type assertion failed")
	}

	tm.logger.Debug("Fetching thread from API: %s", threadID)
	thread, err := tm.client.GetConversation(ctx, threadID)
	if err != nil {
		tm.logger.Error("Failed to get thread: %v", err)
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if err := tm.cache.Set(ctx, cacheKey, thread, cache.QueryCache); err != nil {
		tm.logger.Warning("Failed to cache thread: %v", err)
	} else {
		tm.logger.Debug("Thread cached successfully")
	}

	tm.logger.Info("Successfully retrieved thread: %s", threadID)
	return thread, nil
}

func (tm *ThreadManager) ListThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	tm.logger.Debug("Listing threads - Page: %d, Search: %s", params.Page, params.Search)

	if !tm.client.IsAuthenticated() {
		tm.logger.Error("Cannot list threads: not authenticated")
		return nil, ErrNotAuthenticated
	}

	if params.Page == 1 && params.Search == "" {
		if entry, exists := tm.cache.Get(ctx, "threads:list"); exists {
			if output, ok := entry.Value.(*aiyou.UserThreadsOutput); ok {
				tm.logger.Debug("Threads list retrieved from cache")
				return output, nil
			}
			tm.logger.Warning("Cache entry exists but type assertion failed")
		}
	}

	tm.logger.Debug("Fetching threads from API")
	output, err := tm.client.GetUserThreads(ctx, params)
	if err != nil {
		tm.logger.Error("Failed to list threads: %v", err)
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	if params.Page == 1 && params.Search == "" {
		if err := tm.cache.Set(ctx, "threads:list", output, cache.QueryCache); err != nil {
			tm.logger.Warning("Failed to cache threads list: %v", err)
		} else {
			tm.logger.Debug("Threads list cached successfully")
		}
	}

	tm.logger.Info("Successfully retrieved %d threads", len(output.Threads))
	return output, nil
}

func (tm *ThreadManager) DeleteThread(ctx context.Context, threadID string) error {
	tm.logger.Debug("Attempting to delete thread: %s", threadID)

	if threadID == "" {
		tm.logger.Error("Invalid thread ID: empty string")
		return ErrInvalidThreadID
	}

	if !tm.client.IsAuthenticated() {
		tm.logger.Error("Cannot delete thread: not authenticated")
		return ErrNotAuthenticated
	}

	tm.logger.Debug("Deleting thread via API: %s", threadID)
	if err := tm.client.DeleteThread(ctx, threadID); err != nil {
		tm.logger.Error("Failed to delete thread: %v", err)
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	cacheKey := fmt.Sprintf("thread:%s", threadID)
	if err := tm.cache.Delete(ctx, cacheKey); err != nil {
		tm.logger.Warning("Failed to remove thread from cache: %v", err)
	}

	if err := tm.cache.Delete(ctx, "threads:list"); err != nil {
		tm.logger.Warning("Failed to invalidate threads list cache: %v", err)
	}

	tm.logger.Info("Successfully deleted thread: %s", threadID)
	return nil
}

func (tm *ThreadManager) SetTestClient(client interfaces.AIClient) {
	tm.logger.Debug("Setting test client")
	tm.client = client
}
