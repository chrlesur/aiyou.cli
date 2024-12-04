package cli

import (
	"context"
	"io"
	"testing"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *AppConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: &AppConfig{
				Version: "1.0.0",
				Config: &config.Config{
					APIEndpoint: "https://api.test.com",
					LogLevel:    "info",
					MaxThreads:  4,
				},
				Logger: logrus.New(),
			},
			wantErr: false,
		},
		{
			name:    "nil configuration",
			cfg:     nil,
			wantErr: true,
			errMsg:  "app config is required",
		},
		{
			name: "missing logger",
			cfg: &AppConfig{
				Version: "1.0.0",
				Config:  &config.Config{},
			},
			wantErr: true,
			errMsg:  "logger is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, app)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, app)
			assert.Equal(t, tt.cfg.Version, app.version)
			assert.NotNil(t, app.rootCmd)
			assert.NotNil(t, app.cache)
			assert.NotNil(t, app.client)
		})
	}
}

func TestApp_Commands(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
		Logger:  logger,
	})
	require.NoError(t, err)

	// Expected commands
	expectedCommands := map[string]bool{
		"version":    true,
		"completion": true,
		"config":     true,
		"login":      true,
		"logout":     true,
		"chat":       true,
		"thread":     true,
	}

	// Verify all expected commands are registered
	for _, cmd := range app.rootCmd.Commands() {
		if _, ok := expectedCommands[cmd.Name()]; !ok {
			t.Errorf("Unexpected command found: %s", cmd.Name())
		}
		delete(expectedCommands, cmd.Name())
	}

	// Verify no expected commands are missing
	for cmdName := range expectedCommands {
		t.Errorf("Expected command missing: %s", cmdName)
	}
}

func TestApp_Run(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
		Logger:  logger,
	})
	require.NoError(t, err)

	// Test run after close
	err = app.Close()
	require.NoError(t, err)

	err = app.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is already closed")
}
