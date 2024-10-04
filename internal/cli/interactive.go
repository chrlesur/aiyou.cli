package cli

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "strings"
    "time"

    "github.com/chrlesur/aiyou.cli/internal/api"
    "github.com/chrlesur/aiyou.cli/pkg/logger"
)

type Message struct {
    Role    string
    Content string
    Time    time.Time
}

func RunInteractiveMode(client *api.AIYOUClient, systemPrompt string) {
    logger.Info("Starting interactive mode")
    logger.Debug(fmt.Sprintf("System prompt: %s", systemPrompt))

    var conversation []Message

    fmt.Println("Interactive mode activated. Type '/quit' to exit, '/save' to save the conversation.")

    // Créer un nouveau thread pour la conversation
    threadID, err := client.CreateThread()
    if err != nil {
        logger.Error(fmt.Sprintf("Error creating thread: %v", err))
        return
    }
    logger.Debug(fmt.Sprintf("Created new thread with ID: %s", threadID))

    runInteractiveLoop(client, systemPrompt, threadID, &conversation)
}

func runInteractiveLoop(client *api.AIYOUClient, systemPrompt, threadID string, conversation *[]Message) {
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("You: ")
        input, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                logger.Info("EOF detected, exiting interactive mode")
                fmt.Println("\nGoodbye!")
                return
            }
            logger.Error(fmt.Sprintf("Error reading input: %v", err))
            fmt.Printf("Error reading input: %v\n", err)
            continue
        }

        input = strings.TrimSpace(input)
        if input == "" {
            logger.Debug("Empty input received, ignoring")
            continue
        }

        logger.Debug(fmt.Sprintf("User input received: %s", input))

        switch input {
        case "/quit":
            logger.Info("User requested to quit interactive mode")
            fmt.Println("Goodbye!")
            return
        case "/save":
            logger.Info("User requested to save the conversation")
            saveConversation(*conversation)
            continue
        default:
            handleInput(client, input, systemPrompt, threadID, conversation)
        }
    }
}

func handleInput(client *api.AIYOUClient, input, systemPrompt, threadID string, conversation *[]Message) {
    logger.Debug(fmt.Sprintf("Processing user input: %s", input))
    *conversation = append(*conversation, Message{Role: "user", Content: input, Time: time.Now()})

    combinedInput := fmt.Sprintf("%s\n\nUser: %s", systemPrompt, input)
    logger.Debug(fmt.Sprintf("Sending combined input to AI: %s", combinedInput))
    
    response, err := client.ChatInThread(threadID, combinedInput)
    if err != nil {
        logger.Error(fmt.Sprintf("Error during chat: %v", err))
        fmt.Printf("Error: %v\n", err)
        return
    }

    timestamp := time.Now().Format("2006-01-02 15:04:05")
    logger.Debug(fmt.Sprintf("AI.YOU response received: %s", response))
    fmt.Printf("[%s] AI.YOU: %s\n\n", timestamp, response)
    *conversation = append(*conversation, Message{Role: "assistant", Content: response, Time: time.Now()})
}

func saveConversation(conversation []Message) {
	logger.Info("Saving conversation...")
	filename := fmt.Sprintf("aiyou.%s.txt", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		logger.Error(fmt.Sprintf("Error creating file: %v", err))
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	for _, msg := range conversation {
		_, err := file.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Time.Format("2006-01-02 15:04:05"), msg.Role, msg.Content))
		if err != nil {
			logger.Error(fmt.Sprintf("Error writing to file: %v", err))
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}
	}

	logger.Info(fmt.Sprintf("Conversation saved to %s", filename))
	fmt.Printf("Conversation saved to %s\n", filename)
}
