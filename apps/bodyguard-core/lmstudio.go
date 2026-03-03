package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// LMStudioClient handles communication with LM Studio API
type LMStudioClient struct {
	apiURL string
	token  string
	client *http.Client
}

// NewLMStudioClient creates a new LM Studio client
func NewLMStudioClient() *LMStudioClient {
	apiURL := os.Getenv("BG_LM_STUDIO_URL")
	if apiURL == "" {
		apiURL = "http://localhost:1234/v1"
	}

	token := os.Getenv("BG_LM_STUDIO_TOKEN")

	return &LMStudioClient{
		apiURL: apiURL,
		token:  token,
		client: &http.Client{},
	}
}

// Complete sends a prompt to LM Studio and returns the response
func (lm *LMStudioClient) Complete(prompt string) (string, error) {
	reqBody := LMRequest{
		Model: "local-model",
		Messages: []LMMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", lm.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authorization header if token is provided
	if lm.token != "" {
		req.Header.Set("Authorization", "Bearer "+lm.token)
	}

	resp, err := lm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call LM Studio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LM Studio returned status %d: %s", resp.StatusCode, string(body))
	}

	var lmResp LMResponse
	if err := json.NewDecoder(resp.Body).Decode(&lmResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(lmResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LM Studio")
	}

	return lmResp.Choices[0].Message.Content, nil
}

// IsAvailable checks if LM Studio is accessible
func (lm *LMStudioClient) IsAvailable() bool {
	req, err := http.NewRequest("GET", lm.apiURL+"/models", nil)
	if err != nil {
		return false
	}

	if lm.token != "" {
		req.Header.Set("Authorization", "Bearer "+lm.token)
	}

	resp, err := lm.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
