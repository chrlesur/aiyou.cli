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

func setupChatTest(t *testing.T) (*App, *MockClient, func()) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := NewMockClient()

	// Configuration par défaut du mock
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

	mockClient.GetUserAssistantsFn = func(ctx context.Context) (*aiyou.AssistantsResponse, error) {
		return &aiyou.AssistantsResponse{
			Members: []aiyou.Assistant{
				{ID: "test-assistant"},
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
							{
								Type: "text",
								Text: "This is a mock response",
							},
						},
					},
				},
			},
		}, nil
	}

	mockClient.CreateChatCompletionStreamFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
		return &aiyou.StreamReader{}, nil
	}

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
		Logger:  logger,
	})
	require.NoError(t, err)

	app.client = mockClient
	app.chatManager.SetTestClient(mockClient)
	app.SetLoggedIn(true)

	cleanup := func() {
		app.Close()
	}

	return app, mockClient, cleanup
}

func TestChatCmd_SingleMessage(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	tests := []struct {
		name       string
		args       []string
		setupMock  func()
		wantOutput string
		wantErr    bool
	}{
		{
			name: "successful single message",
			args: []string{"chat", "Hello AI"},
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantOutput: "This is a mock response",
			wantErr:    false,
		},
		{
			name:    "empty message",
			args:    []string{"chat"},
			wantErr: true,
		},
		{
			name: "unauthenticated",
			args: []string{"chat", "Hello"},
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
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

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute command
			app.rootCmd.SetArgs(tt.args)
			err := app.rootCmd.Execute()

			// Restore stdout and get output
			w.Close()
			os.Stdout = oldStdout
			var output strings.Builder
			io.Copy(&output, r)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantOutput != "" {
					assert.Contains(t, output.String(), tt.wantOutput)
				}
			}
		})
	}
}

func TestChatCmd_Interactive(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		input     string
		setupMock func()
		wantErr   bool
	}{
		{
			name:  "successful interactive session",
			input: "Hello AI\nexit\n",
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: false,
		},
		{
			name:  "unauthenticated session",
			input: "Hello\nexit\n",
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return false
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

			// Setup input
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

			// Execute command
			app.rootCmd.SetArgs([]string{"chat", "-i"})
			err = app.rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatCmd_AssistantSelection(t *testing.T) {
	app, mockClient, cleanup := setupChatTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		args      []string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "specific assistant",
			args: []string{"chat", "-a", "test-assistant", "Hello"},
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
			},
			wantErr: false,
		},
		{
			name: "invalid assistant",
			args: []string{"chat", "-a", "invalid-assistant", "Hello"},
			setupMock: func() {
				mockClient.IsAuthenticatedFn = func() bool {
					return true
				}
				mockClient.CreateChatCompletionFn = func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
					if assistantID == "invalid-assistant" {
						return nil, fmt.Errorf("invalid assistant ID")
					}
					return nil, nil
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

			app.rootCmd.SetArgs(tt.args)
			err := app.rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
