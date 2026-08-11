// Package aiprovider implements multi-provider AI chat and streaming,
// mirroring GenOffice's ai-provider package. Supports OpenAI, Anthropic,
// Ollama, Gemini, and any OpenAI-compatible endpoint.
package aiprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ProviderID identifies an AI provider.
type ProviderID string

const (
	ProviderOpenAI    ProviderID = "openai"
	ProviderAnthropic ProviderID = "anthropic"
	ProviderOllama    ProviderID = "ollama"
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
	Messages    []ChatMessage            `json:"messages"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature *float64                 `json:"temperature,omitempty"`
}

// ChatResponse is a response from the AI provider.
type ChatResponse struct {
	Text      string         `json:"text"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"`
	Model     string         `json:"model,omitempty"`
	Usage     *UsageInfo     `json:"usage,omitempty"`
}

// UsageInfo tracks token usage.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
	// streamClient has no timeout — SSE connections stay open
	streamClient *http.Client
}

// New creates a new AI provider service with default settings.
func New() *Service {
	return &Service{
		settings: DefaultSettings(),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		streamClient: &http.Client{
			Timeout: 0, // no timeout for streaming
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

// ListModels returns available models for the current provider.
func (s *Service) ListModels(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	switch settings.Provider {
	case ProviderOllama:
		return s.listOllamaModels(ctx, settings)
	case ProviderOpenAI:
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo", "o1", "o1-mini"}, nil
	case ProviderAnthropic:
		return []string{"claude-sonnet-4-20250514", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"}, nil
	default:
		return []string{settings.Model}, nil
	}
}

// Chat sends a chat request and returns the full response.
func (s *Service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	switch settings.Provider {
	case ProviderOpenAI, ProviderGemini, ProviderGenSpark:
		return s.chatOpenAI(ctx, settings, req)
	case ProviderAnthropic:
		return s.chatAnthropic(ctx, settings, req)
	case ProviderOllama:
		return s.chatOllama(ctx, settings, req)
	default:
		return s.chatOpenAI(ctx, settings, req)
	}
}

// Stream sends a streaming chat request and calls onChunk for each piece.
// The callback is invoked on a goroutine; callers should handle synchronization.
func (s *Service) Stream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) error {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	switch settings.Provider {
	case ProviderOpenAI, ProviderGemini, ProviderGenSpark:
		return s.streamOpenAI(ctx, settings, req, onChunk)
	case ProviderAnthropic:
		return s.streamAnthropic(ctx, settings, req, onChunk)
	case ProviderOllama:
		return s.streamOllama(ctx, settings, req, onChunk)
	default:
		return s.streamOpenAI(ctx, settings, req, onChunk)
	}
}


// ─── OpenAI / OpenAI-compatible ─────────────────────────────────────

func (s *Service) openAIBaseURL(settings Settings) string {
	if settings.BaseURL != "" {
		return strings.TrimRight(settings.BaseURL, "/")
	}
	return "https://api.openai.com/v1"
}

func (s *Service) buildOpenAIBody(settings Settings, req ChatRequest, stream bool) ([]byte, error) {
	body := map[string]interface{}{
		"model":    settings.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
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
	return json.Marshal(body)
}

func (s *Service) chatOpenAI(ctx context.Context, settings Settings, req ChatRequest) (*ChatResponse, error) {
	baseURL := s.openAIBaseURL(settings)

	jsonBody, err := s.buildOpenAIBody(settings, req, false)
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
		Model string `json:"model"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	chatResp := &ChatResponse{Model: result.Model}
	if result.Usage != nil {
		chatResp.Usage = &UsageInfo{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}
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

// streamOpenAI implements SSE streaming for OpenAI-compatible APIs.
func (s *Service) streamOpenAI(ctx context.Context, settings Settings, req ChatRequest, onChunk func(StreamChunk)) error {
	baseURL := s.openAIBaseURL(settings)

	jsonBody, err := s.buildOpenAIBody(settings, req, true)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			onChunk(StreamChunk{Done: true})
			return nil
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(StreamChunk{Text: chunk.Choices[0].Delta.Content})
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	onChunk(StreamChunk{Done: true})
	return nil
}


// ─── Anthropic (Claude) ─────────────────────────────────────────────

func (s *Service) anthropicBaseURL(settings Settings) string {
	if settings.BaseURL != "" {
		return strings.TrimRight(settings.BaseURL, "/")
	}
	return "https://api.anthropic.com/v1"
}

func (s *Service) buildAnthropicBody(settings Settings, req ChatRequest, stream bool) ([]byte, error) {
	var systemPrompt string
	var messages []map[string]string
	for _, m := range req.Messages {
		if m.Role == "system" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += m.Content
		} else {
			messages = append(messages, map[string]string{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	body := map[string]interface{}{
		"model":      settings.Model,
		"max_tokens": maxTokens,
		"messages":   messages,
		"stream":     stream,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}

	return json.Marshal(body)
}

func (s *Service) setAnthropicHeaders(httpReq *http.Request, settings Settings) {
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", settings.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
}

func (s *Service) chatAnthropic(ctx context.Context, settings Settings, req ChatRequest) (*ChatResponse, error) {
	baseURL := s.anthropicBaseURL(settings)

	jsonBody, err := s.buildAnthropicBody(settings, req, false)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	s.setAnthropicHeaders(httpReq, settings)

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
		Model string `json:"model"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	chatResp := &ChatResponse{Model: result.Model}
	if result.Usage != nil {
		chatResp.Usage = &UsageInfo{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		}
	}
	for _, c := range result.Content {
		if c.Type == "text" {
			chatResp.Text += c.Text
		}
	}
	return chatResp, nil
}

// streamAnthropic implements SSE streaming for Anthropic's Messages API.
func (s *Service) streamAnthropic(ctx context.Context, settings Settings, req ChatRequest, onChunk func(StreamChunk)) error {
	baseURL := s.anthropicBaseURL(settings)

	jsonBody, err := s.buildAnthropicBody(settings, req, true)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	s.setAnthropicHeaders(httpReq, settings)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta *struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				onChunk(StreamChunk{Text: event.Delta.Text})
			}
		case "message_stop":
			onChunk(StreamChunk{Done: true})
			return nil
		case "error":
			onChunk(StreamChunk{Error: data, Done: true})
			return fmt.Errorf("anthropic stream error: %s", data)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	onChunk(StreamChunk{Done: true})
	return nil
}


// ─── Ollama (local models) ──────────────────────────────────────────

func (s *Service) ollamaBaseURL(settings Settings) string {
	if settings.BaseURL != "" {
		return strings.TrimRight(settings.BaseURL, "/")
	}
	return "http://localhost:11434"
}

func (s *Service) chatOllama(ctx context.Context, settings Settings, req ChatRequest) (*ChatResponse, error) {
	baseURL := s.ollamaBaseURL(settings)

	body := map[string]interface{}{
		"model":    settings.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		body["options"] = map[string]interface{}{"temperature": *req.Temperature}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama API call failed (is ollama running?): %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &ChatResponse{
		Text:  result.Message.Content,
		Model: result.Model,
	}, nil
}

// streamOllama implements newline-delimited JSON streaming for Ollama.
func (s *Service) streamOllama(ctx context.Context, settings Settings, req ChatRequest, onChunk func(StreamChunk)) error {
	baseURL := s.ollamaBaseURL(settings)

	body := map[string]interface{}{
		"model":    settings.Model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature != nil {
		body["options"] = map[string]interface{}{"temperature": *req.Temperature}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Ollama stream failed (is ollama running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if chunk.Message.Content != "" {
			onChunk(StreamChunk{Text: chunk.Message.Content})
		}
		if chunk.Done {
			onChunk(StreamChunk{Done: true})
			return nil
		}
	}

	onChunk(StreamChunk{Done: true})
	return nil
}

func (s *Service) listOllamaModels(ctx context.Context, settings Settings) ([]string, error) {
	baseURL := s.ollamaBaseURL(settings)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}

	names := make([]string, len(result.Models))
	for i, m := range result.Models {
		names[i] = m.Name
	}
	return names, nil
}
