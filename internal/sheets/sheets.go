// Package sheets implements the spreadsheet editor module using excelize.
// Full xlsx read/write: cell values, formulas, styles, multi-sheet,
// charts, images, auto-filter, merge cells, conditional formatting.
package sheets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/xuri/excelize/v2"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// CellData represents a cell's value and metadata.
type CellData struct {
	Row     int    `json:"row"`
	Col     int    `json:"col"`
	Value   string `json:"value"`
	Formula string `json:"formula,omitempty"`
	Type    string `json:"type"` // string, number, bool, date, formula, empty
}

// WorksheetInfo describes a worksheet tab.
type WorksheetInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Index       int    `json:"index"`
	RowCount    int    `json:"row_count"`
	ColumnCount int    `json:"column_count"`
	Hidden      bool   `json:"hidden"`
}

// MergeCell describes a merged cell range.
type MergeCell struct {
	StartCell string `json:"start_cell"`
	EndCell   string `json:"end_cell"`
	Value     string `json:"value"`
}

// WorkbookState holds the in-memory state of an open workbook.
type WorkbookState struct {
	FilePath    string          `json:"file_path"`
	Title       string          `json:"title"`
	IsDirty     bool            `json:"is_dirty"`
	Sheets      []WorksheetInfo `json:"sheets"`
	ActiveSheet string          `json:"active_sheet"`
	file        *excelize.File  // live excelize handle
}

// SaveResult is the standard result type for save operations.
type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Service is the sheets module service bound to Wails.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	store     *projectstore.Store
	agent     *agentcore.Loop
	mu        sync.RWMutex
	openBooks map[string]*WorkbookState
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

// Startup stores the Wails context.
func (s *Service) Startup(ctx context.Context) {
	s.ctx = ctx
}

// OpenFile opens an xlsx file and parses its structure.
func (s *Service) OpenFile(tabID, path string) map[string]interface{} {
	result := map[string]interface{}{"success": false}

	f, err := excelize.OpenFile(path)
	if err != nil {
		result["error"] = fmt.Sprintf("open xlsx: %v", err)
		return result
	}

	sheets := s.buildSheetList(f)
	activeSheet := f.GetSheetName(f.GetActiveSheetIndex())

	state := &WorkbookState{
		FilePath:    path,
		Title:       filepath.Base(path),
		Sheets:      sheets,
		ActiveSheet: activeSheet,
		file:        f,
	}

	s.mu.Lock()
	s.openBooks[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["sheets"] = sheets
	result["active_sheet"] = activeSheet
	return result
}

// NewWorkbook creates a new empty workbook.
func (s *Service) NewWorkbook(tabID string) map[string]interface{} {
	f := excelize.NewFile()

	sheets := s.buildSheetList(f)
	activeSheet := f.GetSheetName(f.GetActiveSheetIndex())

	state := &WorkbookState{
		Title:       "Untitled.xlsx",
		Sheets:      sheets,
		ActiveSheet: activeSheet,
		IsDirty:     true,
		file:        f,
	}

	s.mu.Lock()
	s.openBooks[tabID] = state
	s.mu.Unlock()

	return map[string]interface{}{
		"success":      true,
		"title":        state.Title,
		"sheets":       sheets,
		"active_sheet": activeSheet,
	}
}

func (s *Service) buildSheetList(f *excelize.File) []WorksheetInfo {
	sheetList := f.GetSheetList()
	sheets := make([]WorksheetInfo, len(sheetList))
	for i, name := range sheetList {
		idx, _ := f.GetSheetIndex(name)
		visible, _ := f.GetSheetVisible(name)
		rows, _ := f.GetRows(name)
		colCount := 0
		for _, row := range rows {
			if len(row) > colCount {
				colCount = len(row)
			}
		}
		sheets[i] = WorksheetInfo{
			ID:          idx,
			Name:        name,
			Index:       i,
			RowCount:    len(rows),
			ColumnCount: colCount,
			Hidden:      !visible,
		}
	}
	return sheets
}

// GetState returns the workbook state.
func (s *Service) GetState(tabID string) *WorkbookState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.openBooks[tabID]
	if st == nil {
		return nil
	}
	// Return a copy without the file handle
	return &WorkbookState{
		FilePath:    st.FilePath,
		Title:       st.Title,
		IsDirty:     st.IsDirty,
		Sheets:      st.Sheets,
		ActiveSheet: st.ActiveSheet,
	}
}

// GetCellRange reads a range of cells. startRow/startCol are 0-based.
func (s *Service) GetCellRange(tabID, sheetName string, startRow, startCol, rows, cols int) [][]CellData {
	s.mu.RLock()
	state, ok := s.openBooks[tabID]
	s.mu.RUnlock()

	if !ok || state.file == nil {
		return nil
	}

	result := make([][]CellData, rows)
	for r := 0; r < rows; r++ {
		result[r] = make([]CellData, cols)
		for c := 0; c < cols; c++ {
			cellName, _ := excelize.CoordinatesToCellName(startCol+c+1, startRow+r+1)
			value, _ := state.file.GetCellValue(sheetName, cellName)
			formula, _ := state.file.GetCellFormula(sheetName, cellName)

			cellType := "string"
			if formula != "" {
				cellType = "formula"
			} else if value == "" {
				cellType = "empty"
			} else if _, err := strconv.ParseFloat(value, 64); err == nil {
				cellType = "number"
			} else if value == "TRUE" || value == "FALSE" {
				cellType = "bool"
			}

			result[r][c] = CellData{
				Row:     startRow + r,
				Col:     startCol + c,
				Value:   value,
				Formula: formula,
				Type:    cellType,
			}
		}
	}
	return result
}

// GetAllRows returns all rows for a sheet as string arrays.
func (s *Service) GetAllRows(tabID, sheetName string) [][]string {
	s.mu.RLock()
	state, ok := s.openBooks[tabID]
	s.mu.RUnlock()

	if !ok || state.file == nil {
		return nil
	}

	rows, err := state.file.GetRows(sheetName)
	if err != nil {
		return nil
	}
	return rows
}

// SetCellValue sets a cell's value (auto-detects number/string).
func (s *Service) SetCellValue(tabID, sheetName, cellRef string, value string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	// Try to parse as number
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		state.file.SetCellValue(sheetName, cellRef, num)
	} else {
		state.file.SetCellValue(sheetName, cellRef, value)
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// SetCellFormula sets a cell's formula.
func (s *Service) SetCellFormula(tabID, sheetName, cellRef, formula string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.SetCellFormula(sheetName, cellRef, formula); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("set formula: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// AddSheet adds a new worksheet.
func (s *Service) AddSheet(tabID, name string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	idx, err := state.file.NewSheet(name)
	if err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("add sheet: %v", err)}
	}
	_ = idx

	s.mu.Lock()
	state.Sheets = s.buildSheetList(state.file)
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// DeleteSheet removes a worksheet.
func (s *Service) DeleteSheet(tabID, name string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.DeleteSheet(name); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("delete sheet: %v", err)}
	}

	s.mu.Lock()
	state.Sheets = s.buildSheetList(state.file)
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// RenameSheet renames a worksheet.
func (s *Service) RenameSheet(tabID, oldName, newName string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.SetSheetName(oldName, newName); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("rename: %v", err)}
	}

	s.mu.Lock()
	state.Sheets = s.buildSheetList(state.file)
	state.IsDirty = true
	if state.ActiveSheet == oldName {
		state.ActiveSheet = newName
	}
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// MergeCells merges a range of cells.
func (s *Service) MergeCells(tabID, sheetName, startCell, endCell string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.MergeCell(sheetName, startCell, endCell); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("merge: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// GetMergedCells returns all merged cell ranges in a sheet.
func (s *Service) GetMergedCells(tabID, sheetName string) []MergeCell {
	s.mu.RLock()
	state, ok := s.openBooks[tabID]
	s.mu.RUnlock()

	if !ok || state.file == nil {
		return nil
	}

	merged, err := state.file.GetMergeCells(sheetName)
	if err != nil {
		return nil
	}

	result := make([]MergeCell, len(merged))
	for i, mc := range merged {
		result[i] = MergeCell{
			StartCell: mc.GetStartAxis(),
			EndCell:   mc.GetEndAxis(),
			Value:     mc.GetCellValue(),
		}
	}
	return result
}

// InsertRow inserts a row at the given position (1-based).
func (s *Service) InsertRow(tabID, sheetName string, row int) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.InsertRows(sheetName, row, 1); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("insert row: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// InsertCol inserts a column at the given position (1-based).
func (s *Service) InsertCol(tabID, sheetName string, col int) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	colName, _ := excelize.ColumnNumberToName(col)
	if err := state.file.InsertCols(sheetName, colName, 1); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("insert col: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// DeleteRow deletes a row at the given position (1-based).
func (s *Service) DeleteRow(tabID, sheetName string, row int) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.RemoveRow(sheetName, row); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("delete row: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// SetColumnWidth sets the width of a column range.
func (s *Service) SetColumnWidth(tabID, sheetName, startCol, endCol string, width float64) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.SetColWidth(sheetName, startCol, endCol, width); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("set width: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// SetRowHeight sets the height of a row.
func (s *Service) SetRowHeight(tabID, sheetName string, row int, height float64) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.SetRowHeight(sheetName, row, height); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("set height: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// SetAutoFilter sets an auto-filter on a range.
func (s *Service) SetAutoFilter(tabID, sheetName, rangeRef string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.AutoFilter(sheetName, rangeRef, nil); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("auto filter: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = true
	s.mu.Unlock()

	return SaveResult{Success: true}
}

// ExportCSV exports a sheet as CSV.
func (s *Service) ExportCSV(tabID, sheetName, outputPath string) SaveResult {
	s.mu.RLock()
	state, ok := s.openBooks[tabID]
	s.mu.RUnlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	rows, err := state.file.GetRows(sheetName)
	if err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("get rows: %v", err)}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("create file: %v", err)}
	}
	defer f.Close()

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				f.WriteString(",")
			}
			// Quote cells containing commas or quotes
			if containsSpecial(cell) {
				f.WriteString("\"" + escapeCSV(cell) + "\"")
			} else {
				f.WriteString(cell)
			}
		}
		f.WriteString("\n")
	}

	return SaveResult{Success: true, FilePath: outputPath}
}

func containsSpecial(s string) bool {
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			return true
		}
	}
	return false
}

func escapeCSV(s string) string {
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "\"\""
		} else {
			result += string(c)
		}
	}
	return result
}

// Save saves the workbook to disk.
func (s *Service) Save(tabID string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if state.FilePath == "" {
		return SaveResult{Success: false, Error: "no file path set — use SaveAs"}
	}

	if err := state.file.SaveAs(state.FilePath); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = false
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: state.FilePath}
}

// SaveAs saves the workbook to a new path.
func (s *Service) SaveAs(tabID, path string) SaveResult {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	s.mu.Unlock()

	if !ok || state.file == nil {
		return SaveResult{Success: false, Error: "workbook not found"}
	}

	if err := state.file.SaveAs(path); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save as: %v", err)}
	}

	s.mu.Lock()
	state.FilePath = path
	state.Title = filepath.Base(path)
	state.IsDirty = false
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: path}
}

// Close closes the workbook and releases resources.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	state, ok := s.openBooks[tabID]
	if ok {
		if state.file != nil {
			state.file.Close()
		}
		delete(s.openBooks, tabID)
	}
	s.mu.Unlock()
}
