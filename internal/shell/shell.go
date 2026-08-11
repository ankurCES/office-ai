// Package shell implements the unified shell/tab manager,
// mirroring GenOffice's apps/shell. Owns tab lifecycle, Home screen,
// app settings, recent files, updater, and cloud projects.
package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ankurCES/office-ai/pkg/i18n"
)

// TabKind identifies the type of tab content.
type TabKind string

const (
	TabHome     TabKind = "home"
	TabDocs     TabKind = "docs"
	TabSheets   TabKind = "sheets"
	TabSlides   TabKind = "slides"
	TabPDF      TabKind = "pdf"
	TabMarkdown TabKind = "markdown"
)

// TabSummary describes a tab for the frontend tab bar.
type TabSummary struct {
	ID       string  `json:"id"`
	Kind     TabKind `json:"kind"`
	Title    string  `json:"title"`
	FilePath string  `json:"file_path,omitempty"`
	IsDirty  bool    `json:"is_dirty"`
	IsActive bool    `json:"is_active"`
}

// RecentFile represents a recently opened file.
type RecentFile struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Kind      TabKind   `json:"kind"`
	OpenedAt  time.Time `json:"opened_at"`
	IsStarred bool      `json:"is_starred"`
}

// AppSettings holds persistent application preferences.
type AppSettings struct {
	Language     i18n.Lang `json:"language"`
	Theme        string    `json:"theme"` // "light", "dark", "system"
	OnboardDone  bool      `json:"onboard_done"`
	DefaultSaveDir string  `json:"default_save_dir,omitempty"`
	UpdateChannel  string  `json:"update_channel"` // "stable", "beta"
}

// tab is the internal tab record.
type tab struct {
	ID       string
	Kind     TabKind
	Title    string
	FilePath string
	IsDirty  bool
}

// Service is the shell/tab manager service bound to the Wails frontend.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	mu        sync.RWMutex
	tabs      []*tab
	activeID  string
	nextID    int
	settings  AppSettings
	recents   []RecentFile
	settingsPath string
	recentsPath  string
}

// New creates a new shell Service.
func New(i18nSvc *i18n.Service) *Service {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".quill")
	os.MkdirAll(dataDir, 0755)

	s := &Service{
		i18nSvc:      i18nSvc,
		nextID:       1,
		settingsPath: filepath.Join(dataDir, "app-settings.json"),
		recentsPath:  filepath.Join(dataDir, "recent-files.json"),
		settings: AppSettings{
			Language:      i18n.EN,
			Theme:         "system",
			UpdateChannel: "stable",
		},
	}
	// Initialize with Home tab
	s.tabs = []*tab{{ID: "home", Kind: TabHome, Title: "Quill"}}
	s.activeID = "home"
	s.loadSettings()
	s.loadRecents()
	return s
}

// OnStartup is called when the Wails app starts.
func (s *Service) OnStartup(ctx context.Context) {
	s.ctx = ctx
	s.i18nSvc.SetLang(s.settings.Language)
}

// OnShutdown is called when the Wails app shuts down.
func (s *Service) OnShutdown(ctx context.Context) {
	s.saveSettings()
	s.saveRecents()
}

// --- Tab Management (mirrors GenOffice TabManager) ---

// GetTabs returns all open tabs.
func (s *Service) GetTabs() []TabSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summaries := make([]TabSummary, len(s.tabs))
	for i, t := range s.tabs {
		summaries[i] = TabSummary{
			ID:       t.ID,
			Kind:     t.Kind,
			Title:    t.Title,
			FilePath: t.FilePath,
			IsDirty:  t.IsDirty,
			IsActive: t.ID == s.activeID,
		}
	}
	return summaries
}

// OpenTab creates a new tab of the given kind and returns its ID.
func (s *Service) OpenTab(kind TabKind, filePath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("tab-%d", s.nextID)
	s.nextID++

	title := s.untitledFor(kind)
	if filePath != "" {
		title = filepath.Base(filePath)
	}

	t := &tab{
		ID:       id,
		Kind:     kind,
		Title:    title,
		FilePath: filePath,
	}
	s.tabs = append(s.tabs, t)
	s.activeID = id

	// Record as recent file
	if filePath != "" {
		s.recordRecent(filePath, kind)
	}
	return id
}

// ActivateTab switches to the given tab.
func (s *Service) ActivateTab(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tabs {
		if t.ID == id {
			s.activeID = id
			return
		}
	}
}

// CloseTab closes a tab by ID. Returns true if closed.
func (s *Service) CloseTab(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "home" {
		return false // can't close home
	}

	idx := -1
	for i, t := range s.tabs {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}

	s.tabs = append(s.tabs[:idx], s.tabs[idx+1:]...)

	// Activate nearest tab
	if s.activeID == id {
		if idx > 0 {
			s.activeID = s.tabs[idx-1].ID
		} else if len(s.tabs) > 0 {
			s.activeID = s.tabs[0].ID
		}
	}
	return true
}

// SetTabDirty marks a tab as having unsaved changes.
func (s *Service) SetTabDirty(id string, dirty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tabs {
		if t.ID == id {
			t.IsDirty = dirty
			return
		}
	}
}

// SetTabTitle updates a tab's display title.
func (s *Service) SetTabTitle(id, title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tabs {
		if t.ID == id {
			t.Title = title
			return
		}
	}
}

// --- Settings ---

// GetSettings returns the current app settings.
func (s *Service) GetSettings() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// UpdateSetting updates a single setting key.
func (s *Service) UpdateSetting(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch key {
	case "language":
		if lang, ok := value.(string); ok {
			s.settings.Language = i18n.Lang(lang)
			s.i18nSvc.SetLang(s.settings.Language)
		}
	case "theme":
		if theme, ok := value.(string); ok {
			s.settings.Theme = theme
		}
	case "onboard_done":
		if done, ok := value.(bool); ok {
			s.settings.OnboardDone = done
		}
	case "default_save_dir":
		if dir, ok := value.(string); ok {
			s.settings.DefaultSaveDir = dir
		}
	case "update_channel":
		if ch, ok := value.(string); ok {
			s.settings.UpdateChannel = ch
		}
	}
	s.saveSettings()
}

// --- Recent Files ---

// GetRecentFiles returns the recent files list.
func (s *Service) GetRecentFiles() []RecentFile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RecentFile, len(s.recents))
	copy(result, s.recents)
	return result
}

// ToggleStarred toggles the starred status of a recent file.
func (s *Service) ToggleStarred(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.recents {
		if s.recents[i].Path == path {
			s.recents[i].IsStarred = !s.recents[i].IsStarred
			break
		}
	}
	s.saveRecents()
}

// RemoveRecent removes a file from the recents list.
func (s *Service) RemoveRecent(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.recents {
		if s.recents[i].Path == path {
			s.recents = append(s.recents[:i], s.recents[i+1:]...)
			break
		}
	}
	s.saveRecents()
}

func (s *Service) untitledFor(kind TabKind) string {
	switch kind {
	case TabDocs:
		return "Untitled Document"
	case TabSheets:
		return "Untitled Spreadsheet"
	case TabSlides:
		return "Untitled Presentation"
	case TabPDF:
		return "Untitled PDF"
	case TabMarkdown:
		return "Untitled"
	default:
		return "Untitled"
	}
}

func (s *Service) recordRecent(path string, kind TabKind) {
	// Remove existing entry for this path
	for i := range s.recents {
		if s.recents[i].Path == path {
			s.recents = append(s.recents[:i], s.recents[i+1:]...)
			break
		}
	}
	// Prepend
	entry := RecentFile{
		Path:     path,
		Name:     filepath.Base(path),
		Kind:     kind,
		OpenedAt: time.Now(),
	}
	s.recents = append([]RecentFile{entry}, s.recents...)
	// Cap at 50
	if len(s.recents) > 50 {
		s.recents = s.recents[:50]
	}
}

func (s *Service) loadSettings() {
	data, err := os.ReadFile(s.settingsPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.settings)
}

func (s *Service) saveSettings() {
	data, _ := json.MarshalIndent(s.settings, "", "  ")
	os.WriteFile(s.settingsPath, data, 0644)
}

func (s *Service) loadRecents() {
	data, err := os.ReadFile(s.recentsPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.recents)
}

func (s *Service) saveRecents() {
	data, _ := json.MarshalIndent(s.recents, "", "  ")
	os.WriteFile(s.recentsPath, data, 0644)
}
