package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chrlesur/aiyou.cli/internal/api"
	"github.com/chrlesur/aiyou.cli/internal/cli"
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

const VERSION = "0.2-alpha"

var (
	debug             bool
	silent            bool
	aiyouAssistantID  string
	instruction       string
	instructionFile   string
	showAssistantInfo bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Enable silent mode (only show errors)")
	rootCmd.PersistentFlags().StringVarP(&aiyouAssistantID, "assistant", "a", "", "AI.YOU assistant ID")
	rootCmd.PersistentFlags().StringVarP(&instruction, "instruction", "i", "", "Additional instruction for the assistant")
	rootCmd.PersistentFlags().StringVarP(&instructionFile, "instruction-file", "f", "", "File containing additional instructions for the assistant")
	rootCmd.PersistentFlags().BoolVar(&showAssistantInfo, "show-assistant-info", false, "Show assistant information after successful login")

	rootCmd.AddCommand(versionCmd, chatCmd, interactiveCmd)
}

var rootCmd = &cobra.Command{
	Use:   "aiyou.cli",
	Short: "aiyou.cli is a command-line interface for AI.YOU",
	Long: `aiyou.cli allows you to interact with AI.YOU assistants directly from your terminal.
It supports both single message interactions and an interactive chat mode.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetDebugMode(debug)
		logger.SetSilentMode(silent)
		if !silent {
			logger.Info(fmt.Sprintf("aiyou.cli version %s", VERSION))
		}
		if debug && !silent {
			logger.Debug("Debug mode activated")
		}
		err := godotenv.Load()
		if err != nil {
			logger.Warning("Error loading .env file")
		} else if !silent {
			logger.Info(".env file loaded")
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of aiyou.cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aiyou.cli version %s\n", VERSION)
	},
}

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Send a single message to the AI.YOU assistant",
	Long: `Send a single message to the AI.YOU assistant.
The message can be provided as a command-line argument or via stdin.
If no argument is provided and no stdin input is detected, the command will prompt for input.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getAIYOUClient()

		var input string
		if len(args) > 0 {
			input = strings.Join(args, " ")
			logger.Debug(fmt.Sprintf("Input received from command line arguments: %s", input))
		} else {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					logger.Error(fmt.Sprintf("Error reading from stdin: %v", err))
					return err
				}
				input = strings.TrimSpace(string(bytes))
				logger.Debug(fmt.Sprintf("Input received from stdin: %s", input))
			} else {
				if !silent {
					fmt.Print("Enter your message: ")
				}
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					input = scanner.Text()
					logger.Debug(fmt.Sprintf("Input received from user prompt: %s", input))
				} else {
					logger.Error("Error reading input")
					return fmt.Errorf("error reading input")
				}
			}
		}

		if input == "" {
			logger.Error("No input provided")
			return fmt.Errorf("no input provided")
		}

		var finalInstruction string
		if instructionFile != "" {
			loadedInstruction, err := client.LoadInstructionFromFile(instructionFile)
			if err != nil {
				logger.Error(fmt.Sprintf("Error loading instruction file: %v", err))
				return err
			}
			finalInstruction = loadedInstruction
			logger.Debug(fmt.Sprintf("Instruction loaded from file: %s", finalInstruction))
		} else {
			finalInstruction = instruction
			logger.Debug(fmt.Sprintf("Using provided instruction: %s", finalInstruction))
		}

		logger.Debug("Sending chat request to AI.YOU")
		response, err := client.Chat(strings.NewReader(input), finalInstruction)
		if err != nil {
			logger.Error(fmt.Sprintf("Error during chat: %v", err))
			return err
		}
		logger.Debug(fmt.Sprintf("Received response from AI.YOU: %s", response))

		if silent {
			fmt.Println(response)
		} else {
			timestamp := time.Now().Format("2006-01-02 15:04:05")
			fmt.Printf("[%s] AI.YOU: %s\n", timestamp, response)
		}
		return nil
	},
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Start an interactive chat session with the AI.YOU assistant",
	Run: func(cmd *cobra.Command, args []string) {
		client := getAIYOUClient()

		var systemPrompt string
		if instructionFile != "" {
			var err error
			systemPrompt, err = client.LoadInstructionFromFile(instructionFile)
			if err != nil {
				logger.Error(fmt.Sprintf("Error loading instruction file: %v", err))
				fmt.Printf("Error loading instruction file: %v\n", err)
				return
			}
		} else {
			systemPrompt = instruction
		}

		logger.Debug(fmt.Sprintf("System prompt for interactive mode: %s", systemPrompt))
		cli.RunInteractiveMode(client, systemPrompt)
	},
}

func getAIYOUClient() *api.AIYOUClient {
	email := os.Getenv("AIYOU_EMAIL")
	password := os.Getenv("AIYOU_PASSWORD")
	if email == "" || password == "" {
		logger.Error("AIYOU_EMAIL and AIYOU_PASSWORD must be set in .env file")
		os.Exit(1)
	}
	if aiyouAssistantID == "" {
		logger.Error("Assistant ID must be provided using the -a flag")
		os.Exit(1)
	}
	client := api.NewAIYOUClient(aiyouAssistantID, debug)
	if err := client.Login(email, password); err != nil {
		logger.Error(fmt.Sprintf("Login failed: %v", err))
		os.Exit(1)
	}

	if showAssistantInfo {
		assistantInfo, err := client.GetAssistantInfo()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to fetch assistant info: %v", err))
		} else {
			logger.Info(fmt.Sprintf("Connected to assistant: %s (ID: %s)", assistantInfo.Name, assistantInfo.ID))
			logger.Info(fmt.Sprintf("Description: %s", assistantInfo.Description))
			logger.Info(fmt.Sprintf("Model: %s", assistantInfo.Model))
			logger.Info(fmt.Sprintf("Active Script: %v", assistantInfo.ActiveScript))
			logger.Info(fmt.Sprintf("Number of tools: %d", len(assistantInfo.Tools)))
			for i, tool := range assistantInfo.Tools {
				logger.Info(fmt.Sprintf("Tool %d: %s (%s)", i+1, tool.Title, tool.DisplayName))
			}
		}
	}

	return client
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
}
