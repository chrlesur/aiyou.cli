// Package cli provides the command-line interface implementation.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// chatCmd represents the chat command and its subcommands
func (a *App) newChatCmd() *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Chat with an AI assistant",
		Long: `Start a chat session with an AI assistant. You can either:
- Send a single message and get a response
- Start an interactive chat session
- Use message streaming for real-time responses

Examples:
 # Send a single message
 aiyou chat "What is the capital of France?"

 # Start an interactive session
 aiyou chat -i

 # Chat with a specific assistant
 aiyou chat -a asst_123 "Hello!"

 # Enable response streaming
 aiyou chat -s "Tell me a story"`,
		RunE: a.runChat,
	}

	// Add flags
	chatCmd.Flags().StringP("assistant", "a", "", "ID of the assistant to chat with")
	chatCmd.Flags().BoolP("interactive", "i", false, "Start an interactive chat session")
	chatCmd.Flags().BoolP("stream", "s", false, "Enable response streaming")
	chatCmd.Flags().Float32P("temperature", "t", 0.7, "Response temperature (0.0-1.0)")
	chatCmd.Flags().Int("max-tokens", 0, "Maximum tokens in response (0 for no limit)")

	return chatCmd
}

// runChat handles both interactive and single-message chat modes
func (a *App) runChat(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get flags
	assistantID, _ := cmd.Flags().GetString("assistant")
	interactive, _ := cmd.Flags().GetBool("interactive")
	stream, _ := cmd.Flags().GetBool("stream")
	temperature, _ := cmd.Flags().GetFloat32("temperature")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")

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
		return a.runInteractiveChat(ctx, assistantID, stream, temperature, maxTokens)
	}

	// Handle single message mode
	if len(args) == 0 {
		return fmt.Errorf("message is required when not in interactive mode")
	}

	message := strings.Join(args, " ")
	return a.sendSingleMessage(ctx, message, assistantID, stream, temperature, maxTokens)
}

// runInteractiveChat handles interactive chat sessions
func (a *App) runInteractiveChat(ctx context.Context, assistantID string, stream bool, temperature float32, maxTokens int) error {
	a.log.Info("Starting interactive chat session. Type 'exit' or press Ctrl+C to end.")
	a.log.Infof("Using assistant: %s", assistantID)

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

		// Send message and handle response
		if stream {
			err = a.handleStreamingResponse(ctx, input, assistantID, temperature, maxTokens)
		} else {
			err = a.handleSingleResponse(ctx, input, assistantID, temperature, maxTokens)
		}

		if err != nil {
			a.log.Errorf("Error: %v", err)
			// Don't exit on error, allow retry
		}
	}

	return scanner.Err()
}

// sendSingleMessage handles one-off message sending
func (a *App) sendSingleMessage(ctx context.Context, message, assistantID string, stream bool, temperature float32, maxTokens int) error {
	if stream {
		return a.handleStreamingResponse(ctx, message, assistantID, temperature, maxTokens)
	}
	return a.handleSingleResponse(ctx, message, assistantID, temperature, maxTokens)
}

// handleSingleResponse processes a message and displays the response
func (a *App) handleSingleResponse(ctx context.Context, message, assistantID string, temperature float32, maxTokens int) error {
	// Add spinner/progress indicator
	a.startProgress("Thinking")
	defer a.stopProgress()

	response, err := a.chatManager.SendMessage(ctx, message, assistantID)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("received empty response from assistant")
	}

	// Format and display response
	fmt.Printf("\n%s\n", response.Choices[0].Message.Content)

	return nil
}

// handleStreamingResponse handles streaming responses from the assistant
func (a *App) handleStreamingResponse(ctx context.Context, message, assistantID string, temperature float32, maxTokens int) error {
	stream, err := a.chatManager.SendMessageStream(ctx, message, assistantID)
	if err != nil {
		return fmt.Errorf("failed to start message stream: %w", err)
	}
	defer stream.Close()

	fmt.Println() // New line before response

	for {
		chunk, err := stream.ReadChunk()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		if len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.Content) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	fmt.Println() // New line after response
	return nil
}

// getDefaultAssistant returns the ID of the default assistant
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

// startProgress displays a progress indicator
func (a *App) startProgress(message string) {
	// Implementation of progress indicator
	// This could be a spinner or simple dots
	fmt.Printf("%s...", message)
}

// stopProgress stops the progress indicator
func (a *App) stopProgress() {
	fmt.Println()
}
