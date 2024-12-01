package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Test de la configuration par défaut
	cfg, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.LogLevel)

	// Test avec variable d'environnement
	os.Setenv("AIYOU_LOG_LEVEL", "debug")
	cfg, err = New()
	assert.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				APIEndpoint: "https://api.aiyou.cloud",
				MaxThreads:  4,
			},
			wantErr: false,
		},
		{
			name: "invalid - empty endpoint",
			cfg: &Config{
				MaxThreads: 4,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
