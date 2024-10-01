package api

import (
	"fmt"
	"time"

	"github.com/chrlesur/aiyou.cli/pkg/logger"
)

type Run struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Response string `json:"response"`
}

func (c *AIYOUClient) createRun(threadID string) (string, error) {
	logger.Debug(fmt.Sprintf("Creating run for thread %s", threadID))

	runData := map[string]string{
		"assistantId": c.AssistantID,
	}

	var runResp Run
	err := c.apiCaller.Call(fmt.Sprintf("/v1/threads/%s/runs", threadID), "POST", runData, &runResp)
	if err != nil {
		return "", fmt.Errorf("error creating run: %w", err)
	}

	logger.Debug(fmt.Sprintf("Run created with ID: %s", runResp.ID))
	return runResp.ID, nil
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
