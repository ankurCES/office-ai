package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	svc := New()
	if svc == nil {
		t.Fatal("New() returned nil")
	}
	if svc.settings.Provider != ProviderOpenAI {
		t.Errorf("default provider = %q, want %q", svc.settings.Provider, ProviderOpenAI)
	}
	if svc.settings.Model != "gpt-4o" {
		t.Errorf("default model = %q, want %q", svc.settings.Model, "gpt-4o")
	}
}

func TestUpdateSettings(t *testing.T) {
	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    "claude-sonnet-4-20250514",
	})
	got := svc.GetSettings()
	if got.Provider != ProviderAnthropic {
		t.Errorf("provider = %q, want %q", got.Provider, ProviderAnthropic)
	}
	if got.APIKey != "test-key" {
		t.Errorf("api_key = %q, want %q", got.APIKey, "test-key")
	}
}

func TestListModels(t *testing.T) {
	svc := New()
	svc.UpdateSettings(Settings{Provider: ProviderOpenAI})
	models, err := svc.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("ListModels returned empty list")
	}
	found := false
	for _, m := range models {
		if m == "gpt-4o" {
			found = true
		}
	}
	if !found {
		t.Error("gpt-4o not in OpenAI model list")
	}
}

func TestChatOpenAI(t *testing.T) {
	// Mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", 401)
			return
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{
					"content": "Hello, world!",
				}},
			},
			"model": "gpt-4o",
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-4o",
		BaseURL:  server.URL,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Text != "Hello, world!" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello, world!")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 15 {
		t.Errorf("usage.total_tokens = %v, want 15", resp.Usage)
	}
}

func TestChatAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			http.Error(w, "bad version", 400)
			return
		}
		resp := map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "Claude says hi"},
			},
			"model": "claude-sonnet-4-20250514",
			"usage": map[string]interface{}{
				"input_tokens":  8,
				"output_tokens": 3,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    "claude-sonnet-4-20250514",
		BaseURL:  server.URL,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Text != "Claude says hi" {
		t.Errorf("text = %q, want %q", resp.Text, "Claude says hi")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 11 {
		t.Errorf("usage.total_tokens = %v, want 11", resp.Usage)
	}
}

func TestChatOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"message": map[string]interface{}{
				"content": "Ollama response",
			},
			"model": "llama3",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOllama,
		Model:    "llama3",
		BaseURL:  server.URL,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Text != "Ollama response" {
		t.Errorf("text = %q, want %q", resp.Text, "Ollama response")
	}
}

func TestStreamOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{"Hello", ", ", "world", "!"}
		for _, chunk := range chunks {
			data := fmt.Sprintf(`{"choices":[{"delta":{"content":"%s"}}]}`, chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-4o",
		BaseURL:  server.URL,
	})

	var collected strings.Builder
	var doneReceived bool
	err := svc.Stream(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk StreamChunk) {
		if chunk.Done {
			doneReceived = true
		}
		collected.WriteString(chunk.Text)
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if collected.String() != "Hello, world!" {
		t.Errorf("streamed text = %q, want %q", collected.String(), "Hello, world!")
	}
	if !doneReceived {
		t.Error("never received done=true")
	}
}

func TestStreamAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		events := []string{
			`{"type":"content_block_start"}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi "}}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"there"}}`,
			`{"type":"message_stop"}`,
		}
		for _, ev := range events {
			fmt.Fprintf(w, "data: %s\n\n", ev)
			flusher.Flush()
		}
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderAnthropic,
		APIKey:   "test-key",
		Model:    "claude-sonnet-4-20250514",
		BaseURL:  server.URL,
	})

	var collected strings.Builder
	err := svc.Stream(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk StreamChunk) {
		collected.WriteString(chunk.Text)
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if collected.String() != "Hi there" {
		t.Errorf("streamed text = %q, want %q", collected.String(), "Hi there")
	}
}

func TestStreamOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, _ := w.(http.Flusher)
		chunks := []struct {
			msg  string
			done bool
		}{
			{"Hello", false},
			{" from", false},
			{" Ollama", false},
			{"", true},
		}
		for _, c := range chunks {
			data := map[string]interface{}{
				"message": map[string]string{"content": c.msg},
				"done":    c.done,
			}
			json.NewEncoder(w).Encode(data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOllama,
		Model:    "llama3",
		BaseURL:  server.URL,
	})

	var collected strings.Builder
	err := svc.Stream(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}, func(chunk StreamChunk) {
		collected.WriteString(chunk.Text)
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if collected.String() != "Hello from Ollama" {
		t.Errorf("streamed text = %q, want %q", collected.String(), "Hello from Ollama")
	}
}

func TestChatOpenAIToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{
					"content": "",
					"tool_calls": []map[string]interface{}{
						{
							"id":   "call_123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "get_weather",
								"arguments": `{"city":"NYC"}`,
							},
						},
					},
				}},
			},
			"model": "gpt-4o",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "weather?"}},
		Tools: []map[string]interface{}{
			{"name": "get_weather", "parameters": map[string]interface{}{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("tool name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
}

func TestChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"rate limited"}}`, 429)
	}))
	defer server.Close()

	svc := New()
	svc.UpdateSettings(Settings{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		BaseURL:  server.URL,
	})

	_, err := svc.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want to contain '429'", err.Error())
	}
}
