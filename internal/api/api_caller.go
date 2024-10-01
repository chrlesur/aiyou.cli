package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/chrlesur/aiyou.cli/pkg/logger"
)

type APICaller interface {
	Call(endpoint, method string, data interface{}, response interface{}) error
	SetToken(token string)
}

type HTTPAPICaller struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewHTTPAPICaller(baseURL string) *HTTPAPICaller {
	logger.Debug(fmt.Sprintf("Initializing HTTPAPICaller with base URL: %s", baseURL))
	return &HTTPAPICaller{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *HTTPAPICaller) SetToken(token string) {
	logger.Debug("Setting API token")
	c.token = token
}

func (c *HTTPAPICaller) Call(endpoint, method string, data interface{}, response interface{}) error {
	logger.Debug(fmt.Sprintf("Making API call: %s %s", method, endpoint))

	var req *http.Request
	var err error

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			logger.Error(fmt.Sprintf("Error marshaling request data: %v", err))
			return fmt.Errorf("error marshaling request data: %w", err)
		}
		req, err = http.NewRequest(method, c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, c.baseURL+endpoint, nil)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("Error creating request: %v", err))
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	logger.Debug(fmt.Sprintf("Sending request to: %s", req.URL.String()))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("Error sending request: %v", err))
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Error reading response body: %v", err))
		return fmt.Errorf("error reading response body: %w", err)
	}

	logger.Debug(fmt.Sprintf("Received response with status code: %d", resp.StatusCode))
	logger.Debug(fmt.Sprintf("Response body : %s", string(body)))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		logger.Error(fmt.Sprintf("API error: status code %d, body: %s", resp.StatusCode, string(body)))
		return fmt.Errorf("API error: status code %d", resp.StatusCode)
	}

	if err := json.Unmarshal(body, response); err != nil {
		logger.Error(fmt.Sprintf("Error unmarshaling response: %v", err))
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	logger.Debug("API call successful")
	return nil
}
