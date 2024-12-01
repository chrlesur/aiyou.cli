package interfaces

import (
	"context"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
)

// AIClient définit l'interface pour interagir avec le service AI
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
}
