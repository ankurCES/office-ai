// Package pdf implements the PDF viewer/editor module using pdfcpu.
// Handles PDF page info, text extraction, page operations (extract, split,
// merge, rotate), watermarks, and metadata.
package pdf

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
)

// PageInfo describes a single PDF page.
type PageInfo struct {
	Index  int     `json:"index"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PDFMeta holds PDF document metadata.
type PDFMeta struct {
	Title    string `json:"title,omitempty"`
	Author   string `json:"author,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Creator  string `json:"creator,omitempty"`
	Producer string `json:"producer,omitempty"`
}

// PDFState holds the in-memory state of an open PDF.
type PDFState struct {
	FilePath  string     `json:"file_path"`
	Title     string     `json:"title"`
	IsDirty   bool       `json:"is_dirty"`
	PageCount int        `json:"page_count"`
	Pages     []PageInfo `json:"pages"`
	Meta      PDFMeta    `json:"meta"`
}

// OpResult is the standard result type for PDF operations.
type OpResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path,omitempty"`
	Error    string `json:"error,omitempty"`
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

// Startup stores the Wails context.
func (s *Service) Startup(ctx context.Context) {
	s.ctx = ctx
}

// OpenFile opens a PDF file and parses its structure via pdfcpu.
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

	// Parse with pdfcpu to get page count and dimensions
	pdfCtx, err := pdfapi.ReadContextFile(path)
	if err != nil {
		result["error"] = fmt.Sprintf("parse PDF: %v", err)
		return result
	}

	_ = pdfCtx.EnsurePageCount()
	pageCount := pdfCtx.PageCount

	pages := make([]PageInfo, pageCount)
	dims, dimsErr := pdfCtx.PageDims()
	for i := 0; i < pageCount; i++ {
		w, h := 612.0, 792.0 // default US Letter
		if dimsErr == nil && i < len(dims) {
			w = dims[i].Width
			h = dims[i].Height
		}
		pages[i] = PageInfo{Index: i, Width: w, Height: h}
	}

	// Extract metadata from XRefTable
	meta := PDFMeta{}
	if xrt := pdfCtx.XRefTable; xrt != nil {
		meta.Title = xrt.Title
		meta.Author = xrt.Author
		meta.Creator = xrt.Creator
		meta.Producer = xrt.Producer
		meta.Subject = xrt.Subject
	}

	title := filepath.Base(path)
	if meta.Title != "" {
		title = meta.Title
	}

	state := &PDFState{
		FilePath:  path,
		Title:     title,
		PageCount: pageCount,
		Pages:     pages,
		Meta:      meta,
	}

	s.mu.Lock()
	s.openPDFs[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["page_count"] = pageCount
	result["pages"] = pages
	result["meta"] = meta
	return result
}

// GetState returns the current state of an open PDF.
func (s *Service) GetState(tabID string) *PDFState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openPDFs[tabID]
}

// ExtractText extracts raw content streams from the PDF into a temp dir
// and returns the output directory path.
func (s *Service) ExtractText(tabID string) (string, error) {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("PDF not found")
	}

	outDir, err := os.MkdirTemp("", "pdf-text-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	if err := pdfapi.ExtractContentFile(state.FilePath, outDir, nil, nil); err != nil {
		return "", fmt.Errorf("extract content: %w", err)
	}

	return outDir, nil
}

// ExtractPages extracts specified pages into a new PDF file.
func (s *Service) ExtractPages(tabID string, pageNums []int, outputPath string) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}

	// pdfcpu uses 1-based page numbers
	pageStrs := make([]string, len(pageNums))
	for i, p := range pageNums {
		pageStrs[i] = fmt.Sprintf("%d", p+1)
	}
	selectedPages := []string{strings.Join(pageStrs, ",")}

	if err := pdfapi.ExtractPagesFile(state.FilePath, outputPath, selectedPages, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("extract pages: %v", err)}
	}

	return OpResult{Success: true, FilePath: outputPath}
}

// MergeFiles merges multiple PDF files into one.
func (s *Service) MergeFiles(inputPaths []string, outputPath string) OpResult {
	if len(inputPaths) < 2 {
		return OpResult{Success: false, Error: "need at least 2 files to merge"}
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("create output: %v", err)}
	}
	defer outFile.Close()

	var readers []io.ReadSeeker
	var closers []io.Closer
	for _, p := range inputPaths {
		f, err := os.Open(p)
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
			return OpResult{Success: false, Error: fmt.Sprintf("open %s: %v", filepath.Base(p), err)}
		}
		readers = append(readers, f)
		closers = append(closers, f)
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	if err := pdfapi.MergeRaw(readers, outFile, false, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("merge: %v", err)}
	}

	return OpResult{Success: true, FilePath: outputPath}
}

// SplitFile splits a PDF into individual page files.
func (s *Service) SplitFile(tabID, outputDir string) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("create dir: %v", err)}
	}

	if err := pdfapi.SplitFile(state.FilePath, outputDir, 1, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("split: %v", err)}
	}

	return OpResult{Success: true, FilePath: outputDir}
}

// RotatePages rotates specified pages by the given angle (90, 180, 270).
func (s *Service) RotatePages(tabID string, rotation int, pageNums []int) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}

	if rotation != 90 && rotation != 180 && rotation != 270 {
		return OpResult{Success: false, Error: "rotation must be 90, 180, or 270"}
	}

	pageStrs := make([]string, len(pageNums))
	for i, p := range pageNums {
		pageStrs[i] = fmt.Sprintf("%d", p+1)
	}
	selectedPages := []string{strings.Join(pageStrs, ",")}

	if err := pdfapi.RotateFile(state.FilePath, "", rotation, selectedPages, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("rotate: %v", err)}
	}

	return s.reload(tabID, state.FilePath)
}

// AddWatermark adds a text watermark to all pages.
func (s *Service) AddWatermark(tabID, text, outputPath string) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}

	desc := fmt.Sprintf("scale:1.0, rotation:45, opacity:0.3")
	onTop := true
	if err := pdfapi.AddTextWatermarksFile(state.FilePath, outputPath, nil, onTop, text, desc, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("watermark: %v", err)}
	}

	return OpResult{Success: true, FilePath: outputPath}
}

// Validate checks if a PDF file is valid.
func (s *Service) Validate(path string) OpResult {
	if err := pdfapi.ValidateFile(path, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("invalid: %v", err)}
	}
	return OpResult{Success: true, FilePath: path}
}

// Optimize reduces PDF file size.
func (s *Service) Optimize(tabID, outputPath string) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}

	if err := pdfapi.OptimizeFile(state.FilePath, outputPath, nil); err != nil {
		return OpResult{Success: false, Error: fmt.Sprintf("optimize: %v", err)}
	}

	return OpResult{Success: true, FilePath: outputPath}
}

// Save is a no-op for PDFs since pdfcpu operates on files directly.
func (s *Service) Save(tabID string) OpResult {
	s.mu.RLock()
	state, ok := s.openPDFs[tabID]
	s.mu.RUnlock()

	if !ok {
		return OpResult{Success: false, Error: "PDF not found"}
	}
	return OpResult{Success: true, FilePath: state.FilePath}
}

// Close closes an open PDF.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openPDFs, tabID)
}

func (s *Service) reload(tabID, path string) OpResult {
	s.Close(tabID)
	result := s.OpenFile(tabID, path)
	if success, ok := result["success"].(bool); ok && success {
		return OpResult{Success: true, FilePath: path}
	}
	errMsg := "reload failed"
	if e, ok := result["error"].(string); ok {
		errMsg = e
	}
	return OpResult{Success: false, Error: errMsg}
}
