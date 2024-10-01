package api

import (
	"fmt"

	"github.com/chrlesur/aiyou.cli/pkg/logger"
)

func (c *AIYOUClient) createThread() (string, error) {
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