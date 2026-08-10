// Package pdf implements the PDF viewer/editor module,
// mirroring GenOffice's apps/pdf. Handles PDF rendering, text layer,
// thumbnails, page extraction, and basic text editing.
package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
)

// PageInfo describes a single PDF page.
type PageInfo struct {
	Index  int     `json:"index"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PDFState holds the in-memory state of an open PDF.
type PDFState struct {
	FilePath  string     `json:"file_path"`
	Title     string     `json:"title"`
	IsDirty   bool       `json:"is_dirty"`
	PageCount int        `json:"page_count"`
	Pages     []PageInfo `json:"pages"`
}

// Service is the PDF module service bound to Wails.
type Service struct {
	ctx      context.Context
	i18nSvc  *i18n.Service
	agent    *agentcore.Loop
	mu       sync.RWMutex
	openPDFs map[string]*PDFState
}

// New creates a new PDF Service.
func New(i18nSvc *i18n.Service, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:  i18nSvc,
		agent:    agent,
		openPDFs: make(map[string]*PDFState),
	}
}

// OpenFile opens a PDF file.
func (s *Service) OpenFile(tabID, path string) map[string]interface{} {
	result := map[string]interface{}{"success": false}

	data, err := os.ReadFile(path)
	if err != nil {
		result["error"] = fmt.Sprintf("read file: %v", err)
		return result
	}

	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		result["error"] = "not a valid PDF file"
		return result
	}

	// TODO: Use pdfcpu or pdfium for real page parsing
	state := &PDFState{
		FilePath:  path,
		Title:     filepath.Base(path),
		PageCount: 1, // placeholder
		Pages:     []PageInfo{{Index: 0, Width: 612, Height: 792}},
	}

	s.mu.Lock()
	s.openPDFs[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["state"] = state
	return result
}

// GetPagePreview returns a base64-encoded preview image of a page.
func (s *Service) GetPagePreview(tabID string, pageIndex int) string {
	// TODO: Render PDF page to image using pdfium or pdfcpu
	return ""
}

// ExtractPages extracts specified pages into a new PDF.
func (s *Service) ExtractPages(tabID string, pages []int, outputPath string) map[string]interface{} {
	// TODO: Use pdfcpu to extract pages
	return map[string]interface{}{
		"success": false,
		"error":   "page extraction not yet implemented",
	}
}

// Save saves any edits to the PDF.
func (s *Service) Save(tabID string) map[string]interface{} {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"success": false, "error": "PDF not found"}
	}
	if state.FilePath == "" {
		return map[string]interface{}{"success": false, "error": "no file path"}
	}

	return map[string]interface{}{"success": true, "file_path": state.FilePath}
}

// Close closes an open PDF.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openPDFs, tabID)
}
