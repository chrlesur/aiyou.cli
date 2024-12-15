// Package cli provides the command-line interface implementation
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chrlesur/aiyou.cli/internal/api"
	"github.com/spf13/cobra"
)

// newChatCmd creates and returns the chat command
func (a *App) newChatCmd() *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Chat with an AI assistant",
		Long: `Start a chat session with an AI assistant. You can either:
- Send a single message and get a response
- Start an interactive chat session
- Use message streaming for real-time responses
- Configure advanced response parameters

Advanced Parameters:
  - temperature: Controls randomness in responses (0.0-1.0)
    Lower values make responses more focused and deterministic
    Higher values make responses more creative and diverse

  - top-p: Controls response diversity via nucleus sampling (0.0-1.0)
    Lower values make responses more focused on likely tokens
    Higher values allow for more diverse token selection

  - max-tokens: Limits the length of the response
    0 means no limit, positive values set a specific limit

Examples:
  # Send a simple message
  aiyou chat "What is the capital of France?"

  # Start an interactive session
  aiyou chat -i

  # Enable streaming for real-time responses
  aiyou chat -s "Tell me a story"

  # Use advanced parameters for creative writing
  aiyou chat --temperature 0.8 --top-p 0.9 "Write a creative story"

  # Limit response length
  aiyou chat --max-tokens 100 "Summarize this concept"

  # Combine multiple parameters
  aiyou chat -s --temperature 0.8 --max-tokens 200 "Generate a poem"

  # Use with specific assistant
  aiyou chat -a asst_123 "Hello"`,
		RunE: a.runChat,
	}

	// Basic flags
	chatCmd.Flags().StringP("assistant", "a", "", "ID of the assistant to chat with")
	chatCmd.Flags().BoolP("interactive", "i", false, "Start an interactive chat session")
	chatCmd.Flags().BoolP("stream", "s", false, "Enable real-time response streaming")

	// Advanced parameter flags
	chatCmd.Flags().Float32P("temperature", "t", 0.7, "Response temperature (0.0-1.0)")
	chatCmd.Flags().Float32("top-p", 1.0, "Top-p sampling parameter (0.0-1.0)")
	chatCmd.Flags().Int("max-tokens", 0, "Maximum tokens in response (0 for no limit)")

	// Add flag validations
	chatCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Validate flags first
		temp, _ := cmd.Flags().GetFloat32("temperature")
		if temp < 0.0 || temp > 1.0 {
			return fmt.Errorf("temperature must be between 0.0 and 1.0")
		}

		topP, _ := cmd.Flags().GetFloat32("top-p")
		if topP < 0.0 || topP > 1.0 {
			return fmt.Errorf("top-p must be between 0.0 and 1.0")
		}

		maxTokens, _ := cmd.Flags().GetInt("max-tokens")
		if maxTokens < 0 {
			return fmt.Errorf("max-tokens cannot be negative")
		}

		return nil
	}

	return chatCmd
}

// runChat handles the chat command execution
func (a *App) runChat(cmd *cobra.Command, args []string) error {

	// Check authentication first
	if !a.IsLoggedIn() {
		return fmt.Errorf("authentication required: please login first using 'aiyou login'")
	}

	ctx := cmd.Context()

	// Get flags
	assistantID, _ := cmd.Flags().GetString("assistant")
	interactive, _ := cmd.Flags().GetBool("interactive")
	stream, _ := cmd.Flags().GetBool("stream")
	temperature, _ := cmd.Flags().GetFloat32("temperature")
	topP, _ := cmd.Flags().GetFloat32("top-p")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")

	// Create and validate parameters
	params := api.ChatParameters{
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
	}

	if err := params.Validate(); err != nil {
		return err
	}

	// Validate assistant ID
	if assistantID == "" {
		defaultAssistant, err := a.getDefaultAssistant(ctx)
		if err != nil {
			return fmt.Errorf("no assistant specified and couldn't get default: %w", err)
		}
		assistantID = defaultAssistant
		a.log.Debugf("Using default assistant: %s", assistantID)
	}

	// Handle interactive mode
	if interactive {
		return a.runInteractiveChat(ctx, assistantID, stream, params)
	}

	// Handle single message mode
	if len(args) == 0 {
		return fmt.Errorf("message is required when not in interactive mode")
	}

	message := strings.Join(args, " ")
	return a.sendSingleMessage(ctx, message, assistantID, stream, params)
}

// sendSingleMessage handles sending a single message and receiving the response
func (a *App) sendSingleMessage(ctx context.Context, message, assistantID string, stream bool, params api.ChatParameters) error {
	if stream {
		return a.handleStreamingResponse(ctx, message, assistantID)
	}

	response, err := a.chatManager.SendMessageWithParams(ctx, message, assistantID, params)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("received empty response from assistant")
	}

	fmt.Printf("\n%s\n", response.Choices[0].Message.Content[0].Text)
	return nil
}

// handleStreamingResponse manages streaming responses from the assistant
func (a *App) handleStreamingResponse(ctx context.Context, message, assistantID string) error {
	stream, err := a.chatManager.SendMessageStream(ctx, message, assistantID)
	if err != nil {
		return fmt.Errorf("failed to start message stream: %w", err)
	}
	defer stream.Close()

	// Channel for handling streaming errors
	errChan := make(chan error, 1)
	// Channel for handling cancellation
	done := make(chan struct{})

	// Goroutine for reading the stream
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				chunk, err := stream.ReadChunk()
				if err != nil {
					if err == io.EOF {
						return
					}
					errChan <- err
					return
				}

				if len(chunk.Choices) > 0 && len(chunk.Choices[0].Message.Content) > 0 {
					for _, part := range chunk.Choices[0].Message.Content {
						fmt.Print(part.Text)
					}
				}
			}
		}
	}()

	// Wait for streaming completion or error
	select {
	case err := <-errChan:
		return err
	case <-done:
		fmt.Println() // New line after complete response
		return nil
	}
}

// runInteractiveChat manages an interactive chat session
func (a *App) runInteractiveChat(ctx context.Context, assistantID string, stream bool, params api.ChatParameters) error {
	a.log.Info("Starting interactive chat session. Type 'exit' or press Ctrl+C to end.")
	a.log.Debugf("Using assistant: %s", assistantID)
	if stream {
		a.log.Info("Streaming mode enabled")
	}
	a.log.Debugf("Parameters: temperature=%.2f, top_p=%.2f, max_tokens=%d",
		params.Temperature, params.TopP, params.MaxTokens)

	// Start chat session
	err := a.chatManager.StartConversation(ctx, assistantID)
	if err != nil {
		return fmt.Errorf("failed to start chat session: %w", err)
	}
	defer func() {
		if err := a.chatManager.EndConversation(ctx); err != nil {
			a.log.Errorf("Failed to properly end chat session: %v", err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		if stream {
			err = a.handleStreamingResponse(ctx, input, assistantID)
		} else {
			err = a.sendSingleMessage(ctx, input, assistantID, false, params)
		}

		if err != nil {
			a.log.Errorf("Error: %v", err)
			// Don't exit on error, allow retry
		}
	}

	return scanner.Err()
}

// getDefaultAssistant attempts to get a default assistant ID
func (a *App) getDefaultAssistant(ctx context.Context) (string, error) {
	// First check cache
	if entry, exists := a.cache.Get(ctx, "default_assistant"); exists {
		if id, ok := entry.Value.(string); ok {
			return id, nil
		}
	}

	// If not in cache, get list of assistants
	assistants, err := a.chatManager.GetAssistants(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get assistants: %w", err)
	}

	if len(assistants.Members) == 0 {
		return "", fmt.Errorf("no assistants available")
	}

	// Use first assistant as default
	defaultID := assistants.Members[0].ID

	// Cache the result
	a.cache.Set(ctx, "default_assistant", defaultID, "assistant")

	return defaultID, nil
}
