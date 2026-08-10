// Package docs implements the document editor module,
// mirroring GenOffice's apps/docs. Handles docx parsing, editing,
// saving (paragraph-patch), AI skill, and file operations.
package docs

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

// DocState holds the in-memory state of an open document.
type DocState struct {
	FilePath    string   `json:"file_path"`
	IsDirty     bool     `json:"is_dirty"`
	Title       string   `json:"title"`
	WordCount   int      `json:"word_count"`
	PageCount   int      `json:"page_count"`
	Paragraphs  []string `json:"paragraphs,omitempty"` // simplified; real would be rich XML
}

// OpenFileResult describes the outcome of opening a document.
type OpenFileResult struct {
	Success  bool      `json:"success"`
	FilePath string    `json:"file_path"`
	Title    string    `json:"title"`
	Error    string    `json:"error,omitempty"`
	State    *DocState `json:"state,omitempty"`
}

// SaveResult describes the outcome of saving a document.
type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
}

// Service is the docs module service bound to Wails.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	store     *projectstore.Store
	agent     *agentcore.Loop
	mu        sync.RWMutex
	openDocs  map[string]*DocState // keyed by tab ID
}

// New creates a new docs Service.
func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:  i18nSvc,
		store:    store,
		agent:    agent,
		openDocs: make(map[string]*DocState),
	}
}

// OpenFile opens a .docx file and returns its parsed state.
func (s *Service) OpenFile(tabID, path string) OpenFileResult {
	if path == "" {
		return OpenFileResult{Success: false, Error: "no file path provided"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return OpenFileResult{Success: false, Error: fmt.Sprintf("read file: %v", err)}
	}

	// Parse docx - simplified; real implementation uses docx-engine equivalent
	state := &DocState{
		FilePath: path,
		Title:    filepath.Base(path),
	}

	// Basic docx validation (ZIP with word/document.xml)
	if len(data) < 4 || string(data[:2]) != "PK" {
		return OpenFileResult{Success: false, Error: "not a valid docx file"}
	}

	// TODO: Full XML parsing with docxengine package
	state.WordCount = len(data) / 6 // rough estimate
	state.PageCount = 1

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	return OpenFileResult{
		Success:  true,
		FilePath: path,
		Title:    state.Title,
		State:    state,
	}
}

// NewBlank creates a new blank document.
func (s *Service) NewBlank(tabID string) OpenFileResult {
	state := &DocState{
		Title:     "Untitled Document",
		WordCount: 0,
		PageCount: 1,
	}

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	return OpenFileResult{
		Success: true,
		Title:   state.Title,
		State:   state,
	}
}

// Save saves the document to its current path.
func (s *Service) Save(tabID string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok {
		return SaveResult{Success: false, Error: "document not found"}
	}
	if state.FilePath == "" {
		return SaveResult{Success: false, Error: "no file path - use SaveAs"}
	}

	// TODO: Use docxengine to generate the actual docx bytes with paragraph-patch save
	return SaveResult{Success: true, FilePath: state.FilePath}
}

// SaveAs saves the document to a new path.
func (s *Service) SaveAs(tabID, path string) SaveResult {
	s.mu.Lock()
	state, ok := s.openDocs[tabID]
	if ok {
		state.FilePath = path
		state.Title = filepath.Base(path)
		state.IsDirty = false
	}
	s.mu.Unlock()

	if !ok {
		return SaveResult{Success: false, Error: "document not found"}
	}

	// TODO: Use docxengine to generate the actual docx bytes
	return SaveResult{Success: true, FilePath: path}
}

// GetState returns the current state of an open document.
func (s *Service) GetState(tabID string) *DocState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openDocs[tabID]
}

// IsDirty returns whether the document has unsaved changes.
func (s *Service) IsDirty(tabID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDocs[tabID]; ok {
		return state.IsDirty
	}
	return false
}

// Close closes an open document and cleans up resources.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openDocs, tabID)
}

// --- AI Skill ---

// GetDocsSkill returns the agent skill for document operations.
func (s *Service) GetDocsSkill() *agentcore.Skill {
	return &agentcore.Skill{
		ID: "docs",
		SystemPrompt: `You are an AI assistant for document editing. You can:
- Insert, replace, and delete text in the document
- Format paragraphs (alignment, spacing, indentation)
- Insert tables, images, and other elements
- Apply styles and themes
- Manage headers, footers, and page numbering
Always describe what you changed after each edit.`,
		Tools: []agentcore.ToolDef{
			{
				Name:        "insert_text",
				Description: "Insert text at the cursor position or specified paragraph index",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text":      map[string]interface{}{"type": "string", "description": "Text to insert"},
						"paragraph": map[string]interface{}{"type": "integer", "description": "Paragraph index (0-based)"},
					},
					"required": []string{"text"},
				},
			},
			{
				Name:        "replace_text",
				Description: "Replace text matching a search string",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search":  map[string]interface{}{"type": "string"},
						"replace": map[string]interface{}{"type": "string"},
					},
					"required": []string{"search", "replace"},
				},
			},
			{
				Name:        "format_paragraph",
				Description: "Apply formatting to a paragraph",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"paragraph": map[string]interface{}{"type": "integer"},
						"alignment": map[string]interface{}{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
						"style":     map[string]interface{}{"type": "string"},
					},
					"required": []string{"paragraph"},
				},
			},
		},
		ExecuteTool: func(ctx context.Context, call agentcore.ToolCall) agentcore.ToolResult {
			// TODO: implement actual document manipulation
			return agentcore.ToolResult{
				ID:     call.ID,
				Name:   call.Name,
				Output: fmt.Sprintf("Executed %s with input %v", call.Name, call.Input),
			}
		},
	}
}
