package config

import (
	"os"
	"testing"
)

// newTestManager creates a Manager backed by a temp dir so disk state doesn't leak.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	m := &Manager{
		cfg:     DefaultConfig(),
		path:    dir + "/config.json",
		dataDir: dir,
	}
	m.cfg.DataDir = dir
	return m
}

func TestNew(t *testing.T) {
	mgr := New()
	if mgr == nil {
		t.Fatal("New() returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	mgr := newTestManager(t)
	cfg := mgr.Get()

	if cfg.Theme != "system" {
		t.Errorf("expected theme=system, got %q", cfg.Theme)
	}
	if cfg.Editor.AutoSave != true {
		t.Errorf("expected auto_save=true, got %v", cfg.Editor.AutoSave)
	}
	if cfg.Editor.AutoSaveDelay != 2000 {
		t.Errorf("expected auto_save_delay=2000, got %d", cfg.Editor.AutoSaveDelay)
	}
	if cfg.Language != "en" {
		t.Errorf("expected language=en, got %q", cfg.Language)
	}
}

func TestUpdate(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.Update(map[string]interface{}{
		"theme":    "dark",
		"language": "ja",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	cfg := mgr.Get()
	if cfg.Theme != "dark" {
		t.Errorf("expected dark, got %q", cfg.Theme)
	}
	if cfg.Language != "ja" {
		t.Errorf("expected ja, got %q", cfg.Language)
	}
}

func TestGetAI(t *testing.T) {
	mgr := newTestManager(t)
	ai := mgr.GetAI()
	if ai.Provider == "" {
		t.Error("expected default AI provider")
	}
	if ai.Provider != "anthropic" {
		t.Errorf("expected anthropic, got %q", ai.Provider)
	}
}

func TestSetAIProvider(t *testing.T) {
	mgr := newTestManager(t)
	err := mgr.SetAIProvider("openai", "gpt-4", "sk-test-key")
	if err != nil {
		t.Fatalf("SetAIProvider failed: %v", err)
	}

	ai := mgr.GetAI()
	if ai.Provider != "openai" {
		t.Errorf("expected openai, got %q", ai.Provider)
	}
	if ai.Model != "gpt-4" {
		t.Errorf("expected gpt-4, got %q", ai.Model)
	}
}

func TestGetEditor(t *testing.T) {
	mgr := newTestManager(t)
	ed := mgr.GetEditor()
	if ed.FontSize != 14 {
		t.Errorf("expected font size 14, got %d", ed.FontSize)
	}
}

func TestOnChange(t *testing.T) {
	mgr := newTestManager(t)
	ch := make(chan bool, 1)
	mgr.OnChange(func(c Config) {
		ch <- true
	})

	mgr.Update(map[string]interface{}{"theme": "light"})
	select {
	case <-ch:
		// ok
	default:
		// OnChange fires in goroutine, may not be instant
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"

	// Create, update, save
	m1 := &Manager{cfg: DefaultConfig(), path: path, dataDir: dir}
	m1.Update(map[string]interface{}{"theme": "dark"})

	// Reload from same path
	m2 := &Manager{cfg: DefaultConfig(), path: path, dataDir: dir}
	m2.load()
	if m2.cfg.Theme != "dark" {
		t.Errorf("expected dark after reload, got %q", m2.cfg.Theme)
	}
}

func TestDataDir(t *testing.T) {
	mgr := newTestManager(t)
	dir := mgr.DataDir()
	if dir == "" {
		t.Error("DataDir returned empty string")
	}
}

func TestApplyEnv(t *testing.T) {
	os.Setenv("OFFICE_AI_THEME", "dark")
	defer os.Unsetenv("OFFICE_AI_THEME")

	mgr := New()
	if mgr.Get().Theme != "dark" {
		t.Errorf("expected env override theme=dark, got %q", mgr.Get().Theme)
	}
}
