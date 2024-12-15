package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Backup existing environment variables
	oldLogLevel := os.Getenv("AIYOU_LOG_LEVEL")
	oldAPIEndpoint := os.Getenv("AIYOU_API_ENDPOINT")
	oldDebug := os.Getenv("AIYOU_DEBUG")

	// Cleanup after tests
	defer func() {
		os.Setenv("AIYOU_LOG_LEVEL", oldLogLevel)
		os.Setenv("AIYOU_API_ENDPOINT", oldAPIEndpoint)
		os.Setenv("AIYOU_DEBUG", oldDebug)
	}()

	tests := []struct {
		name        string
		setupEnv    func()
		wantConfig  *Config
		wantErr     bool
		checkConfig func(*testing.T, *Config)
	}{
		{
			name: "default configuration",
			setupEnv: func() {
				os.Unsetenv("AIYOU_LOG_LEVEL")
				os.Unsetenv("AIYOU_API_ENDPOINT")
				os.Unsetenv("AIYOU_DEBUG")
			},
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "info", cfg.LogLevel)
				assert.Equal(t, "https://ai.dragonflygroup.fr", cfg.APIEndpoint)
				assert.Equal(t, 4, cfg.MaxThreads)
				assert.False(t, cfg.Debug)
				assert.NotEmpty(t, cfg.ConfigDir)
				assert.NotEmpty(t, cfg.CacheDir)
			},
		},
		{
			name: "environment variables override",
			setupEnv: func() {
				os.Setenv("AIYOU_LOG_LEVEL", "debug")
				os.Setenv("AIYOU_API_ENDPOINT", "https://test.dragonflygroup.fr")
				os.Setenv("AIYOU_DEBUG", "true")
			},
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "debug", cfg.LogLevel)
				assert.Equal(t, "https://test.dragonflygroup.fr", cfg.APIEndpoint)
				assert.True(t, cfg.Debug)
			},
		},
		{
			name: "invalid log level",
			setupEnv: func() {
				os.Setenv("AIYOU_LOG_LEVEL", "invalid")
				os.Setenv("AIYOU_API_ENDPOINT", "https://ai.dragonflygroup.fr")
			},
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "info", cfg.LogLevel,
					"Expected invalid log level to fallback to 'info'")
			},
		},
		{
			name: "valid log levels",
			setupEnv: func() {
				for _, level := range []string{"debug", "info", "warn", "error"} {
					os.Setenv("AIYOU_LOG_LEVEL", level)
					cfg, err := Load()
					assert.NoError(t, err)
					assert.Equal(t, level, cfg.LogLevel)
				}
			},
			checkConfig: func(t *testing.T, cfg *Config) {
				// Cette vérification est faite dans setupEnv
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			cfg, err := Load()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.checkConfig != nil {
				tt.checkConfig(t, cfg)
			}
		})
	}
}

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				APIEndpoint: "https://ai.dragonflygroup.fr",
				MaxThreads:  4,
				LogLevel:    "info",
				ConfigDir:   "/tmp/.aiyou",
				CacheDir:    "/tmp/.aiyou/cache",
			},
			wantErr: false,
		},
		{
			name: "invalid - empty endpoint",
			cfg: &Config{
				MaxThreads: 4,
				LogLevel:   "info",
			},
			wantErr: true,
			errMsg:  "api_endpoint is required",
		},
		{
			name: "invalid - zero max threads",
			cfg: &Config{
				APIEndpoint: "https://ai.dragonflygroup.fr",
				MaxThreads:  0,
				LogLevel:    "info",
			},
			wantErr: true,
			errMsg:  "max_threads must be greater than 0",
		},
		{
			name: "invalid - negative max threads",
			cfg: &Config{
				APIEndpoint: "https://ai.dragonflygroup.fr",
				MaxThreads:  -1,
				LogLevel:    "info",
			},
			wantErr: true,
			errMsg:  "max_threads must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
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

func TestEnsureConfigDir(t *testing.T) {
	// Create temporary home directory for testing
	tmpHome := t.TempDir()

	// Save and restore original home directory
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE") // Pour Windows
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// Set temporary home for both Unix and Windows
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	configDir, err := ensureConfigDir()
	require.NoError(t, err)

	// Check if config directory was created
	assert.DirExists(t, configDir)

	// Check if it's under the temp home directory
	tmpHomeCleaned := filepath.Clean(tmpHome)
	configDirCleaned := filepath.Clean(configDir)
	assert.True(t, strings.HasPrefix(configDirCleaned, tmpHomeCleaned),
		"Expected config dir to be under %s, got %s", tmpHomeCleaned, configDirCleaned)

	// Check if directory exists and is writable
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "Expected config path to be a directory")

	// Test creating directory again (should not error)
	configDir2, err := ensureConfigDir()
	require.NoError(t, err)
	assert.Equal(t, configDir, configDir2)
}

func TestIsValidLogLevel(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{"debug", true},
		{"DEBUG", true},
		{"info", true},
		{"INFO", true},
		{"warn", true},
		{"error", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := isValidLogLevel(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}
