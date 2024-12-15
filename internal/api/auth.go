// internal/api/auth.go
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
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

var (
	// Erreurs existantes
	ErrNotAuthenticated   = errors.New("not authenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")

	// Nouvelles erreurs
	ErrStorageFailure = errors.New("token storage failure")
	ErrTokenCorrupted = errors.New("stored token is corrupted")
)

// AuthManager gère l'authentification et les tokens
type AuthManager struct {
	mu     sync.RWMutex
	client interfaces.AIClient // Changement ici: utilisation de l'interface
	cfg    *config.Config
	store  storage.TokenStore
	token  string
	expiry time.Time
}

// NewAuthManager crée une nouvelle instance d'AuthManager
func NewAuthManager(client *aiyou.Client, cfg *config.Config) (*AuthManager, error) {
	if client == nil {
		return nil, errors.New("client is required")
	}
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	// Créer l'adaptateur
	adapter := &ClientAdapter{
		Client:          client,
		isAuthenticated: false,
	}

	// Déterminer le chemin du stockage
	tokenPath := cfg.TokenStorePath
	if tokenPath == "" {
		tokenPath = filepath.Join(cfg.ConfigDir, "auth", "tokens.dat")
	}

	// Générer une clé de chiffrement pour le stockage
	key := make([]byte, storage.KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate storage key: %w", err)
	}

	// Créer le stockage de tokens
	store, err := storage.NewFileStore(tokenPath, key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token storage: %w", err)
	}

	return &AuthManager{
		client: adapter,
		cfg:    cfg,
		store:  store,
	}, nil
}

// ClientAdapter adapte le client aiyou.Client pour l'interface interfaces.AIClient
type ClientAdapter struct {
	*aiyou.Client
	isAuthenticated bool
	token           string
	mu              sync.RWMutex
}

func (ca *ClientAdapter) IsAuthenticated() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	// Un client n'est authentifié que s'il a un token et que le client existe
	return ca.isAuthenticated && ca.token != "" && ca.Client != nil
}

func (ca *ClientAdapter) GetToken() string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	// Ne retourner le token que si le client est authentifié ET que le token existe
	if !ca.isAuthenticated || ca.token == "" {
		return ""
	}
	return ca.token
}

func (ca *ClientAdapter) SetToken(token string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.token = token
	ca.isAuthenticated = token != ""
}

func (ca *ClientAdapter) Authenticate(email, password string) error {
	newClient, err := aiyou.NewClient(email, password)
	if err != nil {
		return err
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.Client = newClient
	// Générer un token simulé pour les tests
	ca.token = fmt.Sprintf("token-%s-%d", email, time.Now().Unix())
	ca.isAuthenticated = true

	return nil
}

func (ca *ClientAdapter) RefreshToken() error {
	// À implémenter selon les besoins
	return nil
}

// Login authentifie l'utilisateur et stocke le token
func (a *AuthManager) Login(ctx context.Context, email, password string) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	// Authentifier via le client
	if err := a.client.Authenticate(email, password); err != nil {
		if err.Error() == "invalid credentials" {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Récupérer le token
	token := a.client.GetToken()
	if token == "" {
		return fmt.Errorf("no token received after authentication")
	}

	// Mettre à jour l'état interne
	a.mu.Lock()
	a.token = token
	a.expiry = time.Now().Add(24 * time.Hour)
	a.mu.Unlock()

	// Sauvegarder le token
	tokenData := storage.TokenData{
		Token:     token,
		Expiry:    a.expiry,
		CreatedAt: time.Now(),
	}

	if err := a.store.Save(ctx, tokenData); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// GetToken retourne le token actuel
func (a *AuthManager) GetToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Vérifier d'abord l'expiration
	if time.Now().After(a.expiry) {
		// Le token est expiré
		a.token = "" // Nettoyer le token expiré
		return "", ErrTokenExpired
	}

	// Si nous avons un token valide en mémoire
	if a.token != "" {
		return a.token, nil
	}

	// Essayer de charger depuis le stockage
	tokenData, err := a.store.Load(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	// Pas de token stocké
	if tokenData == nil {
		return "", ErrNotAuthenticated
	}

	// Vérifier l'expiration du token stocké
	if time.Now().After(tokenData.Expiry) {
		// Nettoyer le token expiré
		_ = a.store.Clear(ctx)
		return "", ErrTokenExpired
	}

	// Mise à jour de l'état en mémoire avec le token stocké
	a.token = tokenData.Token
	a.expiry = tokenData.Expiry

	return a.token, nil
}

// IsAuthenticated vérifie si l'utilisateur est authentifié
func (a *AuthManager) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Si le token est expiré, on n'est plus authentifié
	if time.Now().After(a.expiry) {
		return false
	}

	// Vérifier l'état en mémoire
	if a.token != "" {
		return true
	}

	// Essayer de charger depuis le stockage
	tokenData, err := a.store.Load(context.Background())
	if err != nil || tokenData == nil {
		return false
	}

	// Vérifier l'expiration du token stocké
	if time.Now().After(tokenData.Expiry) {
		return false
	}

	return true
}

// Logout déconnecte l'utilisateur et nettoie les tokens
func (a *AuthManager) Logout() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Nettoyer l'état en mémoire
	a.token = ""
	a.expiry = time.Time{}

	// Nettoyer le stockage
	if err := a.store.Clear(context.Background()); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// tryAutoLogin tente une reconnexion automatique si possible
func (a *AuthManager) tryAutoLogin(ctx context.Context) error {
	tokenData, err := a.store.Load(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if tokenData == nil || time.Now().After(tokenData.Expiry) {
		return ErrTokenExpired
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.token = tokenData.Token
	a.expiry = tokenData.Expiry

	return nil
}

// loadTokenFromStore charge le token depuis le stockage
func (a *AuthManager) loadTokenFromStore(ctx context.Context) (*storage.TokenData, error) {
	tokenData, err := a.store.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}
	return tokenData, nil
}

// saveTokenToStore sauvegarde le token dans le stockage
func (a *AuthManager) saveTokenToStore(ctx context.Context, token string, expiry time.Time) error {
	tokenData := storage.TokenData{
		Token:     token,
		Expiry:    expiry,
		CreatedAt: time.Now(),
	}
	if err := a.store.Save(ctx, tokenData); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}
	return nil
}

// updateInMemoryToken met à jour le token en mémoire
func (a *AuthManager) updateInMemoryToken(token string, expiry time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
	a.expiry = expiry
}
