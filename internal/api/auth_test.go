package api

import (
	"context"
	"testing"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.golib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name    string
		client  *aiyou.Client
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "success with valid inputs",
			client:  &aiyou.Client{},
			cfg:     &config.Config{},
			wantErr: false,
		},
		{
			name:    "error with nil client",
			client:  nil,
			cfg:     &config.Config{},
			wantErr: true,
		},
		{
			name:    "error with nil config",
			client:  &aiyou.Client{},
			cfg:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(tt.client, tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestAuthManager_Login(t *testing.T) {
	cfg := &config.Config{}
	validEmail := "test@example.com"
	validPassword := "password123"

	tests := []struct {
		name     string
		email    string
		password string
		setup    func(*testing.T) *aiyou.Client
		wantErr  bool
	}{
		{
			name:     "successful login",
			email:    validEmail,
			password: validPassword,
			setup: func(t *testing.T) *aiyou.Client {
				return &aiyou.Client{}
			},
			wantErr: false,
		},
		{
			name:     "empty email",
			email:    "",
			password: validPassword,
			setup: func(t *testing.T) *aiyou.Client {
				return &aiyou.Client{}
			},
			wantErr: true,
		},
		{
			name:     "empty password",
			email:    validEmail,
			password: "",
			setup: func(t *testing.T) *aiyou.Client {
				return &aiyou.Client{}
			},
			wantErr: true,
		},
		{
			name:     "nil context",
			email:    validEmail,
			password: validPassword,
			setup: func(t *testing.T) *aiyou.Client {
				return &aiyou.Client{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup(t)
			manager, err := NewAuthManager(client, cfg)
			require.NoError(t, err)

			ctx := context.Background()
			if tt.name == "nil context" {
				ctx = nil
			}

			err = manager.Login(ctx, tt.email, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, manager.IsAuthenticated())
			}
		})
	}
}

func TestAuthManager_Logout(t *testing.T) {
	cfg := &config.Config{}
	client := &aiyou.Client{}

	tests := []struct {
		name    string
		setup   func(*AuthManager)
		wantErr bool
	}{
		{
			name: "successful logout when logged in",
			setup: func(am *AuthManager) {
				ctx := context.Background()
				_ = am.Login(ctx, "test@example.com", "password123")
			},
			wantErr: false,
		},
		{
			name: "successful logout when not logged in",
			setup: func(am *AuthManager) {
				// No setup needed - already logged out
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(client, cfg)
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(manager)
			}

			err = manager.Logout()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, manager.IsAuthenticated())
			}
		})
	}
}

func TestAuthManager_GetToken(t *testing.T) {
	cfg := &config.Config{}
	client := &aiyou.Client{}

	tests := []struct {
		name      string
		setup     func(*AuthManager)
		wantErr   bool
		wantToken string
	}{
		{
			name: "get token when authenticated",
			setup: func(am *AuthManager) {
				ctx := context.Background()
				_ = am.Login(ctx, "test@example.com", "password123")
				// Simuler un token valide
				am.mu.Lock()
				am.token = "test-token"
				am.mu.Unlock()
			},
			wantErr:   false,
			wantToken: "test-token",
		},
		{
			name: "error when not authenticated",
			setup: func(am *AuthManager) {
				// No setup - not authenticated
			},
			wantErr:   true,
			wantToken: "",
		},
		{
			name: "error when token expired",
			setup: func(am *AuthManager) {
				ctx := context.Background()
				_ = am.Login(ctx, "test@example.com", "password123")
				am.mu.Lock()
				am.token = "expired-token"
				am.expiry = time.Now().Add(-1 * time.Hour) // Expired token
				am.mu.Unlock()
			},
			wantErr:   true,
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(client, cfg)
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(manager)
			}

			ctx := context.Background()
			token, err := manager.GetToken(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func TestAuthManager_IsAuthenticated(t *testing.T) {
	cfg := &config.Config{}
	client := &aiyou.Client{}

	tests := []struct {
		name       string
		setup      func(*AuthManager)
		expectAuth bool
	}{
		{
			name: "authenticated with valid token",
			setup: func(am *AuthManager) {
				ctx := context.Background()
				_ = am.Login(ctx, "test@example.com", "password123")
			},
			expectAuth: true,
		},
		{
			name: "not authenticated",
			setup: func(am *AuthManager) {
				// No setup needed
			},
			expectAuth: false,
		},
		{
			name: "expired token",
			setup: func(am *AuthManager) {
				ctx := context.Background()
				_ = am.Login(ctx, "test@example.com", "password123")
				am.expiry = time.Now().Add(-1 * time.Hour)
			},
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(client, cfg)
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(manager)
			}

			assert.Equal(t, tt.expectAuth, manager.IsAuthenticated())
		})
	}
}

func TestAuthManager_ConcurrentAccess(t *testing.T) {
	cfg := &config.Config{}
	client := &aiyou.Client{}
	manager, err := NewAuthManager(client, cfg)
	require.NoError(t, err)

	// Test concurrent login/logout operations
	done := make(chan bool)
	go func() {
		ctx := context.Background()
		_ = manager.Login(ctx, "test@example.com", "password123")
		done <- true
	}()

	go func() {
		_ = manager.Logout()
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// No assertions needed - we're just verifying that there are no race conditions
}
