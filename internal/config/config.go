// Package config manages the aiyou CLI configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	// API configuration
	APIEndpoint string `mapstructure:"api_endpoint"`
	APIKey      string `mapstructure:"api_key"`
	APITimeout  int    `mapstructure:"api_timeout"`

	// Logging configuration
	LogLevel  string `mapstructure:"log_level"`
	LogFormat string `mapstructure:"log_format"`
	LogFile   string `mapstructure:"log_file"`

	// Application configuration
	ConfigDir    string
	CacheDir     string
	MaxThreads   int  `mapstructure:"max_threads"`
	Debug        bool `mapstructure:"debug"`
	ShowProgress bool `mapstructure:"show_progress"`
}

// defaultConfig returns the default configuration.
func defaultConfig() *Config {
	return &Config{
		APIEndpoint:  "https://api.aiyou.cloud/v1",
		APITimeout:   30,
		LogLevel:     "info",
		LogFormat:    "text",
		MaxThreads:   4,
		ShowProgress: true,
	}
}

// New creates a new configuration instance.
func New() (*Config, error) {
	cfg := defaultConfig()

	v := viper.New()
	v.SetEnvPrefix("AIYOU")
	v.AutomaticEnv()

	// Setup config paths
	configDir, err := ensureConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to setup config directory: %w", err)
	}
	cfg.ConfigDir = configDir
	cfg.CacheDir = filepath.Join(configDir, "cache")

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Merge configuration
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate checks if the configuration is valid.
func (c *Config) validate() error {
	if c.APIEndpoint == "" {
		return fmt.Errorf("api_endpoint is required")
	}
	if c.MaxThreads < 1 {
		return fmt.Errorf("max_threads must be greater than 0")
	}
	return nil
}

// ensureConfigDir creates and returns the configuration directory path.
func ensureConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".aiyou")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}

	return configDir, nil
}
