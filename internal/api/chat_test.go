package api

import (
	"context"
	"errors"
	"io"
	"sync"
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

// MockClient pour les tests
type MockClient struct {
	mu              sync.RWMutex
	isAuthenticated bool
	mockToken       string
	lastLoginTime   time.Time

	// Fonctions mock
	AuthenticateFn               func(email, password string) error
	GetTokenFn                   func() string
	RefreshTokenFn               func() error
	IsAuthenticatedFn            func() bool
	CreateChatCompletionFn       func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error)
	CreateChatCompletionStreamFn func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error)
	SaveConversationFn           func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error)
	GetUserAssistantsFn          func(ctx context.Context) (*aiyou.AssistantsResponse, error)
}

func NewMockClient() *MockClient {
	return &MockClient{
		AuthenticateFn: func(email, password string) error {
			return nil
		},
		GetTokenFn: func() string {
			return "mock-token"
		},
		RefreshTokenFn: func() error {
			return nil
		},
		IsAuthenticatedFn: func() bool {
			return true
		},
		CreateChatCompletionFn: func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
			return &aiyou.ChatCompletionResponse{
				Choices: []aiyou.Choice{
					{
						Message: aiyou.Message{
							Role: "assistant",
							Content: []aiyou.ContentPart{
								{
									Type: "text",
									Text: "This is a mock response",
								},
							},
						},
					},
				},
			}, nil
		},
		CreateChatCompletionStreamFn: func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
			return &aiyou.StreamReader{}, nil
		},
		SaveConversationFn: func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
			return &aiyou.SaveConversationResponse{}, nil
		},
		GetUserAssistantsFn: func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
			return &aiyou.AssistantsResponse{
				Members: []aiyou.Assistant{{ID: "test-assistant"}},
			}, nil
		},
	}
}

// Implémentation des méthodes de l'interface
func (m *MockClient) CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	if m.CreateChatCompletionFn != nil {
		return m.CreateChatCompletionFn(ctx, messages, assistantID)
	}
	return nil, errors.New("CreateChatCompletionFn not implemented")
}

func (m *MockClient) CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
	if m.CreateChatCompletionStreamFn != nil {
		return m.CreateChatCompletionStreamFn(ctx, messages, assistantID)
	}
	return nil, errors.New("CreateChatCompletionStreamFn not implemented")
}

func (m *MockClient) SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
	if m.SaveConversationFn != nil {
		return m.SaveConversationFn(ctx, req)
	}
	return nil, errors.New("SaveConversationFn not implemented")
}

func (m *MockClient) GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if m.GetUserAssistantsFn != nil {
		return m.GetUserAssistantsFn(ctx)
	}
	return nil, errors.New("GetUserAssistantsFn not implemented")
}

func (m *MockClient) GetToken() string {
	if m.GetTokenFn != nil {
		return m.GetTokenFn()
	}
	return m.mockToken
}

func (m *MockClient) SetToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockToken = token
	m.isAuthenticated = token != ""
	if token != "" {
		m.lastLoginTime = time.Now()
	}
}

func (m *MockClient) IsAuthenticated() bool {
	if m.IsAuthenticatedFn != nil {
		return m.IsAuthenticatedFn()
	}
	return m.isAuthenticated
}

func (m *MockClient) Authenticate(email, password string) error {
	if m.AuthenticateFn != nil {
		return m.AuthenticateFn(email, password)
	}
	return nil
}

func (m *MockClient) RefreshToken() error {
	if m.RefreshTokenFn != nil {
		return m.RefreshTokenFn()
	}
	return nil
}

// Test helpers
func testSetup(t *testing.T) (*ChatManager, *MockClient, cache.Cache, func()) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := NewMockClient()

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
	cm, mockClient, _, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name        string
		message     string
		assistantID string
		setupMocks  func()
		wantErr     bool
		errType     error
	}{
		{
			name:        "successful message",
			message:     "Hello, AI!",
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: false,
		},
		{
			name:        "message too long",
			message:     string(make([]byte, 4001)),
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: true,
			errType: ErrMessageTooLong,
		},
		{
			name:        "unauthenticated",
			message:     "Test message",
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
				}
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name:        "api error",
			message:     "Test message",
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				mockClient.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			resp, err := cm.SendMessage(ctx, tt.message, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Choices)
			}
		})
	}
}

func TestChatManager_StartConversation(t *testing.T) {
	ctx := context.Background()
	cm, mockClient, _, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name        string
		assistantID string
		setupMocks  func()
		wantErr     bool
		errType     error
	}{
		{
			name:        "valid assistant",
			assistantID: "valid-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: false,
		},
		{
			name:        "unauthenticated",
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
				}
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
		{
			name:        "already active session",
			assistantID: "test-assistant",
			setupMocks: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				// Start a session first
				_ = cm.StartConversation(ctx, "other-assistant")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			err := cm.StartConversation(ctx, tt.assistantID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cm.activeSession)
				assert.Equal(t, tt.assistantID, cm.activeSession.AssistantID)
			}
		})
	}
}

func TestChatManager_EndConversation(t *testing.T) {
	ctx := context.Background()
	cm, mockClient, _, cleanup := testSetup(t)
	defer cleanup()

	tests := []struct {
		name       string
		setupState func()
		wantErr    bool
		errType    error
	}{
		{
			name: "end active session",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				_ = cm.StartConversation(ctx, "test-assistant")
			},
			wantErr: false,
		},
		{
			name: "no active session",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: true,
			errType: ErrNoActiveSession,
		},
		{
			name: "unauthenticated",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
				}
			},
			wantErr: true,
			errType: ErrNotAuthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupState != nil {
				tt.setupState()
			}

			err := cm.EndConversation(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Nil(t, cm.activeSession)
			}
		})
	}
}
