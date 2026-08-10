// Package aiprovider implements multi-provider AI chat and streaming,
// mirroring GenOffice's ai-provider package.
package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// ProviderID identifies an AI provider.
type ProviderID string

const (
	ProviderOpenAI    ProviderID = "openai"
	ProviderAnthropic ProviderID = "anthropic"
	ProviderGemini    ProviderID = "gemini"
	ProviderGenSpark  ProviderID = "genspark"
)

// Settings configures AI provider selection and credentials.
type Settings struct {
	Provider ProviderID `json:"provider"`
	APIKey   string     `json:"api_key"`
	Model    string     `json:"model"`
	BaseURL  string     `json:"base_url,omitempty"`
}

// ChatMessage is a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatToolCall is a tool call from the model.
type ChatToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	InputJSON string `json:"input_json"`
}

// ChatRequest is a request to the AI provider.
type ChatRequest struct {
	Messages []ChatMessage            `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

// ChatResponse is a response from the AI provider.
type ChatResponse struct {
	Text      string         `json:"text"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"`
}

// StreamChunk is a single chunk from a streaming response.
type StreamChunk struct {
	Text  string `json:"text,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
}

// Service manages AI provider configuration and calls.
type Service struct {
	mu       sync.RWMutex
	settings Settings
	client   *http.Client
}

// New creates a new AI provider service with default settings.
func New() *Service {
	return &Service{
		settings: DefaultSettings(),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// DefaultSettings returns sensible defaults (OpenAI GPT-4o).
func DefaultSettings() Settings {
	return Settings{
		Provider: ProviderOpenAI,
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "gpt-4o",
	}
}

// GetSettings returns current AI settings.
func (s *Service) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// UpdateSettings changes the AI provider configuration.
func (s *Service) UpdateSettings(settings Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
}

// Chat sends a chat request and returns the full response.
func (s *Service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	switch settings.Provider {
	case ProviderOpenAI:
		return s.chatOpenAI(ctx, settings, req)
	case ProviderAnthropic:
		return s.chatAnthropic(ctx, settings, req)
	default:
		return s.chatOpenAI(ctx, settings, req) // OpenAI-compatible fallback
	}
}

// Stream sends a streaming chat request and calls onChunk for each piece.
func (s *Service) Stream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) error {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	_ = settings
	_ = onChunk
	// TODO: implement streaming for each provider
	return fmt.Errorf("streaming not yet implemented")
}

func (s *Service) chatOpenAI(ctx context.Context, settings Settings, req ChatRequest) (*ChatResponse, error) {
	baseURL := settings.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// Build OpenAI request body
	body := map[string]interface{}{
		"model":    settings.Model,
		"messages": req.Messages,
	}
	if len(req.Tools) > 0 {
		var tools []map[string]interface{}
		for _, t := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"type":     "function",
				"function": t,
			})
		}
		body["tools"] = tools
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	chatResp := &ChatResponse{}
	if len(result.Choices) > 0 {
		msg := result.Choices[0].Message
		chatResp.Text = msg.Content
		for _, tc := range msg.ToolCalls {
			chatResp.ToolCalls = append(chatResp.ToolCalls, ChatToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				InputJSON: tc.Function.Arguments,
			})
		}
	}
	return chatResp, nil
}

func (s *Service) chatAnthropic(ctx context.Context, settings Settings, req ChatRequest) (*ChatResponse, error) {
	baseURL := settings.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	// Extract system message
	var systemPrompt string
	var messages []map[string]string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt += m.Content + "\n"
		} else {
			messages = append(messages, map[string]string{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	body := map[string]interface{}{
		"model":      settings.Model,
		"max_tokens": 4096,
		"messages":   messages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", settings.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	chatResp := &ChatResponse{}
	for _, c := range result.Content {
		if c.Type == "text" {
			chatResp.Text += c.Text
		}
	}
	return chatResp, nil
}
