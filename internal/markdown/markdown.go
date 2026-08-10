// Package markdown implements the markdown editor module,
// mirroring GenOffice's apps/markdown. Provides file load/save
// and AI skill for markdown editing.
package markdown

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// MarkdownState holds the in-memory state of an open markdown file.
type MarkdownState struct {
	FilePath string `json:"file_path"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	IsDirty  bool   `json:"is_dirty"`
}

// Service is the markdown module service bound to Wails.
type Service struct {
	ctx      context.Context
	i18nSvc  *i18n.Service
	store    *projectstore.Store
	agent    *agentcore.Loop
	mu       sync.RWMutex
	openDocs map[string]*MarkdownState
}

// New creates a new markdown Service.
func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:  i18nSvc,
		store:    store,
		agent:    agent,
		openDocs: make(map[string]*MarkdownState),
	}
}

// OpenFile opens a markdown file.
func (s *Service) OpenFile(tabID, path string) map[string]interface{} {
	result := map[string]interface{}{"success": false}

	data, err := os.ReadFile(path)
	if err != nil {
		result["error"] = fmt.Sprintf("read file: %v", err)
		return result
	}

	state := &MarkdownState{
		FilePath: path,
		Title:    filepath.Base(path),
		Content:  string(data),
	}

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["content"] = state.Content
	return result
}

// NewBlank creates a new blank markdown document.
func (s *Service) NewBlank(tabID string) map[string]interface{} {
	state := &MarkdownState{
		Title:   "Untitled",
		Content: "",
	}

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	return map[string]interface{}{
		"success": true,
		"title":   state.Title,
		"content": state.Content,
	}
}

// UpdateContent updates the markdown content (called on each edit).
func (s *Service) UpdateContent(tabID, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.openDocs[tabID]; ok {
		state.Content = content
		state.IsDirty = true
	}
}

// Save saves the markdown file.
func (s *Service) Save(tabID string) map[string]interface{} {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"success": false, "error": "document not found"}
	}
	if state.FilePath == "" {
		return map[string]interface{}{"success": false, "error": "no file path - use SaveAs"}
	}

	err := os.WriteFile(state.FilePath, []byte(state.Content), 0644)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	s.mu.Lock()
	state.IsDirty = false
	s.mu.Unlock()

	return map[string]interface{}{"success": true, "file_path": state.FilePath}
}

// SaveAs saves the markdown file to a new path.
func (s *Service) SaveAs(tabID, path string) map[string]interface{} {
	s.mu.Lock()
	state, ok := s.openDocs[tabID]
	if !ok {
		s.mu.Unlock()
		return map[string]interface{}{"success": false, "error": "document not found"}
	}

	err := os.WriteFile(path, []byte(state.Content), 0644)
	if err != nil {
		s.mu.Unlock()
		return map[string]interface{}{"success": false, "error": err.Error()}
	}

	state.FilePath = path
	state.Title = filepath.Base(path)
	state.IsDirty = false
	s.mu.Unlock()

	return map[string]interface{}{"success": true, "file_path": path}
}

// Close closes an open markdown document.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openDocs, tabID)
}
