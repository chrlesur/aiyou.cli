package api

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/cache/memory"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupThreadTest(t *testing.T) (*ThreadManager, *MockClient, cache.Cache, func()) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockClient{}

	cacheConfig := cache.DefaultConfig()
	cacheConfig.CleanupInterval = time.Second
	memCache, err := memory.NewMemoryCache(cacheConfig)
	require.NoError(t, err)

	cfg := &config.Config{
		APIEndpoint: "https://test.api.aiyou.cloud",
		MaxThreads:  4,
	}

	tm, err := NewThreadManager(ThreadManagerConfig{
		Client: mockClient,
		Cache:  memCache,
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	cleanup := func() {
		memCache.Close()
	}

	return tm, mockClient, memCache, cleanup
}

func TestThreadManager_CreateThread(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		setupMock func(*MockClient)
		wantErr   bool
		errType   error
	}{
		{
			name: "successful creation",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserThreadsFn = func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
					return &aiyou.UserThreadsOutput{
						Threads: []aiyou.ConversationThread{},
					}, nil
				}
				mockClient.SaveConversationFn = func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
					return &aiyou.SaveConversationResponse{
						ID: "thread_1",
					}, nil
				}
				mockClient.GetConversationFn = func(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
					return &aiyou.ConversationThread{
						ID:        threadID,
						CreatedAt: time.Now(),
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "unauthenticated",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name: "thread limit exceeded",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserThreadsFn = func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
					threads := make([]aiyou.ConversationThread, 4)
					return &aiyou.UserThreadsOutput{Threads: threads}, nil
				}
			},
			wantErr: true,
			errType: ErrThreadLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, mockClient, cache, cleanup := setupThreadTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			thread, err := tm.CreateThread(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, thread)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, thread)
				assert.NotEmpty(t, thread.ID)

				// Vérifier le cache
				cacheKey := fmt.Sprintf("thread:%s", thread.ID)
				entry, exists := cache.Get(ctx, cacheKey)
				assert.True(t, exists)
				cachedThread, ok := entry.Value.(*aiyou.ConversationThread)
				assert.True(t, ok)
				assert.Equal(t, thread.ID, cachedThread.ID)
			}
		})
	}
}

func TestThreadManager_ListThreads(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		params        *aiyou.UserThreadsParams
		setupMock     func(*MockClient)
		wantErr       bool
		expectedCount int
		checkCache    bool
	}{
		{
			name:   "list first page",
			params: &aiyou.UserThreadsParams{Page: 1, ItemsPerPage: 10},
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserThreadsFn = func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
					threads := make([]aiyou.ConversationThread, 2)
					return &aiyou.UserThreadsOutput{
						Threads:      threads,
						TotalItems:   2,
						ItemsPerPage: params.ItemsPerPage,
						CurrentPage:  params.Page,
					}, nil
				}
			},
			wantErr:       false,
			expectedCount: 2,
			checkCache:    true,
		},
		{
			name:   "unauthenticated",
			params: &aiyou.UserThreadsParams{Page: 1},
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr:       true,
			expectedCount: 0,
			checkCache:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, mockClient, cache, cleanup := setupThreadTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			output, err := tm.ListThreads(ctx, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, output)
				assert.Len(t, output.Threads, tt.expectedCount)

				if tt.checkCache && tt.params.Page == 1 {
					cacheEntry, exists := cache.Get(ctx, "threads:list")
					assert.True(t, exists)
					cachedOutput, ok := cacheEntry.Value.(*aiyou.UserThreadsOutput)
					assert.True(t, ok)
					assert.Len(t, cachedOutput.Threads, tt.expectedCount)
				}
			}
		})
	}
}

func TestThreadManager_DeleteThread(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		threadID  string
		setupMock func(*MockClient)
		wantErr   bool
		errType   error
	}{
		{
			name:     "successful deletion",
			threadID: "thread_1",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.DeleteThreadFn = func(ctx context.Context, threadID string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "empty thread ID",
			threadID: "",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: true,
			errType: ErrInvalidThreadID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, mockClient, cache, cleanup := setupThreadTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			err = tm.DeleteThread(ctx, tt.threadID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)

				// Vérifier que le thread a été supprimé du cache
				cacheKey := fmt.Sprintf("thread:%s", tt.threadID)
				_, exists := cache.Get(ctx, cacheKey)
				assert.False(t, exists)
			}
		})
	}
}
