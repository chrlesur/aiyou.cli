package api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/auth/storage"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/internal/interfaces"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired = errors.New("token expired")
	ErrStorageFailure = errors.New("token storage failure")
	ErrTokenCorrupted = errors.New("stored token is corrupted")
)

type AuthManager struct {
	mu sync.RWMutex
	client interfaces.AIClient
	cfg *config.Config
	store storage.TokenStore
	token string
	expiry time.Time
	logger *logger.Logger
}

func NewAuthManager(client *aiyou.Client, cfg *config.Config) (*AuthManager, error) {
	log := logger.GetLogger()
	log.Debug("Initializing AuthManager")

	if client == nil {
		log.Error("Failed to create AuthManager: client is required")
		return nil, errors.New("client is required")
	}
	if cfg == nil {
		log.Error("Failed to create AuthManager: config is required")
		return nil, errors.New("config is required")
	}

	adapter := &ClientAdapter{
		Client: client,
		isAuthenticated: false,
		logger: log,
	}

	tokenPath := cfg.TokenStorePath
	if tokenPath == "" {
		tokenPath = filepath.Join(cfg.ConfigDir, "auth", "tokens.dat")
		log.Debug("Using default token path: %s", tokenPath)
	}

	key := make([]byte, storage.KeySize)
	if _, err := rand.Read(key); err != nil {
		log.Error("Failed to generate storage key: %v", err)
		return nil, fmt.Errorf("failed to generate storage key: %w", err)
	}

	store, err := storage.NewFileStore(tokenPath, key)
	if err != nil {
		log.Error("Failed to initialize token storage: %v", err)
		return nil, fmt.Errorf("failed to initialize token storage: %w", err)
	}

	log.Info("AuthManager initialized successfully")
	return &AuthManager{
		client: adapter,
		cfg: cfg,
		store: store,
		logger: log,
	}, nil
}

type ClientAdapter struct {
	*aiyou.Client
	isAuthenticated bool
	token string
	mu sync.RWMutex
	logger *logger.Logger
}

func (ca *ClientAdapter) IsAuthenticated() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	isAuth := ca.isAuthenticated && ca.token != "" && ca.Client != nil
	ca.logger.Debug("Checking authentication status: %v", isAuth)
	return isAuth
}

func (ca *ClientAdapter) GetToken() string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	if !ca.isAuthenticated || ca.token == "" {
		ca.logger.Debug("No valid token found")
		return ""
	}
	ca.logger.Debug("Retrieved valid token")
	return ca.token
}

func (ca *ClientAdapter) SetToken(token string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.token = token
	ca.isAuthenticated = token != ""
	ca.logger.Debug("Token updated, authentication status: %v", ca.isAuthenticated)
}

func (ca *ClientAdapter) Authenticate(email, password string) error {
	ca.logger.Debug("Attempting authentication for email: %s", email)
	
	newClient, err := aiyou.NewClient(email, password)
	if err != nil {
		ca.logger.Error("Authentication failed: %v", err)
		return err
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.Client = newClient
	ca.token = fmt.Sprintf("token-%s-%d", email, time.Now().Unix())
	ca.isAuthenticated = true

	ca.logger.Info("Authentication successful for email: %s", email)
	return nil
}

func (ca *ClientAdapter) RefreshToken() error {
	// À implémenter selon les besoins
	ca.logger.Debug("Token refresh requested (not implemented)")
	return nil
}

func (a *AuthManager) Login(ctx context.Context, email, password string) error {
	a.logger.Debug("Processing login request for email: %s", email)

	if ctx == nil {
		a.logger.Error("Login failed: context is required")
		return errors.New("context is required")
	}
	if email == "" || password == "" {
		a.logger.Error("Login failed: email and password are required")
		return errors.New("email and password are required")
	}

	if err := a.client.Authenticate(email, password); err != nil {
		if err.Error() == "invalid credentials" {
			a.logger.Error("Login failed: invalid credentials for email: %s", email)
			return ErrInvalidCredentials
		}
		a.logger.Error("Authentication failed: %v", err)
		return fmt.Errorf("authentication failed: %w", err)
	}

	token := a.client.GetToken()
	if token == "" {
		a.logger.Error("No token received after authentication")
		return fmt.Errorf("no token received after authentication")
	}

	a.mu.Lock()
	a.token = token
	a.expiry = time.Now().Add(24 * time.Hour)
	a.mu.Unlock()

	tokenData := storage.TokenData{
		Token: token,
		Expiry: a.expiry,
		CreatedAt: time.Now(),
	}

	if err := a.store.Save(ctx, tokenData); err != nil {
		a.logger.Error("Failed to save token: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	a.logger.Info("Login successful for email: %s", email)
	return nil
}

func (a *AuthManager) GetToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.logger.Debug("Retrieving token")

	if time.Now().After(a.expiry) {
		a.token = ""
		a.logger.Warning("Token has expired")
		return "", ErrTokenExpired
	}

	if a.token != "" {
		a.logger.Debug("Using in-memory token")
		return a.token, nil
	}

	tokenData, err := a.store.Load(ctx)
	if err != nil {
		a.logger.Error("Failed to load token from storage: %v", err)
		return "", fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if tokenData == nil {
		a.logger.Warning("No token found in storage")
		return "", ErrNotAuthenticated
	}

	if time.Now().After(tokenData.Expiry) {
		a.logger.Warning("Stored token has expired")
		_ = a.store.Clear(ctx)
		return "", ErrTokenExpired
	}

	a.token = tokenData.Token
	a.expiry = tokenData.Expiry
	a.logger.Debug("Successfully retrieved token from storage")

	return a.token, nil
}

func (a *AuthManager) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.logger.Debug("Checking authentication status")

	if time.Now().After(a.expiry) {
		a.logger.Debug("Token has expired, not authenticated")
		return false
	}

	if a.token != "" {
		a.logger.Debug("Valid token in memory, authenticated")
		return true
	}

	tokenData, err := a.store.Load(context.Background())
	if err != nil || tokenData == nil {
		a.logger.Debug("No valid token in storage, not authenticated")
		return false
	}

	if time.Now().After(tokenData.Expiry) {
		a.logger.Debug("Stored token has expired, not authenticated")
		return false
	}

	a.logger.Debug("Valid token found in storage, authenticated")
	return true
}

func (a *AuthManager) Logout() error {
	a.logger.Debug("Processing logout request")

	a.mu.Lock()
	defer a.mu.Unlock()

	a.token = ""
	a.expiry = time.Time{}

	if err := a.store.Clear(context.Background()); err != nil {
		a.logger.Error("Failed to clear token storage: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	a.logger.Info("Logout successful")
	return nil
}

func (a *AuthManager) tryAutoLogin(ctx context.Context) error {
	a.logger.Debug("Attempting auto-login")

	tokenData, err := a.store.Load(ctx)
	if err != nil {
		a.logger.Error("Failed to load token for auto-login: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if tokenData == nil || time.Now().After(tokenData.Expiry) {
		a.logger.Warning("No valid token found for auto-login")
		return ErrTokenExpired
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.token = tokenData.Token
	a.expiry = tokenData.Expiry

	a.logger.Info("Auto-login successful")
	return nil
}

func (a *AuthManager) loadTokenFromStore(ctx context.Context) (*storage.TokenData, error) {
	a.logger.Debug("Loading token from storage")

	tokenData, err := a.store.Load(ctx)
	if err != nil {
		a.logger.Error("Failed to load token from storage: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if tokenData != nil {
		a.logger.Debug("Successfully loaded token from storage")
	} else {
		a.logger.Debug("No token found in storage")
	}

	return tokenData, nil
}

func (a *AuthManager) saveTokenToStore(ctx context.Context, token string, expiry time.Time) error {
	a.logger.Debug("Saving token to storage")

	tokenData := storage.TokenData{
		Token: token,
		Expiry: expiry,
		CreatedAt: time.Now(),
	}
	
	if err := a.store.Save(ctx, tokenData); err != nil {
		a.logger.Error("Failed to save token to storage: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	a.logger.Debug("Successfully saved token to storage")
	return nil
}

func (a *AuthManager) updateInMemoryToken(token string, expiry time.Time) {
	a.logger.Debug("Updating in-memory token")
	
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
	a.expiry = expiry

	a.logger.Debug("In-memory token updated")
}