package cli

import (
	"context"
	"testing"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAppTest(t *testing.T) func() {
	logger.ResetForTest()
	log := logger.GetLogger()
	err := log.Configure(logger.Config{
		LogDir: t.TempDir(),
		Level:  logger.ErrorLevel,
		Silent: true,
	})
	require.NoError(t, err)

	return func() {
		logger.ResetForTest()
	}
}

func TestNewApp(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

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
			name: "missing config",
			cfg: &AppConfig{
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "config is required",
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

			if app != nil {
				app.Close()
			}
		})
	}
}

func TestApp_Commands(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)
	defer app.Close()

	expectedCommands := map[string]bool{
		"version":    true,
		"completion": true,
		"config":     true,
		"login":      true,
		"logout":     true,
		"chat":       true,
		"thread":     true,
	}

	for _, cmd := range app.rootCmd.Commands() {
		if _, ok := expectedCommands[cmd.Name()]; !ok {
			t.Errorf("Unexpected command found: %s", cmd.Name())
		}
		delete(expectedCommands, cmd.Name())
	}

	for cmdName := range expectedCommands {
		t.Errorf("Expected command missing: %s", cmdName)
	}
}

func TestApp_Run(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)

	err = app.Close()
	require.NoError(t, err)

	err = app.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is already closed")
}

func TestApp_Close(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)

	err = app.Close()
	require.NoError(t, err)

	err = app.Close()
	assert.NoError(t, err)
}

func TestApp_GetClient(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)
	defer app.Close()

	client := app.GetClient()
	assert.NotNil(t, client)
}

func TestApp_LoginStatus(t *testing.T) {
	cleanup := setupAppTest(t)
	defer cleanup()

	app, err := NewApp(&AppConfig{
		Version: "test-version",
		Config:  &config.Config{},
	})
	require.NoError(t, err)
	defer app.Close()

	assert.False(t, app.IsLoggedIn())

	app.SetLoggedIn(true)
	assert.True(t, app.IsLoggedIn())

	app.SetLoggedIn(false)
	assert.False(t, app.IsLoggedIn())
}
