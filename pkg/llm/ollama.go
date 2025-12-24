package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// OllamaProvider connects to Ollama via its OpenAI-compatible API.
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// OllamaConfig holds configuration for the Ollama provider.
type OllamaConfig struct {
	BaseURL string // e.g., "http://localhost:11434/v1"
	Model   string // e.g., "llama3.2"
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg OllamaConfig) *OllamaProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	// Remove trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &OllamaProvider{
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}
}

// chatRequest represents the OpenAI chat completion request format.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the OpenAI chat completion response format.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// streamChunk represents a streaming response chunk.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Complete generates a non-streaming response.
func (p *OllamaProvider) Complete(ctx context.Context, messages []Message) (string, error) {
	slog.Debug("ollama complete request", "model", p.model, "message_count", len(messages))

	reqBody := chatRequest{
		Model:    p.model,
		Messages: convertMessages(messages),
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		slog.Error("ollama request failed", "err", err)
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("ollama API error", "status", resp.StatusCode, "body", string(respBody))
		return "", fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("ollama error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Stream generates a streaming response.
func (p *OllamaProvider) Stream(ctx context.Context, messages []Message) (<-chan StreamEvent, error) {
	slog.Debug("ollama stream request", "model", p.model, "message_count", len(messages))

	reqBody := chatRequest{
		Model:    p.model,
		Messages: convertMessages(messages),
		Stream:   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// SSE format: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Check for end of stream
			if data == "[DONE]" {
				ch <- StreamEvent{Done: true}
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- StreamEvent{Error: fmt.Errorf("decode chunk: %w", err), Done: true}
				return
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					ch <- StreamEvent{Content: content}
				}

				if chunk.Choices[0].FinishReason == "stop" {
					ch <- StreamEvent{Done: true}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamEvent{Error: fmt.Errorf("read stream: %w", err), Done: true}
			return
		}

		// Stream ended without explicit done signal
		ch <- StreamEvent{Done: true}
	}()

	return ch, nil
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// convertMessages converts llm.Message to the chat API format.
func convertMessages(messages []Message) []chatMessage {
	result := make([]chatMessage, len(messages))
	for i, m := range messages {
		result[i] = chatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}
