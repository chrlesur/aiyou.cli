package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChatTest(t *testing.T) (*App, *MockClient, error) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockClient{}

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
		Logger:  logger,
	})
	if err != nil {
		return nil, nil, err
	}

	app.client = mockClient
	app.chatManager.SetTestClient(mockClient)

	return app, mockClient, nil
}

func TestChatCmd_NoAuth(t *testing.T) {
	app, mockClient, err := setupChatTest(t)
	assert.NoError(t, err)

	// S'assurer que nous ne sommes pas connectés
	app.SetLoggedIn(false)
	mockClient.IsAuthenticatedFn = func() bool {
		return false
	}

	// Spécifier explicitement la commande chat
	app.rootCmd.SetArgs([]string{"chat", "Hello"})
	err = app.rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required")
}

func TestChatCmd_EmptyMessage(t *testing.T) {
	app, mockClient, err := setupChatTest(t)
	assert.NoError(t, err)

	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

	// Mock pour GetUserAssistants
	mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
		return &aiyou.AssistantsResponse{
			Members: []aiyou.Assistant{
				{
					ID:   "default-asst",
					Name: "Default Assistant",
				},
			},
		}, nil
	}

	// Spécifier explicitement la commande chat sans message
	app.rootCmd.SetArgs([]string{"chat"})
	err = app.rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestChatCmd_SingleMessage(t *testing.T) {
	app, mockClient, err := setupChatTest(t)
	assert.NoError(t, err)

	// Configuration de base
	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

	// Mock pour GetUserAssistants (pour le getDefaultAssistant)
	mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
		return &aiyou.AssistantsResponse{
			Members: []aiyou.Assistant{
				{
					ID:   "default-asst",
					Name: "Default Assistant",
				},
			},
		}, nil
	}

	// Mock pour la réponse du chat
	mockClient.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
		return &aiyou.ChatCompletionResponse{
			Choices: []aiyou.Choice{
				{
					Message: aiyou.Message{
						Role: "assistant",
						Content: []aiyou.ContentPart{
							{Type: "text", Text: "Test response"},
						},
					},
				},
			},
		}, nil
	}

	// Rediriger stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		w.Close()
		os.Stdout = oldStdout
	}()

	// Exécuter la commande
	app.rootCmd.SetArgs([]string{"chat", "Hello"})
	err = app.rootCmd.Execute()
	assert.NoError(t, err)

	// Lire la sortie
	w.Close()
	var out strings.Builder
	io.Copy(&out, r)
	assert.Contains(t, out.String(), "Test response")
}

func TestChatCmd_WithParameters(t *testing.T) {
	app, mockClient, err := setupChatTest(t)
	assert.NoError(t, err)

	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

	// Rediriger stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	tests := []struct {
		name       string
		args       []string
		setupMock  func(*MockClient)
		wantErr    bool
		wantOutput string
	}{
		{
			name: "valid parameters",
			args: []string{"chat", "--assistant", "test-asst", "--temperature", "0.7", "--max-tokens", "100", "Hello"},
			setupMock: func(mock *MockClient) {
				mock.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return &aiyou.ChatCompletionResponse{
						Choices: []aiyou.Choice{
							{
								Message: aiyou.Message{
									Role: "assistant",
									Content: []aiyou.ContentPart{
										{Type: "text", Text: "Hello, I'm the assistant"},
									},
								},
							},
						},
					}, nil
				}
			},
			wantErr:    false,
			wantOutput: "Hello, I'm the assistant",
		},
		{
			name:       "invalid temperature",
			args:       []string{"chat", "--assistant", "test-asst", "--temperature", "1.5", "Hello"},
			setupMock:  func(mock *MockClient) {},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name: "streaming with parameters",
			args: []string{"chat", "--assistant", "test-asst", "--stream", "--temperature", "0.8", "--max-tokens", "50", "Hello"},
			setupMock: func(mock *MockClient) {
				mock.CreateChatCompletionStreamFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
					return nil, fmt.Errorf("streaming not supported in tests")
				}
			},
			wantErr:    true,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			app.rootCmd.SetArgs(tt.args)
			err := app.rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.wantOutput != "" {
					w.Close()
					var out strings.Builder
					io.Copy(&out, r)
					assert.Contains(t, out.String(), tt.wantOutput)

					r, w, _ = os.Pipe()
				}
			}
		})
	}
}

// captureOutput est une fonction d'aide pour capturer la sortie stdout
func captureOutput(f func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = oldStdout

	var out strings.Builder
	io.Copy(&out, r)

	return out.String(), err
}

func TestChatCmd_Flags(t *testing.T) {
	app, _, cleanup := setupAuthTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid temperature high",
			args:    []string{"chat", "--temperature", "1.5", "Hello"},
			wantErr: true,
			errMsg:  "temperature must be between 0.0 and 1.0",
		},
		{
			name:    "invalid temperature low",
			args:    []string{"chat", "--temperature", "-0.1", "Hello"},
			wantErr: true,
			errMsg:  "temperature must be between 0.0 and 1.0",
		},
		{
			name:    "invalid top-p",
			args:    []string{"chat", "--top-p", "1.5", "Hello"},
			wantErr: true,
			errMsg:  "top-p must be between 0.0 and 1.0",
		},
		{
			name:    "invalid max-tokens",
			args:    []string{"chat", "--max-tokens", "-1", "Hello"},
			wantErr: true,
			errMsg:  "max-tokens cannot be negative",
		},
		{
			name:    "valid parameters",
			args:    []string{"chat", "--temperature", "0.8", "--top-p", "0.9", "--max-tokens", "100", "Hello"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command for each test
			cmd := app.newChatCmd()

			// Parse flags
			err := cmd.ParseFlags(tt.args[1:]) // Skip "chat" command
			require.NoError(t, err, "Failed to parse flags")

			// Execute PreRunE
			err = cmd.PreRunE(cmd, tt.args)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatCmd_Authentication(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		loggedIn  bool
		setupMock func(*MockClient)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "not authenticated",
			loggedIn: false,
			setupMock: func(mock *MockClient) {
				mock.IsAuthenticatedFn = func() bool { return false }
			},
			wantErr: true,
			errMsg:  "authentication required",
		},
		{
			name:     "authenticated",
			loggedIn: true,
			setupMock: func(mock *MockClient) {
				// Configuration pour toutes les vérifications d'authentification
				mock.IsAuthenticatedFn = func() bool { return true }

				// Mock pour GetUserAssistants
				mock.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						Members: []aiyou.Assistant{
							{ID: "default-asst"},
						},
					}, nil
				}

				// Mock pour CreateChatCompletion
				mock.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return &aiyou.ChatCompletionResponse{
						Choices: []aiyou.Choice{
							{
								Message: aiyou.Message{
									Content: []aiyou.ContentPart{
										{Type: "text", Text: "Test response"},
									},
								},
							},
						},
					}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Réinitialiser l'état
			app.SetLoggedIn(tt.loggedIn)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}
			// S'assurer que le chatManager utilise le client mocké
			app.chatManager.SetTestClient(mockClient)

			app.rootCmd.SetArgs([]string{"chat", "Hello"})
			err := app.rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatCmd_OutputCapture(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	// Configuration globale de base
	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool { return true }
	mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
		return &aiyou.AssistantsResponse{
			Members: []aiyou.Assistant{
				{ID: "default-asst"},
			},
		}, nil
	}

	// S'assurer que le chatManager utilise le client mocké
	app.chatManager.SetTestClient(mockClient)

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	tests := []struct {
		name         string
		args         []string
		setupMock    func(*MockClient)
		wantOutput   string
		noWantOutput string
	}{
		{
			name: "normal output",
			args: []string{"chat", "Hello"},
			setupMock: func(mock *MockClient) {
				mock.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					return &aiyou.ChatCompletionResponse{
						Choices: []aiyou.Choice{
							{
								Message: aiyou.Message{
									Content: []aiyou.ContentPart{
										{Type: "text", Text: "Hello, human!"},
									},
								},
							},
						},
					}, nil
				}
			},
			wantOutput: "Hello, human!",
		},
		// Pour le moment, nous allons retirer le test de streaming car nous ne pouvons pas
		// facilement mocker StreamReader qui est une structure concrète
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			app.rootCmd.SetArgs(tt.args)
			err := app.rootCmd.Execute()
			assert.NoError(t, err)

			w.Close()
			var out strings.Builder
			io.Copy(&out, r)
			output := out.String()

			if tt.wantOutput != "" {
				assert.Contains(t, output, tt.wantOutput)
			}
			if tt.noWantOutput != "" {
				assert.NotContains(t, output, tt.noWantOutput)
			}

			// Reset pipe for next test
			r, w, _ = os.Pipe()
			os.Stdout = w
		})
	}
}
