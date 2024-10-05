package api

import (
	"fmt"
	"io"
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

type Run struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Response string `json:"response"`
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

func (c *AIYOUClient) CreateThread() (string, error) {
	logger.Debug("Creating new AI.YOU thread")

	var threadResp struct {
		ID string `json:"id"`
	}

	err := c.apiCaller.Call("/v1/threads", "POST", map[string]string{}, &threadResp)
	if err != nil {
		return "", fmt.Errorf("error creating thread: %w", err)
	}

	if threadResp.ID == "" {
		return "", fmt.Errorf("thread ID is empty in response")
	}

	logger.Debug(fmt.Sprintf("Thread created with ID: %s", threadResp.ID))
	return threadResp.ID, nil
}

func (c *AIYOUClient) ChatInThread(threadID, input string) (string, error) {
	logger.Debug(fmt.Sprintf("Chatting in thread %s", threadID))

	err := c.addMessage(threadID, input)
	if err != nil {
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

func (c *AIYOUClient) addMessage(threadID, content string) error {
	logger.Debug(fmt.Sprintf("Adding message to thread %s", threadID))

	messageData := map[string]string{
		"role":    "user",
		"content": content,
	}

	var response interface{}
	err := c.apiCaller.Call(fmt.Sprintf("/v1/threads/%s/messages", threadID), "POST", messageData, &response)
	if err != nil {
		return fmt.Errorf("error adding message: %w", err)
	}

	logger.Debug("Message added successfully")
	return nil
}

func (c *AIYOUClient) createRun(threadID string) (string, error) {
	logger.Debug(fmt.Sprintf("Creating run for thread %s", threadID))

	runData := map[string]string{
		"assistantId": c.AssistantID,
	}

	var runResp struct {
		ID string `json:"id"`
	}
	err := c.apiCaller.Call(fmt.Sprintf("/v1/threads/%s/runs", threadID), "POST", runData, &runResp)
	if err != nil {
		return "", fmt.Errorf("error creating run: %w", err)
	}

	logger.Debug(fmt.Sprintf("Run created with ID: %s", runResp.ID))
	return runResp.ID, nil
}

func (c *AIYOUClient) waitForCompletion(threadID, runID string) (*Run, error) {
	maxAttempts := 30
	delayBetweenAttempts := 2 * time.Second

	for i := 0; i < maxAttempts; i++ {
		logger.Debug(fmt.Sprintf("Attempt %d to retrieve run status", i+1))
		run, err := c.retrieveRun(threadID, runID)
		if err != nil {
			return nil, err
		}

		switch run.Status {
		case "completed":
			logger.Debug("Run completed successfully")
			return run, nil
		case "failed", "cancelled":
			return nil, fmt.Errorf("run failed with status: %s", run.Status)
		default:
			logger.Debug(fmt.Sprintf("Waiting for run completion. Pausing for %v", delayBetweenAttempts))
			time.Sleep(delayBetweenAttempts)
		}
	}

	return nil, fmt.Errorf("timeout waiting for run completion")
}

func (c *AIYOUClient) retrieveRun(threadID, runID string) (*Run, error) {
	logger.Debug(fmt.Sprintf("Retrieving run %s for thread %s", runID, threadID))

	var runStatus Run
	err := c.apiCaller.Call(fmt.Sprintf("/v1/threads/%s/runs/%s", threadID, runID), "POST", map[string]string{}, &runStatus)
	if err != nil {
		return nil, fmt.Errorf("error retrieving run: %w", err)
	}

	logger.Debug(fmt.Sprintf("Run status retrieved: %v", runStatus))
	return &runStatus, nil
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
	err := c.apiCaller.Call(endpoint, "POST", nil, &assistantInfo)
	if err != nil {
		logger.Error(fmt.Sprintf("Error fetching assistant info: %v", err))
		return nil, fmt.Errorf("error fetching assistant info: %w", err)
	}
	logger.Debug(fmt.Sprintf("Successfully retrieved info for assistant: %s", assistantInfo.Name))
	return &assistantInfo, nil
}

func (c *AIYOUClient) Chat(input io.Reader, additionalInstruction string) (string, error) {
	logger.Debug("Starting chat with AI.YOU")

	inputBytes, err := ioutil.ReadAll(input)
	if err != nil {
		logger.Error(fmt.Sprintf("Error reading input: %v", err))
		return "", fmt.Errorf("error reading input: %w", err)
	}
	inputString := strings.TrimSpace(string(inputBytes))

	threadID, err := c.CreateThread()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create thread: %v", err))
		return "", fmt.Errorf("failed to create thread: %w", err)
	}

	if additionalInstruction != "" {
		inputString = fmt.Sprintf("%s\n\nAdditional instruction: %s", inputString, additionalInstruction)
	}

	response, err := c.ChatInThread(threadID, inputString)
	if err != nil {
		logger.Error(fmt.Sprintf("Error during chat: %v", err))
		return "", fmt.Errorf("error during chat: %w", err)
	}

	logger.Debug(fmt.Sprintf("AI.YOU response: %s", response))
	return response, nil
}
