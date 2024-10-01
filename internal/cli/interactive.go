package cli

import (
	"bufio"
	"fmt"
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

func RunInteractiveMode(client *api.AIYOUClient) {
	reader := bufio.NewReader(os.Stdin)
	var conversation []Message

	fmt.Println("Interactive mode activated. Type '/quit' to exit, '/save' to save the conversation.")

	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		timestamp := time.Now().Format("2006-01-02 15:04:05")

		switch input {
		case "/quit":
			fmt.Println("Goodbye!")
			return
		case "/save":
			saveConversation(conversation)
			continue
		}

		conversation = append(conversation, Message{Role: "user", Content: input, Time: time.Now()})

		response, err := client.Chat(input, "")
		if err != nil {
			logger.Error(fmt.Sprintf("Error during chat: %v", err))
			continue
		}

		fmt.Printf("[%s] AI.YOU: %s\n\n", timestamp, response)
		conversation = append(conversation, Message{Role: "assistant", Content: response, Time: time.Now()})
	}
}

func saveConversation(conversation []Message) {
	filename := fmt.Sprintf("conversation_%s.txt", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		logger.Error(fmt.Sprintf("Error creating file: %v", err))
		return
	}
	defer file.Close()

	for _, msg := range conversation {
		_, err := file.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Time.Format("15:04:05"), msg.Role, msg.Content))
		if err != nil {
			logger.Error(fmt.Sprintf("Error writing to file: %v", err))
			return
		}
	}

	fmt.Printf("Conversation saved to %s\n", filename)
}
