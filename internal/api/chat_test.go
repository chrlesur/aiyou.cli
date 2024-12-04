package api

import (
	"context"
	"errors"
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

func setupChatTest(t *testing.T) (*ChatManager, *MockClient, cache.Cache, func()) {
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

	cm, err := NewChatManager(ChatManagerConfig{
		Client: mockClient,
		Cache:  memCache,
		Config: cfg,
		Logger: logger,
	})
	require.NoError(t, err)

	cleanup := func() {
		memCache.Close()
	}

	return cm, mockClient, memCache, cleanup
}

func TestChatManager_SendMessage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		message     string
		assistantID string
		setupMock   func(*MockClient)
		wantErr     bool
		errType     error
	}{
		{
			name:        "successful message",
			message:     "Hello AI",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return &aiyou.ChatCompletionResponse{
						Choices: []aiyou.Choice{
							{
								Message: aiyou.Message{
									Role: "assistant",
									Content: []aiyou.ContentPart{
										{Type: "text", Text: "Hello, human!"},
									},
								},
							},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:        "unauthenticated",
			message:     "Hello",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name:        "api error",
			message:     "Hello",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				mockClient.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
		{
			name:        "empty message",
			message:     "",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, mockClient, cache, cleanup := setupChatTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			response, err := cm.SendMessage(ctx, tt.message, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.Choices)
				assert.Equal(t, "assistant", response.Choices[0].Message.Role)
			}
		})
	}
}

func TestChatManager_StartConversation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		assistantID string
		setupMock   func(*MockClient)
		wantErr     bool
		errType     error
	}{
		{
			name:        "successful start",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: false,
		},
		{
			name:        "unauthenticated",
			assistantID: "test-assistant",
			setupMock: func(mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, mockClient, cache, cleanup := setupChatTest(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			err = cm.StartConversation(ctx, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatManager_EndConversation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupState func(*ChatManager, *MockClient)
		wantErr    bool
		errType    error
	}{
		{
			name: "successful end",
			setupState: func(cm *ChatManager, mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
				_ = cm.StartConversation(ctx, "test-assistant")
			},
			wantErr: false,
		},
		{
			name: "no active session",
			setupState: func(cm *ChatManager, mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return true }
			},
			wantErr: true,
			errType: ErrNoActiveSession,
		},
		{
			name: "unauthenticated",
			setupState: func(cm *ChatManager, mockClient *MockClient) {
				mockClient.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, mockClient, cache, cleanup := setupChatTest(t)
			defer cleanup()

			if tt.setupState != nil {
				tt.setupState(cm, mockClient)
			}

			err := cache.Clear(ctx)
			require.NoError(t, err)

			err = cm.EndConversation(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
