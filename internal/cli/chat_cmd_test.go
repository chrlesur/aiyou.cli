package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChatTest(t *testing.T) (*App, *MockClient, func()) {
	logger.ResetForTest()
	log := logger.GetLogger()
	err := log.Configure(logger.Config{
		LogDir: t.TempDir(),
		Level:  logger.ErrorLevel,
		Silent: true,
	})
	require.NoError(t, err)

	mockClient := &MockClient{}

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)

	app.client = mockClient
	app.chatManager.SetTestClient(mockClient)

	cleanup := func() {
		if app != nil {
			app.Close()
		}
		logger.ResetForTest()
	}

	return app, mockClient, cleanup
}

func TestChatCmd_NoAuth(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	app.SetLoggedIn(false)
	mockClient.IsAuthenticatedFn = func() bool {
		return false
	}

	app.rootCmd.SetArgs([]string{"chat", "Hello"})
	err := app.rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required")
}

func TestChatCmd_EmptyMessage(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

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

	app.rootCmd.SetArgs([]string{"chat"})
	err := app.rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestChatCmd_SingleMessage(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

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

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		w.Close()
		os.Stdout = oldStdout
	}()

	app.rootCmd.SetArgs([]string{"chat", "Hello"})
	err := app.rootCmd.Execute()
	assert.NoError(t, err)

	w.Close()
	var out strings.Builder
	io.Copy(&out, r)
	assert.Contains(t, out.String(), "Test response")
}

func TestChatCmd_WithParameters(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

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
			name:      "invalid temperature",
			args:      []string{"chat", "--assistant", "test-asst", "--temperature", "1.5", "Hello"},
			setupMock: func(mock *MockClient) {},
			wantErr:   true,
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

func TestChatCmd_Flags(t *testing.T) {
	app, _, cleanup := setupChatTest(t)
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
			name:    "valid parameters",
			args:    []string{"chat", "--temperature", "0.8", "--top-p", "0.9", "--max-tokens", "100", "Hello"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := app.newChatCmd()
			err := cmd.ParseFlags(tt.args[1:])
			require.NoError(t, err, "Failed to parse flags")

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
	app, mockClient, cleanup := setupChatTest(t)
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
				mock.IsAuthenticatedFn = func() bool { return true }
				mock.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
					return &aiyou.AssistantsResponse{
						Members: []aiyou.Assistant{
							{ID: "default-asst"},
						},
					}, nil
				}
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
			app.SetLoggedIn(tt.loggedIn)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}
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
