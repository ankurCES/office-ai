// Package agentcore implements the AI agent loop, mirroring GenOffice's agent-core package.
// Provides AgentLoop (turn-based tool-calling loop), AgentSkill (domain-specific tools),
// and skill composition for multi-domain agents.
package agentcore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ankurCES/office-ai/pkg/aiprovider"
)

// ToolDef describes a tool exposed to the AI model (JSON Schema).
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolCall represents a tool invocation from the model.
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult is the outcome of executing a tool.
type ToolResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error,omitempty"`
}

// Message represents a conversation message.
type Message struct {
	Role      string     `json:"role"` // "user", "assistant", "tool"
	Text      string     `json:"text,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// Image represents an inline image for vision-capable models.
type Image struct {
	Base64 string `json:"base64"`
	MIME   string `json:"mime"`
}

// Skill packages one capability domain for the agent loop.
type Skill struct {
	ID           string
	SystemPrompt string
	Tools        []ToolDef
	BuildContext func() string
	ExecuteTool  func(ctx context.Context, call ToolCall) ToolResult
}

// ComposeSkills merges several skills into one (tool names must be globally unique).
func ComposeSkills(id, intro string, skills ...*Skill) *Skill {
	owner := make(map[string]*Skill)
	var allTools []ToolDef
	for _, s := range skills {
		for _, t := range s.Tools {
			if _, exists := owner[t.Name]; exists {
				panic(fmt.Sprintf("duplicate tool name: %s", t.Name))
			}
			owner[t.Name] = s
			allTools = append(allTools, t)
		}
	}
	prompts := []string{intro}
	for _, s := range skills {
		if s.SystemPrompt != "" {
			prompts = append(prompts, s.SystemPrompt)
		}
	}
	return &Skill{
		ID:           id,
		SystemPrompt: joinNonEmpty(prompts, "\n\n"),
		Tools:        allTools,
		BuildContext: func() string {
			var parts []string
			for _, s := range skills {
				if s.BuildContext != nil {
					if ctx := s.BuildContext(); ctx != "" {
						parts = append(parts, ctx)
					}
				}
			}
			return joinNonEmpty(parts, "\n\n")
		},
		ExecuteTool: func(ctx context.Context, call ToolCall) ToolResult {
			s, ok := owner[call.Name]
			if !ok {
				return ToolResult{ID: call.ID, Name: call.Name, Output: "Unknown tool: " + call.Name, IsError: true}
			}
			return s.ExecuteTool(ctx, call)
		},
	}
}

// RunResult is the outcome of an agent run.
type RunResult struct {
	Text      string `json:"text"`
	Cancelled bool   `json:"cancelled"`
	TurnLimit bool   `json:"turn_limit"`
	Truncated bool   `json:"truncated,omitempty"`
}

// Events are callbacks fired during the agent loop.
type Events struct {
	OnText          func(text string)
	OnToolStart     func(call ToolCall)
	OnToolExecuted  func(call ToolCall, result ToolResult)
	OnTurnEnd       func()
	OnDone          func(result RunResult)
	OnError         func(err string)
}

// LoopOptions configures the agent loop.
type LoopOptions struct {
	Skill     *Skill
	Events    Events
	MaxTurns  int // default 8
	MaxHistory int // default 40
}

// Loop is the main agent loop that orchestrates tool-calling turns.
type Loop struct {
	provider *aiprovider.Service
	mu       sync.Mutex
	history  []Message
}

// New creates a new agent Loop backed by the given AI provider.
func New(provider *aiprovider.Service) *Loop {
	return &Loop{provider: provider}
}

// Run executes a user message through the agent loop.
func (l *Loop) Run(ctx context.Context, userText string, opts LoopOptions) RunResult {
	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 8
	}

	l.mu.Lock()
	l.history = append(l.history, Message{Role: "user", Text: userText})
	l.mu.Unlock()

	for turn := 0; turn < maxTurns; turn++ {
		select {
		case <-ctx.Done():
			return RunResult{Cancelled: true}
		default:
		}

		// Build context if skill provides it
		contextText := ""
		if opts.Skill != nil && opts.Skill.BuildContext != nil {
			contextText = opts.Skill.BuildContext()
		}

		// Call the AI provider
		l.mu.Lock()
		history := make([]Message, len(l.history))
		copy(history, l.history)
		l.mu.Unlock()

		resp, err := l.callModel(ctx, opts.Skill, history, contextText)
		if err != nil {
			if opts.Events.OnError != nil {
				opts.Events.OnError(err.Error())
			}
			return RunResult{Text: "", Cancelled: ctx.Err() != nil}
		}

		if opts.Events.OnText != nil && resp.Text != "" {
			opts.Events.OnText(resp.Text)
		}

		// No tool calls → done
		if len(resp.ToolCalls) == 0 {
			result := RunResult{Text: resp.Text}
			if opts.Events.OnDone != nil {
				opts.Events.OnDone(result)
			}
			return result
		}

		// Execute tools
		l.mu.Lock()
		l.history = append(l.history, Message{Role: "assistant", Text: resp.Text, ToolCalls: resp.ToolCalls})
		l.mu.Unlock()

		for _, tc := range resp.ToolCalls {
			if opts.Events.OnToolStart != nil {
				opts.Events.OnToolStart(tc)
			}
			result := opts.Skill.ExecuteTool(ctx, tc)
			if opts.Events.OnToolExecuted != nil {
				opts.Events.OnToolExecuted(tc, result)
			}
			l.mu.Lock()
			l.history = append(l.history, Message{Role: "tool", ToolResult: &result})
			l.mu.Unlock()
		}

		if opts.Events.OnTurnEnd != nil {
			opts.Events.OnTurnEnd()
		}
	}

	return RunResult{TurnLimit: true}
}

// ClearHistory resets the conversation history.
func (l *Loop) ClearHistory() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.history = nil
}

// GetHistory returns a copy of the conversation history.
func (l *Loop) GetHistory() []Message {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]Message, len(l.history))
	copy(result, l.history)
	return result
}

type modelResponse struct {
	Text      string
	ToolCalls []ToolCall
}

func (l *Loop) callModel(ctx context.Context, skill *Skill, history []Message, contextText string) (*modelResponse, error) {
	// Build messages for the provider
	var messages []aiprovider.ChatMessage
	if skill != nil {
		messages = append(messages, aiprovider.ChatMessage{Role: "system", Content: skill.SystemPrompt})
	}
	for _, m := range history {
		messages = append(messages, aiprovider.ChatMessage{Role: m.Role, Content: m.Text})
	}
	if contextText != "" {
		messages = append(messages, aiprovider.ChatMessage{Role: "system", Content: "[Context]\n" + contextText})
	}

	// Convert tools
	var tools []map[string]interface{}
	if skill != nil {
		for _, t := range skill.Tools {
			toolJSON := map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			}
			tools = append(tools, toolJSON)
		}
	}

	resp, err := l.provider.Chat(ctx, aiprovider.ChatRequest{
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		return nil, err
	}

	result := &modelResponse{Text: resp.Text}
	for _, tc := range resp.ToolCalls {
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(tc.InputJSON), &input); err != nil {
			input = map[string]interface{}{"_raw": tc.InputJSON}
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: input,
		})
	}
	return result, nil
}

func joinNonEmpty(parts []string, sep string) string {
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	result := filtered[0]
	for _, p := range filtered[1:] {
		result += sep + p
	}
	return result
}
