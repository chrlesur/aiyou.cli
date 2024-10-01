package api

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/chrlesur/aiyou.cli/pkg/logger"
)

const AIYOUAPIURL = "https://ai.dragonflygroup.fr/api"

type AIYOUClient struct {
	apiCaller   APICaller
	AssistantID string
	Debug       bool
	Timeout     time.Duration
}

type Tool struct {
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Function    struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"input_schema"`
	} `json:"function"`
}

type AssistantInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Image        string `json:"image"`
	AssistantID  string `json:"assistantId"`
	Model        string `json:"model"`
	ActiveScript bool   `json:"activeScript"`
	Tools        []Tool `json:"tools"`
	Voice        string `json:"voice"`
}

func NewAIYOUClient(assistantID string, debug bool) *AIYOUClient {
	logger.Debug(fmt.Sprintf("Creating new AIYOUClient with assistant ID: %s", assistantID))
	return &AIYOUClient{
		apiCaller:   NewHTTPAPICaller(AIYOUAPIURL),
		AssistantID: assistantID,
		Debug:       debug,
		Timeout:     120 * time.Second,
	}
}

func (c *AIYOUClient) Login(email, password string) error {
	logger.Info("Attempting to log in to AI.YOU")
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}

	var loginResp struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}

	err := c.apiCaller.Call("/login", "POST", loginData, &loginResp)
	if err != nil {
		logger.Error(fmt.Sprintf("Login failed: %v", err))
		return fmt.Errorf("login failed: %w", err)
	}

	c.apiCaller.SetToken(loginResp.Token)
	logger.Info("Successfully logged in to AI.YOU")
	return nil
}

func (c *AIYOUClient) Chat(input, additionalInstruction string) (string, error) {
	logger.Debug("Starting chat with AI.YOU")

	threadID, err := c.createThread()
	if err != nil {
		return "", fmt.Errorf("failed to create thread: %w", err)
	}

	prompt := input
	if additionalInstruction != "" {
		prompt = fmt.Sprintf("%s\n\nAdditional instruction: %s", input, additionalInstruction)
	}

	if err := c.addMessage(threadID, prompt); err != nil {
		return "", fmt.Errorf("failed to add message: %w", err)
	}

	runID, err := c.createRun(threadID)
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	completedRun, err := c.waitForCompletion(threadID, runID)
	if err != nil {
		return "", fmt.Errorf("run failed: %w", err)
	}

	return completedRun.Response, nil
}

func (c *AIYOUClient) LoadInstructionFromFile(filename string) (string, error) {
	logger.Debug(fmt.Sprintf("Loading instruction from file: %s", filename))
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Error(fmt.Sprintf("Error reading instruction file: %v", err))
		return "", fmt.Errorf("error reading instruction file: %w", err)
	}
	instruction := strings.TrimSpace(string(content))
	logger.Debug(fmt.Sprintf("Loaded instruction from file: %s", instruction))
	return instruction, nil
}

func (c *AIYOUClient) GetAssistantInfo() (*AssistantInfo, error) {
	logger.Debug(fmt.Sprintf("Fetching info for assistant ID: %s", c.AssistantID))
	endpoint := fmt.Sprintf("/v1/assistants/%s", c.AssistantID)
	var assistantInfo AssistantInfo
	err := c.apiCaller.Call(endpoint, "GET", nil, &assistantInfo)
	if err != nil {
		logger.Error(fmt.Sprintf("Error fetching assistant info: %v", err))
		return nil, fmt.Errorf("error fetching assistant info: %w", err)
	}
	logger.Debug(fmt.Sprintf("Successfully retrieved info for assistant: %s", assistantInfo.Name))
	return &assistantInfo, nil
}
