// Package config provides centralized configuration management for Quill.
// Mirrors GenOffice's config patterns: layered config (defaults → file → env → runtime),
// JSON persistence, and typed accessors for all app settings.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// AIProviderConfig holds credentials and preferences for an AI provider.
type AIProviderConfig struct {
	Provider    string `json:"provider"`     // "openai", "anthropic", "gemini", "ollama"
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	BaseURL     string `json:"base_url,omitempty"`     // custom endpoint (ollama, azure)
	MaxTokens   int    `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// EditorConfig holds editor-wide preferences.
type EditorConfig struct {
	FontFamily    string `json:"font_family"`
	FontSize      int    `json:"font_size"`
	TabSize       int    `json:"tab_size"`
	WordWrap      bool   `json:"word_wrap"`
	LineNumbers   bool   `json:"line_numbers"`
	AutoSave      bool   `json:"auto_save"`
	AutoSaveDelay int    `json:"auto_save_delay_ms"` // milliseconds
	SpellCheck    bool   `json:"spell_check"`
}

// WindowConfig holds window geometry and state.
type WindowConfig struct {
	Width      int  `json:"width"`
	Height     int  `json:"height"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	Maximized  bool `json:"maximized"`
	Fullscreen bool `json:"fullscreen"`
}

// Config is the root configuration structure.
type Config struct {
	AI       AIProviderConfig `json:"ai"`
	Editor   EditorConfig     `json:"editor"`
	Window   WindowConfig     `json:"window"`
	Language string           `json:"language"`
	Theme    string           `json:"theme"` // "light", "dark", "system"
	DataDir  string           `json:"data_dir,omitempty"`
	LogLevel string           `json:"log_level"` // "debug", "info", "warn", "error"
}

// Manager manages loading, saving, and accessing configuration.
type Manager struct {
	mu       sync.RWMutex
	cfg      Config
	path     string
	dataDir  string
	onChange []func(Config)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		AI: AIProviderConfig{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-20250514",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		Editor: EditorConfig{
			FontFamily:    "system-ui, -apple-system, sans-serif",
			FontSize:      14,
			TabSize:       4,
			WordWrap:      true,
			LineNumbers:   false,
			AutoSave:      true,
			AutoSaveDelay: 2000,
			SpellCheck:    true,
		},
		Window: WindowConfig{
			Width:  1440,
			Height: 900,
		},
		Language: "en",
		Theme:    "system",
		LogLevel: "info",
	}
}

// New creates a new config Manager, loading from disk if available.
func New() *Manager {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".quill")
	os.MkdirAll(dataDir, 0755)

	m := &Manager{
		cfg:     DefaultConfig(),
		path:    filepath.Join(dataDir, "config.json"),
		dataDir: dataDir,
	}
	m.cfg.DataDir = dataDir
	m.load()

	// Override with environment variables
	m.applyEnv()
	return m
}

// Get returns a copy of the current configuration.
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// GetAI returns AI provider config.
func (m *Manager) GetAI() AIProviderConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.AI
}

// GetEditor returns editor config.
func (m *Manager) GetEditor() EditorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Editor
}

// DataDir returns the app data directory path.
func (m *Manager) DataDir() string {
	return m.dataDir
}

// Update applies a partial config update (JSON merge).
func (m *Manager) Update(partial map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Marshal current → map → merge → unmarshal back
	cur, _ := json.Marshal(m.cfg)
	var merged map[string]interface{}
	json.Unmarshal(cur, &merged)

	for k, v := range partial {
		merged[k] = v
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &m.cfg); err != nil {
		return err
	}

	// Persist and notify
	m.saveLocked()
	cfg := m.cfg
	for _, fn := range m.onChange {
		go fn(cfg)
	}
	return nil
}

// SetAIKey sets the API key for the configured provider.
func (m *Manager) SetAIKey(key string) error {
	return m.Update(map[string]interface{}{
		"ai": map[string]interface{}{
			"provider": m.cfg.AI.Provider,
			"api_key":  key,
			"model":    m.cfg.AI.Model,
		},
	})
}

// SetAIProvider switches the AI provider.
func (m *Manager) SetAIProvider(provider, model, apiKey string) error {
	return m.Update(map[string]interface{}{
		"ai": map[string]interface{}{
			"provider": provider,
			"model":    model,
			"api_key":  apiKey,
		},
	})
}

// OnChange registers a callback fired when config changes.
func (m *Manager) OnChange(fn func(Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &m.cfg)
}

func (m *Manager) saveLocked() {
	data, _ := json.MarshalIndent(m.cfg, "", "  ")
	os.WriteFile(m.path, data, 0644)
}

// Save persists the current config to disk.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.saveLocked()
	return nil
}

func (m *Manager) applyEnv() {
	if key := os.Getenv("OFFICE_AI_API_KEY"); key != "" {
		m.cfg.AI.APIKey = key
	}
	if model := os.Getenv("OFFICE_AI_MODEL"); model != "" {
		m.cfg.AI.Model = model
	}
	if provider := os.Getenv("OFFICE_AI_PROVIDER"); provider != "" {
		m.cfg.AI.Provider = provider
	}
	if baseURL := os.Getenv("OFFICE_AI_BASE_URL"); baseURL != "" {
		m.cfg.AI.BaseURL = baseURL
	}
	if theme := os.Getenv("OFFICE_AI_THEME"); theme != "" {
		m.cfg.Theme = theme
	}
	if lang := os.Getenv("OFFICE_AI_LANG"); lang != "" {
		m.cfg.Language = lang
	}
}
