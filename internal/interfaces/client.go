package interfaces

import (
	"context"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

type AIClient interface {
	// Méthodes de chat
	CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error)

	// Méthodes de conversation
	SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error)
	GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error)

	// Méthodes d'authentification
	GetToken() string
	SetToken(token string)
	IsAuthenticated() bool
	Authenticate(email, password string) error
	RefreshToken() error

	// Nouvelles méthodes pour les threads
	GetConversation(ctx context.Context, threadID string) (*aiyou.ConversationThread, error)
	GetUserThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error)
	DeleteThread(ctx context.Context, threadID string) error
}
