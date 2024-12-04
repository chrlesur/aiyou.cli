package api

import (
	"context"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

// MockClient implémente l'interface interfaces.AIClient pour les tests
type MockClient struct {
	mu              sync.RWMutex
	isAuthenticated bool
	mockToken       string
	lastLoginTime   time.Time

	// Fonctions mock pour l'authentification
	AuthenticateFn    func(email, password string) error
	GetTokenFn        func() string
	RefreshTokenFn    func() error
	IsAuthenticatedFn func() bool

	// Fonctions mock pour le chat
	CreateChatCompletionFn       func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error)
	CreateChatCompletionStreamFn func(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error)

	// Fonctions mock pour les conversations
	SaveConversationFn func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error)
	GetConversationFn  func(ctx context.Context, threadID string) (*aiyou.ConversationThread, error)
	GetUserThreadsFn   func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error)
	DeleteThreadFn     func(ctx context.Context, threadID string) error

	// Fonctions mock pour les assistants
	GetUserAssistantsFn func(ctx context.Context) (*aiyou.AssistantsResponse, error)
}

// Implémentation des méthodes de l'interface

func (m *MockClient) GetToken() string {
	if m.GetTokenFn != nil {
		return m.GetTokenFn()
	}
	return m.mockToken
}

func (m *MockClient) SetToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockToken = token
	m.isAuthenticated = token != ""
}

func (m *MockClient) IsAuthenticated() bool {
	if m.IsAuthenticatedFn != nil {
		return m.IsAuthenticatedFn()
	}
	return true // Par défaut, retourne true pour les tests
}

func (m *MockClient) Authenticate(email, password string) error {
	if m.AuthenticateFn != nil {
		return m.AuthenticateFn(email, password)
	}
	return nil
}

func (m *MockClient) RefreshToken() error {
	if m.RefreshTokenFn != nil {
		return m.RefreshTokenFn()
	}
	return nil
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	if m.CreateChatCompletionFn != nil {
		return m.CreateChatCompletionFn(ctx, messages, assistantID)
	}
	return nil, nil
}

func (m *MockClient) CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
	if m.CreateChatCompletionStreamFn != nil {
		return m.CreateChatCompletionStreamFn(ctx, messages, assistantID)
	}
	return nil, nil
}

func (m *MockClient) SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
	if m.SaveConversationFn != nil {
		return m.SaveConversationFn(ctx, req)
	}
	return nil, nil
}

func (m *MockClient) GetConversation(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	if m.GetConversationFn != nil {
		return m.GetConversationFn(ctx, threadID)
	}
	return nil, nil
}

func (m *MockClient) GetUserThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	if m.GetUserThreadsFn != nil {
		return m.GetUserThreadsFn(ctx, params)
	}
	return nil, nil
}

func (m *MockClient) DeleteThread(ctx context.Context, threadID string) error {
	if m.DeleteThreadFn != nil {
		return m.DeleteThreadFn(ctx, threadID)
	}
	return nil
}

func (m *MockClient) GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if m.GetUserAssistantsFn != nil {
		return m.GetUserAssistantsFn(ctx)
	}
	return nil, nil
}
