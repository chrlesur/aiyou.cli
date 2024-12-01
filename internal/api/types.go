package api

import (
	"context"

	"github.com/chrlesur/aiyou.golib"
)

// AIClient defines the interface for interacting with the AI service
type AIClient interface {
	CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error)
	SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error)
	GetToken() string
	SetToken(token string)
	IsAuthenticated() bool
	Authenticate(email, password string) error
	RefreshToken() error
   }
