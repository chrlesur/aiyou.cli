package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatCmd_SingleMessage(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()
	app.SetLoggedIn(true)

	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}

	tests := []struct {
		name      string
		args      []string
		setupMock func()
		wantErr   bool
	}{
		{
			name:    "empty message",
			args:    []string{"chat"},
			wantErr: true,
		},
		{
			name: "no authentication",
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
