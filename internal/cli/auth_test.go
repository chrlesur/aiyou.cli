package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

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
			if email == "test@example.com" && password == "valid_password" {
				return nil
			}
			return errors.New("invalid credentials")
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
func setupAuthTest(t *testing.T) (*App, *MockClient, func()) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := NewMockClient()
	mockClient.IsAuthenticatedFn = func() bool {
		return false
	}

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
		Logger:  logger,
	})
	require.NoError(t, err)

	// Remplacer le client existant par notre mock
	app.client = mockClient

	cleanup := func() {
		app.Close()
	}

	return app, mockClient, cleanup
}

func TestLoginCmd(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		email     string
		password  string
		setupMock func()
		wantErr   bool
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "valid_password",
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
				}
				mockClient.AuthenticateFn = func(email, password string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:     "already logged in",
			email:    "test@example.com",
			password: "valid_password",
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			os.Setenv("AIYOU_EMAIL", tt.email)
			os.Setenv("AIYOU_PASSWORD", tt.password)
			defer func() {
				os.Unsetenv("AIYOU_EMAIL")
				os.Unsetenv("AIYOU_PASSWORD")
			}()

			cmd := app.newLoginCmd()
			err := cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, app.IsLoggedIn())
			}
		})
	}
}

func TestLogoutCmd(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	tests := []struct {
		name       string
		setupState func()
		input      string // Pour simuler la réponse à la confirmation
		wantErr    bool
	}{
		{
			name: "successful logout",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				app.SetLoggedIn(true)
			},
			input:   "y\n", // Répondre "y" à la confirmation
			wantErr: false,
		},
		{
			name: "not logged in",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
				}
				app.SetLoggedIn(false)
			},
			input:   "y\n",
			wantErr: true,
		},
		{
			name: "cancelled logout",
			setupState: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				app.SetLoggedIn(true)
			},
			input:   "n\n", // Répondre "n" à la confirmation
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupState != nil {
				tt.setupState()
			}

			// Simuler l'entrée utilisateur pour la confirmation
			oldStdin := os.Stdin
			tmpfile, err := os.CreateTemp("", "test-input")
			require.NoError(t, err)
			defer os.Remove(tmpfile.Name())

			_, err = tmpfile.Write([]byte(tt.input))
			require.NoError(t, err)
			_, err = tmpfile.Seek(0, 0)
			require.NoError(t, err)
			os.Stdin = tmpfile
			defer func() { os.Stdin = oldStdin }()

			cmd := app.newLogoutCmd()
			err = cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.input == "y\n" {
					// Vérifier le statut de connexion seulement si la déconnexion est confirmée
					assert.False(t, app.IsLoggedIn())
				} else {
					// Si la déconnexion est annulée, le statut ne doit pas changer
					assert.True(t, app.IsLoggedIn())
				}
			}
		})
	}
}
