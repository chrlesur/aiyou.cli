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
	a.logger.Debug("Creating chat command")

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
 0 means no limit, positive values set a specific limit`,
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
		a.logger.Debug("Validating chat command flags")

		temp, _ := cmd.Flags().GetFloat32("temperature")
		if temp < 0.0 || temp > 1.0 {
			a.logger.Error("Invalid temperature value: %.2f", temp)
			return fmt.Errorf("temperature must be between 0.0 and 1.0")
		}

		topP, _ := cmd.Flags().GetFloat32("top-p")
		if topP < 0.0 || topP > 1.0 {
			a.logger.Error("Invalid top-p value: %.2f", topP)
			return fmt.Errorf("top-p must be between 0.0 and 1.0")
		}

		maxTokens, _ := cmd.Flags().GetInt("max-tokens")
		if maxTokens < 0 {
			a.logger.Error("Invalid max-tokens value: %d", maxTokens)
			return fmt.Errorf("max-tokens cannot be negative")
		}

		a.logger.Debug("Chat command flags validated successfully")
		return nil
	}

	a.logger.Debug("Chat command created successfully")
	return chatCmd
}

// runChat handles the chat command execution
func (a *App) runChat(cmd *cobra.Command, args []string) error {
	a.logger.Debug("Starting chat execution")

	if !a.IsLoggedIn() {
		a.logger.Error("Chat attempted without authentication")
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

	a.logger.Debug("Chat parameters - Assistant: %s, Interactive: %v, Stream: %v, Temperature: %.2f, TopP: %.2f, MaxTokens: %d",
		assistantID, interactive, stream, temperature, topP, maxTokens)

	params := api.ChatParameters{
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
	}

	if err := params.Validate(); err != nil {
		a.logger.Error("Invalid chat parameters: %v", err)
		return err
	}

	// Validate assistant ID
	if assistantID == "" {
		a.logger.Debug("No assistant specified, getting default assistant")
		defaultAssistant, err := a.getDefaultAssistant(ctx)
		if err != nil {
			a.logger.Error("Failed to get default assistant: %v", err)
			return fmt.Errorf("no assistant specified and couldn't get default: %w", err)
		}
		assistantID = defaultAssistant
		a.logger.Debug("Using default assistant: %s", assistantID)
	}

	if interactive {
		a.logger.Info("Starting interactive chat session")
		return a.runInteractiveChat(ctx, assistantID, stream, params)
	}

	if len(args) == 0 {
		a.logger.Error("No message provided in non-interactive mode")
		return fmt.Errorf("message is required when not in interactive mode")
	}

	message := strings.Join(args, " ")
	a.logger.Debug("Sending single message in non-interactive mode")
	return a.sendSingleMessage(ctx, message, assistantID, stream, params)
}

// sendSingleMessage handles sending a single message and receiving the response
func (a *App) sendSingleMessage(ctx context.Context, message, assistantID string, stream bool, params api.ChatParameters) error {
	a.logger.Debug("Processing single message - Stream: %v, Assistant: %s", stream, assistantID)

	if stream {
		return a.handleStreamingResponse(ctx, message, assistantID)
	}

	response, err := a.chatManager.SendMessageWithParams(ctx, message, assistantID, params)
	if err != nil {
		a.logger.Error("Failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	if len(response.Choices) == 0 {
		a.logger.Error("Received empty response from assistant")
		return fmt.Errorf("received empty response from assistant")
	}

	a.logger.Debug("Message sent successfully, displaying response")
	fmt.Printf("\n%s\n", response.Choices[0].Message.Content[0].Text)
	return nil
}

// handleStreamingResponse manages streaming responses from the assistant
func (a *App) handleStreamingResponse(ctx context.Context, message, assistantID string) error {
	a.logger.Debug("Starting streaming response for assistant: %s", assistantID)

	stream, err := a.chatManager.SendMessageStream(ctx, message, assistantID)
	if err != nil {
		a.logger.Error("Failed to start message stream: %v", err)
		return fmt.Errorf("failed to start message stream: %w", err)
	}
	defer stream.Close()

	errChan := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				a.logger.Debug("Stream context cancelled")
				errChan <- ctx.Err()
				return
			default:
				chunk, err := stream.ReadChunk()
				if err != nil {
					if err == io.EOF {
						a.logger.Debug("Stream completed normally")
						return
					}
					a.logger.Error("Error reading stream chunk: %v", err)
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

	select {
	case err := <-errChan:
		a.logger.Error("Stream error: %v", err)
		return err
	case <-done:
		a.logger.Debug("Stream completed successfully")
		fmt.Println()
		return nil
	}
}

// runInteractiveChat manages an interactive chat session
func (a *App) runInteractiveChat(ctx context.Context, assistantID string, stream bool, params api.ChatParameters) error {
	a.logger.Info("Starting interactive chat session")
	a.logger.Debug("Interactive chat parameters - Assistant: %s, Stream: %v, Temperature: %.2f, TopP: %.2f, MaxTokens: %d",
		assistantID, stream, params.Temperature, params.TopP, params.MaxTokens)

	err := a.chatManager.StartConversation(ctx, assistantID)
	if err != nil {
		a.logger.Error("Failed to start chat session: %v", err)
		return fmt.Errorf("failed to start chat session: %w", err)
	}

	defer func() {
		if err := a.chatManager.EndConversation(ctx); err != nil {
			a.logger.Error("Failed to properly end chat session: %v", err)
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
			a.logger.Debug("User requested to exit interactive session")
			break
		}

		if input == "" {
			continue
		}

		a.logger.Debug("Processing interactive input")
		if stream {
			err = a.handleStreamingResponse(ctx, input, assistantID)
		} else {
			err = a.sendSingleMessage(ctx, input, assistantID, false, params)
		}

		if err != nil {
			a.logger.Error("Error processing message: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		a.logger.Error("Scanner error: %v", err)
	}

	return scanner.Err()
}

// getDefaultAssistant attempts to get a default assistant ID
func (a *App) getDefaultAssistant(ctx context.Context) (string, error) {
	a.logger.Debug("Attempting to get default assistant")

	if entry, exists := a.cache.Get(ctx, "default_assistant"); exists {
		if id, ok := entry.Value.(string); ok {
			a.logger.Debug("Found default assistant in cache: %s", id)
			return id, nil
		}
		a.logger.Warning("Cache entry exists but type assertion failed")
	}

	assistants, err := a.chatManager.GetAssistants(ctx)
	if err != nil {
		a.logger.Error("Failed to get assistants: %v", err)
		return "", fmt.Errorf("failed to get assistants: %w", err)
	}

	if len(assistants.Members) == 0 {
		a.logger.Error("No assistants available")
		return "", fmt.Errorf("no assistants available")
	}

	defaultID := assistants.Members[0].ID
	a.cache.Set(ctx, "default_assistant", defaultID, "assistant")
	a.logger.Debug("Set new default assistant: %s", defaultID)

	return defaultID, nil
}
