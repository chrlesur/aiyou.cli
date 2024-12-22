package api

import (
	"context"
	"sync"
	"time"

	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

// MockClient implémente l'interface interfaces.AIClient pour les tests
type MockClient struct {
	mu              sync.RWMutex
	isAuthenticated bool
	mockToken       string
	lastLoginTime   time.Time
	logger          *logger.Logger

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

func NewMockClient() *MockClient {
	return &MockClient{
		logger: logger.GetLogger(),
	}
}

// Implémentation des méthodes de l'interface

func (m *MockClient) GetToken() string {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: GetToken called")
	if m.GetTokenFn != nil {
		token := m.GetTokenFn()
		m.logger.Debug("MockClient: GetToken returning custom token")
		return token
	}
	m.logger.Debug("MockClient: GetToken returning default mock token")
	return m.mockToken
}

func (m *MockClient) SetToken(token string) {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: SetToken called with token length: %d", len(token))
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mockToken = token
	m.isAuthenticated = token != ""
	m.logger.Debug("MockClient: Authentication status set to: %v", m.isAuthenticated)
}

func (m *MockClient) IsAuthenticated() bool {
	if m.logger != nil {
		m.logger.Debug("MockClient: IsAuthenticated called")
	}

	if m.IsAuthenticatedFn != nil {
		auth := m.IsAuthenticatedFn()
		if m.logger != nil {
			m.logger.Debug("MockClient: IsAuthenticated returning custom value: %v", auth)
		}
		return auth
	}

	if m.logger != nil {
		m.logger.Debug("MockClient: IsAuthenticated returning default true")
	}
	return true
}

func (m *MockClient) Authenticate(email, password string) error {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: Authenticate called with email: %s", email)
	if m.AuthenticateFn != nil {
		err := m.AuthenticateFn(email, password)
		if err != nil {
			m.logger.Error("MockClient: Authentication failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Authentication successful")
		}
		return err
	}
	m.logger.Debug("MockClient: Authentication succeeded with default behavior")
	return nil
}

func (m *MockClient) RefreshToken() error {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: RefreshToken called")
	if m.RefreshTokenFn != nil {
		err := m.RefreshTokenFn()
		if err != nil {
			m.logger.Error("MockClient: Token refresh failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Token refresh successful")
		}
		return err
	}
	m.logger.Debug("MockClient: Token refresh succeeded with default behavior")
	return nil
}

func (m *MockClient) CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: CreateChatCompletion called with assistant ID: %s, messages count: %d", assistantID, len(messages))
	if m.CreateChatCompletionFn != nil {
		resp, err := m.CreateChatCompletionFn(ctx, messages, assistantID)
		if err != nil {
			m.logger.Error("MockClient: Chat completion failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Chat completion successful")
		}
		return resp, err
	}
	m.logger.Debug("MockClient: Chat completion returning nil with default behavior")
	return nil, nil
}

func (m *MockClient) CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
	// Ensure logger is initialized
	m.mu.Lock()
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	m.mu.Unlock()

	m.logger.Debug("MockClient: CreateChatCompletionStream called with assistant: %s", assistantID)
	
	if m.CreateChatCompletionStreamFn != nil {
		response, err := m.CreateChatCompletionStreamFn(ctx, messages, assistantID)
		if err != nil {
			m.logger.Error("MockClient: CreateChatCompletionStream failed: %v", err)
			return nil, err
		}
		m.logger.Debug("MockClient: CreateChatCompletionStream succeeded")
		return response, nil
	}
	
	m.logger.Debug("MockClient: Using default response (nil)")
	return nil, nil
}

func (m *MockClient) SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: SaveConversation called with thread ID: %s", req.ThreadID)
	if m.SaveConversationFn != nil {
		resp, err := m.SaveConversationFn(ctx, req)
		if err != nil {
			m.logger.Error("MockClient: Save conversation failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Conversation saved successfully")
		}
		return resp, err
	}
	m.logger.Debug("MockClient: Save conversation returning nil with default behavior")
	return nil, nil
}

func (m *MockClient) GetConversation(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: GetConversation called for thread ID: %s", threadID)
	if m.GetConversationFn != nil {
		conv, err := m.GetConversationFn(ctx, threadID)
		if err != nil {
			m.logger.Error("MockClient: Get conversation failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Conversation retrieved successfully")
		}
		return conv, err
	}
	m.logger.Debug("MockClient: Get conversation returning nil with default behavior")
	return nil, nil
}

func (m *MockClient) GetUserThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	m.logger.Debug("MockClient: GetUserThreads called with page: %d, itemsPerPage: %d", params.Page, params.ItemsPerPage)
	if m.GetUserThreadsFn != nil {
		threads, err := m.GetUserThreadsFn(ctx, params)
		if err != nil {
			m.logger.Error("MockClient: Get user threads failed: %v", err)
		} else {
			m.logger.Debug("MockClient: User threads retrieved successfully")
		}
		return threads, err
	}
	m.logger.Debug("MockClient: Get user threads returning nil with default behavior")
	return nil, nil
}

func (m *MockClient) DeleteThread(ctx context.Context, threadID string) error {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: DeleteThread called for thread ID: %s", threadID)
	if m.DeleteThreadFn != nil {
		err := m.DeleteThreadFn(ctx, threadID)
		if err != nil {
			m.logger.Error("MockClient: Delete thread failed: %v", err)
		} else {
			m.logger.Debug("MockClient: Thread deleted successfully")
		}
		return err
	}
	m.logger.Debug("MockClient: Delete thread succeeded with default behavior")
	return nil
}

func (m *MockClient) GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	if m.logger == nil {
		m.logger = logger.GetLogger()
	}
	
	m.logger.Debug("MockClient: GetUserAssistants called")
	if m.GetUserAssistantsFn != nil {
		resp, err := m.GetUserAssistantsFn(ctx)
		if err != nil {
			m.logger.Error("MockClient: Get user assistants failed: %v", err)
		} else {
			m.logger.Debug("MockClient: User assistants retrieved successfully, count: %d", len(resp.Members))
		}
		return resp, err
	}
	m.logger.Debug("MockClient: Get user assistants returning nil with default behavior")
	return nil, nil
}

