// Package config manages the aiyou CLI configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	ConfigDir      string
	CacheDir       string
	MaxThreads     int    `mapstructure:"max_threads"`
	Debug          bool   `mapstructure:"debug"`
	ShowProgress   bool   `mapstructure:"show_progress"`
	TokenStorePath string `mapstructure:"token_store_path"`
}

// defaultConfig returns the default configuration.
func defaultConfig() *Config {
	return &Config{
		APIEndpoint:  "https://ai.dragonflygroup.fr",
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
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Définir explicitement les liens avec les variables d'environnement
	v.BindEnv("log_level")
	v.BindEnv("api_endpoint")
	v.BindEnv("api_timeout")
	v.BindEnv("max_threads")
	v.BindEnv("debug")
	v.BindEnv("show_progress")

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

	// Définir les valeurs par défaut
	v.SetDefault("api_endpoint", cfg.APIEndpoint)
	v.SetDefault("api_timeout", cfg.APITimeout)
	v.SetDefault("log_level", cfg.LogLevel)
	v.SetDefault("log_format", cfg.LogFormat)
	v.SetDefault("max_threads", cfg.MaxThreads)
	v.SetDefault("show_progress", cfg.ShowProgress)

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

	// Vérifier les variables d'environnement explicitement
	if envLogLevel := os.Getenv("AIYOU_LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
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

// Load reads and returns the application configuration
func Load() (*Config, error) {
	cfg := &Config{
		APIEndpoint: "https://ai.dragonflygroup.fr",
		LogLevel:    "info", // default value
		MaxThreads:  4,
		Debug:       false,
	}

	v := viper.New()
	v.SetEnvPrefix("AIYOU")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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

	// Handle log level
	if envLogLevel := os.Getenv("AIYOU_LOG_LEVEL"); envLogLevel != "" {
		if isValidLogLevel(envLogLevel) {
			cfg.LogLevel = envLogLevel
		} else {
			// Keep default if invalid
			cfg.LogLevel = "info"
		}
	}

	// Handle other environment variables
	if envAPIEndpoint := os.Getenv("AIYOU_API_ENDPOINT"); envAPIEndpoint != "" {
		cfg.APIEndpoint = envAPIEndpoint
	}

	if envDebug := os.Getenv("AIYOU_DEBUG"); envDebug == "true" {
		cfg.Debug = true
	}

	return cfg, nil
}

func isValidLogLevel(level string) bool {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	return validLevels[strings.ToLower(level)]
}
