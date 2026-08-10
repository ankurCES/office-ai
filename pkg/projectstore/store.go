// Package projectstore provides project and chat persistence,
// mirroring GenOffice's project-store package.
package projectstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProjectInfo holds metadata for a project.
type ProjectInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FilePath  string    `json:"file_path"`
	Kind      string    `json:"kind"` // "docs", "sheets", "slides", "pdf", "markdown"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage represents a single chat message in a project.
type ChatMessage struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatMeta holds metadata about a chat session.
type ChatMeta struct {
	ID        string        `json:"id"`
	ProjectID string        `json:"project_id"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
}

// TimelineEntry is a timestamped event in a project.
type TimelineEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail,omitempty"`
}

// Store persists projects and chat history to disk (JSON files in a data dir).
type Store struct {
	mu      sync.RWMutex
	dataDir string
	index   map[string]*ProjectInfo
}

// New creates a new Store. Data is stored in ~/.office-ai/projects/.
func New() *Store {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".office-ai", "projects")
	os.MkdirAll(dataDir, 0755)

	s := &Store{
		dataDir: dataDir,
		index:   make(map[string]*ProjectInfo),
	}
	s.loadIndex()
	return s
}

// CreateProject creates and persists a new project.
func (s *Store) CreateProject(info ProjectInfo) (*ProjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info.ID == "" {
		info.ID = fmt.Sprintf("proj-%d", time.Now().UnixNano())
	}
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()

	s.index[info.ID] = &info
	return &info, s.saveIndex()
}

// GetProject returns a project by ID.
func (s *Store) GetProject(id string) (*ProjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.index[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return p, nil
}

// ListProjects returns all projects.
func (s *Store) ListProjects() []*ProjectInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ProjectInfo
	for _, p := range s.index {
		result = append(result, p)
	}
	return result
}

// UpdateProject updates a project's metadata.
func (s *Store) UpdateProject(id string, info ProjectInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.index[id]; !ok {
		return fmt.Errorf("project not found: %s", id)
	}
	info.UpdatedAt = time.Now()
	s.index[id] = &info
	return s.saveIndex()
}

// DeleteProject removes a project.
func (s *Store) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.index, id)
	// Remove chat history
	chatDir := filepath.Join(s.dataDir, id)
	os.RemoveAll(chatDir)
	return s.saveIndex()
}

// SaveChat persists a chat session for a project.
func (s *Store) SaveChat(chat ChatMeta) error {
	chatDir := filepath.Join(s.dataDir, chat.ProjectID)
	os.MkdirAll(chatDir, 0755)

	data, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(chatDir, chat.ID+".json"), data, 0644)
}

// LoadChat loads a chat session.
func (s *Store) LoadChat(projectID, chatID string) (*ChatMeta, error) {
	path := filepath.Join(s.dataDir, projectID, chatID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var chat ChatMeta
	if err := json.Unmarshal(data, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

func (s *Store) loadIndex() {
	path := filepath.Join(s.dataDir, "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.index)
}

func (s *Store) saveIndex() error {
	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dataDir, "index.json"), data, 0644)
}
