package cli

import (
	"context"
	"os"
	"testing"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClient pour les tests
type MockClient struct {
	// Fonctions mock pour l'authentification
	AuthenticateFn    func(email, password string) error
	GetTokenFn        func() string
	RefreshTokenFn    func() error
	IsAuthenticatedFn func() bool
	SetTokenFn        func(token string)

	// Fonctions mock pour le chat et les threads
	CreateChatCompletionFn       func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error)
	CreateChatCompletionStreamFn func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error)
	SaveConversationFn           func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error)
	GetConversationFn            func(ctx context.Context, threadID string) (*aiyou.ConversationThread, error)
	GetUserThreadsFn             func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error)
	DeleteThreadFn               func(ctx context.Context, threadID string) error
	GetUserAssistantsFn          func(ctx context.Context) (*aiyou.AssistantsResponse, error)
}

// Implementation des méthodes de l'interface
func (m *MockClient) GetToken() string {
	if m.GetTokenFn != nil {
		return m.GetTokenFn()
	}
	return ""
}

func (m *MockClient) SetToken(token string) {
	if m.SetTokenFn != nil {
		m.SetTokenFn(token)
	}
}

func (m *MockClient) IsAuthenticated() bool {
	if m.IsAuthenticatedFn != nil {
		return m.IsAuthenticatedFn()
	}
	return false
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

func (m *MockClient) CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	if m.CreateChatCompletionFn != nil {
		return m.CreateChatCompletionFn(ctx, messages, assistantID)
	}
	return nil, nil
}

func (m *MockClient) CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
	if m.CreateChatCompletionStreamFn != nil {
		return m.CreateChatCompletionStreamFn(ctx, messages, assistantID)
	}
	return nil, nil
}

func (m *MockClient) SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
	if m.SaveConversationFn != nil {
		return m.SaveConversationFn(ctx, req)
	}
	return nil, nil
}

func (m *MockClient) GetConversation(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	if m.GetConversationFn != nil {
		return m.GetConversationFn(ctx, threadID)
	}
	return nil, nil
}

func (m *MockClient) GetUserThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	if m.GetUserThreadsFn != nil {
		return m.GetUserThreadsFn(ctx, params)
	}
	return nil, nil
}

func (m *MockClient) DeleteThread(ctx context.Context, threadID string) error {
	if m.DeleteThreadFn != nil {
		return m.DeleteThreadFn(ctx, threadID)
	}
	return nil
}

func (m *MockClient) GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if m.GetUserAssistantsFn != nil {
		return m.GetUserAssistantsFn(ctx)
	}
	return nil, nil
}

// setupAuthTest crée un environnement de test pour l'authentification
func setupAuthTest(t *testing.T) (*App, *MockClient, func()) {
	// Reset and configure logger for tests
	logger.ResetForTest()
	log := logger.GetLogger()
	err := log.Configure(logger.Config{
		LogDir: t.TempDir(),
		Level:  logger.ErrorLevel,
		Silent: true,
	})
	require.NoError(t, err)

	log.Debug("Setting up auth test environment")

	mockClient := &MockClient{}

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)

	app.client = mockClient

	cleanup := func() {
		log.Debug("Cleaning up auth test environment")
		app.Close()
		logger.ResetForTest()
	}

	return app, mockClient, cleanup
}

func TestLoginCmd(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	log := logger.GetLogger()

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
			log.Debug("Running test case: %s", tt.name)

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
				log.Debug("Expected error occurred: %v", err)
			} else {
				assert.NoError(t, err)
				assert.True(t, app.IsLoggedIn())
				log.Debug("Login successful")
			}
		})
	}
}

func TestLogoutCmd(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	log := logger.GetLogger()

	tests := []struct {
		name       string
		setupState func()
		input      string
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
			input:   "y\n",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Debug("Running test case: %s", tt.name)

			if tt.setupState != nil {
				tt.setupState()
			}

			// Simulate user input
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
				log.Debug("Expected error occurred: %v", err)
			} else {
				assert.NoError(t, err)
				assert.False(t, app.IsLoggedIn())
				log.Debug("Logout successful")
			}
		})
	}
}
