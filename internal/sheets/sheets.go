// Package sheets implements the spreadsheet editor module,
// mirroring GenOffice's apps/sheets. Uses a Rust sidecar (xlsx-engine)
// for parsing/recalc and a Go gateway for XML manipulation.
package sheets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// CellValue represents a cell's value.
type CellValue struct {
	Row    int         `json:"row"`
	Col    int         `json:"col"`
	Value  interface{} `json:"value"`
	Formula string    `json:"formula,omitempty"`
}

// WorksheetInfo describes a worksheet.
type WorksheetInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RowCount    int    `json:"row_count"`
	ColumnCount int    `json:"column_count"`
	Hidden      bool   `json:"hidden"`
	TabColor    string `json:"tab_color,omitempty"`
}

// WorkbookState holds the in-memory state of an open spreadsheet.
type WorkbookState struct {
	FilePath   string          `json:"file_path"`
	Title      string          `json:"title"`
	IsDirty    bool            `json:"is_dirty"`
	Sheets     []WorksheetInfo `json:"sheets"`
	ActiveSheet string         `json:"active_sheet"`
}

// Service is the sheets module service bound to Wails.
type Service struct {
	ctx         context.Context
	i18nSvc     *i18n.Service
	store       *projectstore.Store
	agent       *agentcore.Loop
	mu          sync.RWMutex
	openBooks   map[string]*WorkbookState
	sidecarPath string
}

// New creates a new sheets Service.
func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:   i18nSvc,
		store:     store,
		agent:     agent,
		openBooks: make(map[string]*WorkbookState),
	}
}

// SetSidecarPath sets the path to the Rust xlsx-engine sidecar binary.
func (s *Service) SetSidecarPath(path string) {
	s.sidecarPath = path
}

// OpenFile opens an xlsx file via the Rust sidecar.
func (s *Service) OpenFile(tabID, path string) map[string]interface{} {
	result := map[string]interface{}{"success": false}

	if path == "" {
		result["error"] = "no file path provided"
		return result
	}

	// Try to use the Rust sidecar for parsing
	snapshot, err := s.callSidecar("open", map[string]interface{}{"path": path})
	if err != nil {
		// Fallback: basic xlsx detection
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result["error"] = fmt.Sprintf("read file: %v", readErr)
			return result
		}
		if len(data) < 4 || string(data[:2]) != "PK" {
			result["error"] = "not a valid xlsx file"
			return result
		}
		// Create minimal state without sidecar
		snapshot = map[string]interface{}{
			"sheets": []map[string]interface{}{
				{"id": "sheet1", "name": "Sheet1", "row_count": 100, "column_count": 26},
			},
		}
	}

	state := &WorkbookState{
		FilePath: path,
		Title:    filepath.Base(path),
	}

	// Parse sheets from sidecar response
	if sheetsRaw, ok := snapshot["sheets"]; ok {
		sheetsJSON, _ := json.Marshal(sheetsRaw)
		json.Unmarshal(sheetsJSON, &state.Sheets)
	}
	if len(state.Sheets) > 0 {
		state.ActiveSheet = state.Sheets[0].ID
	}

	s.mu.Lock()
	s.openBooks[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["state"] = state
	return result
}

// NewBlank creates a new blank workbook.
func (s *Service) NewBlank(tabID string) map[string]interface{} {
	state := &WorkbookState{
		Title: "Untitled Spreadsheet",
		Sheets: []WorksheetInfo{
			{ID: "sheet1", Name: "Sheet1", RowCount: 1000, ColumnCount: 26},
		},
		ActiveSheet: "sheet1",
	}

	s.mu.Lock()
	s.openBooks[tabID] = state
	s.mu.Unlock()

	return map[string]interface{}{
		"success": true,
		"title":   state.Title,
		"state":   state,
	}
}

// GetCellRange reads a range of cells from the active worksheet.
func (s *Service) GetCellRange(tabID string, startRow, startCol, endRow, endCol int) []CellValue {
	// TODO: delegate to Rust sidecar for actual cell data
	return nil
}

// SetCellValue sets a single cell's value.
func (s *Service) SetCellValue(tabID string, row, col int, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.openBooks[tabID]; ok {
		state.IsDirty = true
	}
	// TODO: delegate to sidecar for formula recalculation
}

// Recalc triggers formula recalculation via the Rust sidecar.
func (s *Service) Recalc(tabID string) error {
	_, err := s.callSidecar("recalc", map[string]interface{}{"tab_id": tabID})
	return err
}

// Save saves the workbook.
func (s *Service) Save(tabID string) map[string]interface{} {
	s.mu.RLock()
	state, ok := s.openBooks[tabID]
	s.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"success": false, "error": "workbook not found"}
	}
	if state.FilePath == "" {
		return map[string]interface{}{"success": false, "error": "no file path - use SaveAs"}
	}

	// TODO: Use xlsx-gateway to write back changes
	return map[string]interface{}{"success": true, "file_path": state.FilePath}
}

// Close closes an open workbook.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openBooks, tabID)
}

// callSidecar invokes the Rust xlsx-engine sidecar.
func (s *Service) callSidecar(command string, args map[string]interface{}) (map[string]interface{}, error) {
	sidecar := s.sidecarPath
	if sidecar == "" {
		// Try to find the sidecar in the expected build location
		sidecar = filepath.Join("native", "xlsx-engine", "target", "release", "xlsx-sidecar")
	}

	if _, err := os.Stat(sidecar); os.IsNotExist(err) {
		return nil, fmt.Errorf("xlsx-engine sidecar not found at %s", sidecar)
	}

	input, _ := json.Marshal(map[string]interface{}{
		"command": command,
		"args":    args,
	})

	cmd := exec.Command(sidecar)
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), fmt.Sprintf("XLSX_INPUT=%s", string(input)))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sidecar error: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse sidecar output: %w", err)
	}
	return result, nil
}
