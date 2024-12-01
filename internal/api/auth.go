// Package api provides the core API interaction layer for the AI.YOU CLI
package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.golib"
)

var (
	// ErrNotAuthenticated is returned when an operation requires authentication
	ErrNotAuthenticated = errors.New("not authenticated")
	// ErrInvalidCredentials is returned when login fails
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrTokenExpired is returned when the current token has expired
	ErrTokenExpired = errors.New("token expired")
)

// AuthManager handles all authentication-related operations including
// token management and session state.
type AuthManager struct {
	mu     sync.RWMutex
	client *aiyou.Client
	cfg    *config.Config
	token  string
	expiry time.Time
}

// NewAuthManager creates a new authentication manager with the provided client and configuration.
// It initializes with default settings for token refresh behavior.
func NewAuthManager(client *aiyou.Client, cfg *config.Config) (*AuthManager, error) {
	if client == nil {
		return nil, errors.New("client is required")
	}
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	return &AuthManager{
		client: client,
		cfg:    cfg,
	}, nil
}

// Login authenticates a user with the provided email and password.
// It returns an error if authentication fails or if the API is unavailable.
func (a *AuthManager) Login(ctx context.Context, email, password string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	// Create a new client with authentication
	client, err := aiyou.NewClient(email, password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Update internal state with new client
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client = client
	a.expiry = time.Now().Add(24 * time.Hour)
	return nil
}

// GetToken returns the current authentication token.
// It returns an error if not authenticated.
func (a *AuthManager) GetToken(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("context is required")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.client == nil {
		return "", ErrNotAuthenticated
	}

	// Vérifier que le token n'a pas expiré
	if time.Now().After(a.expiry) {
		return "", ErrTokenExpired
	}

	return a.token, nil
}

// IsAuthenticated returns true if there is a valid authentication session.
func (a *AuthManager) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.client != nil && time.Now().Before(a.expiry)
}

// Logout removes the current authentication session.
// It returns an error if the logout operation fails.
func (a *AuthManager) Logout() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client == nil {
		return nil // Already logged out
	}

	// Reset all authentication state
	a.client = nil
	a.token = ""
	a.expiry = time.Time{}

	return nil
}
