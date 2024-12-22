package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/cache/memory"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAssistantTest(t *testing.T) (*AssistantManager, *MockClient, cache.Cache, func()) {
	// Reset and initialize logger
	logger.ResetForTest()
	log := logger.GetLogger()
	err := log.Configure(logger.Config{
		LogDir: t.TempDir(), // Use test's temp directory
		Level: logger.ErrorLevel,
		Silent: true,
	})
	require.NoError(t, err)

	mockClient := NewMockClient()

	cacheConfig := cache.DefaultConfig()
	cacheConfig.CleanupInterval = time.Second
	memCache, err := memory.NewMemoryCache(cacheConfig)
	require.NoError(t, err)

	cfg := &config.Config{
		APIEndpoint: "https://test.api.aiyou.cloud",
		MaxThreads: 4,
	}

	am, err := NewAssistantManager(AssistantManagerConfig{
		Client: mockClient,
		Cache: memCache,
		Config: cfg,
	})
	require.NoError(t, err)

	cleanup := func() {
		memCache.Close()
		// Réinitialiser le logger pour les autres tests
		log.SetSilentMode(false)
		log.SetLevel(logger.InfoLevel)
	}

	return am, mockClient, memCache, cleanup
}

func TestAssistantManager_ListAssistants(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		setupMock func(*MockClient)
		wantErr bool
		expectedCount int
		checkCache bool
	}{
		{
			name: "successful list",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						TotalItems: 2,
						Members: []aiyou.Assistant{
							{
								ID: "asst_1",
								Name: "Assistant 1",
								Model: "gpt-4",
							},
							{
								ID: "asst_2",
								Name: "Assistant 2",
								Model: "gpt-3.5-turbo",
							},
						},
					}, nil
				}
			},
			wantErr: false,
			expectedCount: 2,
			checkCache: true,
		},
		{
			name: "unauthenticated",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			expectedCount: 0,
			checkCache: false,
		},
		{
			name: "api error",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
			expectedCount: 0,
			checkCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am, mockClient, cache, cleanup := setupAssistantTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			assistants, err := am.ListAssistants(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, assistants)
			} else {
				assert.NoError(t, err)
				assert.Len(t, assistants, tt.expectedCount)

				if tt.checkCache {
					cacheEntry, exists := cache.Get(ctx, "assistants:list")
					assert.True(t, exists)
					cachedAssistants, ok := cacheEntry.Value.([]aiyou.Assistant)
					assert.True(t, ok)
					assert.Len(t, cachedAssistants, tt.expectedCount)
				}
			}
		})
	}
}

func TestAssistantManager_GetAssistant(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		assistantID string
		setupMock func(*MockClient)
		wantErr bool
		errType error
	}{
		{
			name: "existing assistant",
			assistantID: "asst_1",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						Members: []aiyou.Assistant{
							{
								ID: "asst_1",
								Name: "Test Assistant",
								Model: "gpt-4",
							},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "empty assistant ID",
			assistantID: "",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: true,
			errType: ErrInvalidAssistantID,
		},
		{
			name: "non-existent assistant",
			assistantID: "asst_999",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						Members: []aiyou.Assistant{},
					}, nil
				}
			},
			wantErr: true,
			errType: ErrAssistantNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am, mockClient, cache, cleanup := setupAssistantTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			assistant, err := am.GetAssistant(ctx, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, assistant)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, assistant)
				assert.Equal(t, tt.assistantID, assistant.ID)
			}
		})
	}
}

func TestAssistantManager_SelectAssistant(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		assistantID string
		setupMock func(*MockClient)
		wantErr bool
		checkCache bool
	}{
		{
			name: "successful selection",
			assistantID: "asst_1",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						Members: []aiyou.Assistant{
							{
								ID: "asst_1",
								Name: "Test Assistant",
								Model: "gpt-4",
							},
						},
					}, nil
				}
			},
			wantErr: false,
			checkCache: true,
		},
		{
			name: "invalid assistant ID",
			assistantID: "",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: true,
			checkCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am, mockClient, cache, cleanup := setupAssistantTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			err = am.SelectAssistant(ctx, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				selected, err := am.GetSelectedAssistant(ctx)
				assert.NoError(t, err)
				assert.Equal(t, tt.assistantID, selected.ID)

				if tt.checkCache {
					cacheEntry, exists := cache.Get(ctx, "selected_assistant")
					assert.True(t, exists)
					assert.Equal(t, tt.assistantID, cacheEntry.Value)
				}
			}
		})
	}
}