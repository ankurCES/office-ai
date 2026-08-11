package sheets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

func newTestService() *Service {
	i := i18n.New()
	st := projectstore.New()
	return New(i, st, nil)
}

func createTestXlsx(t *testing.T) string {
	t.Helper()
	svc := newTestService()
	result := svc.NewWorkbook("test-new")
	if result["success"] != true {
		t.Fatalf("NewWorkbook failed: %v", result["error"])
	}

	// Set some cell values
	svc.SetCellValue("test-new", "Sheet1", "A1", "Name")
	svc.SetCellValue("test-new", "Sheet1", "B1", "Value")
	svc.SetCellValue("test-new", "Sheet1", "A2", "Alpha")
	svc.SetCellValue("test-new", "Sheet1", "B2", "42")

	// Save to temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.xlsx")
	res := svc.SaveAs("test-new", path)
	if !res.Success {
		t.Fatalf("SaveAs failed: %s", res.Error)
	}
	svc.Close("test-new")
	return path
}

func TestNewWorkbook(t *testing.T) {
	svc := newTestService()
	result := svc.NewWorkbook("tab1")
	if result["success"] != true {
		t.Fatalf("NewWorkbook failed: %v", result)
	}
	sheets := result["sheets"].([]WorksheetInfo)
	if len(sheets) != 1 {
		t.Errorf("sheet count = %d, want 1", len(sheets))
	}
	if sheets[0].Name != "Sheet1" {
		t.Errorf("sheet name = %q, want Sheet1", sheets[0].Name)
	}
	svc.Close("tab1")
}

func TestOpenFile(t *testing.T) {
	path := createTestXlsx(t)

	svc := newTestService()
	result := svc.OpenFile("tab1", path)
	if result["success"] != true {
		t.Fatalf("OpenFile failed: %v", result["error"])
	}
	if result["title"] != "test.xlsx" {
		t.Errorf("title = %q, want test.xlsx", result["title"])
	}
	svc.Close("tab1")
}

func TestGetCellRange(t *testing.T) {
	path := createTestXlsx(t)

	svc := newTestService()
	svc.OpenFile("tab1", path)

	cells := svc.GetCellRange("tab1", "Sheet1", 0, 0, 2, 2)
	if len(cells) != 2 || len(cells[0]) != 2 {
		t.Fatalf("cell range size = %dx%d, want 2x2", len(cells), len(cells[0]))
	}
	if cells[0][0].Value != "Name" {
		t.Errorf("A1 = %q, want Name", cells[0][0].Value)
	}
	if cells[1][1].Value != "42" {
		t.Errorf("B2 = %q, want 42", cells[1][1].Value)
	}
	if cells[1][1].Type != "number" {
		t.Errorf("B2 type = %q, want number", cells[1][1].Type)
	}
	svc.Close("tab1")
}

func TestSetCellFormula(t *testing.T) {
	svc := newTestService()
	svc.NewWorkbook("tab1")
	svc.SetCellValue("tab1", "Sheet1", "A1", "10")
	svc.SetCellValue("tab1", "Sheet1", "A2", "20")

	res := svc.SetCellFormula("tab1", "Sheet1", "A3", "SUM(A1:A2)")
	if !res.Success {
		t.Fatalf("SetCellFormula: %s", res.Error)
	}

	cells := svc.GetCellRange("tab1", "Sheet1", 2, 0, 1, 1)
	if cells[0][0].Formula != "SUM(A1:A2)" {
		t.Errorf("formula = %q, want SUM(A1:A2)", cells[0][0].Formula)
	}
	svc.Close("tab1")
}

func TestAddDeleteSheet(t *testing.T) {
	svc := newTestService()
	svc.NewWorkbook("tab1")

	res := svc.AddSheet("tab1", "Data")
	if !res.Success {
		t.Fatalf("AddSheet: %s", res.Error)
	}

	state := svc.GetState("tab1")
	if len(state.Sheets) != 2 {
		t.Errorf("sheet count = %d, want 2", len(state.Sheets))
	}

	res = svc.DeleteSheet("tab1", "Data")
	if !res.Success {
		t.Fatalf("DeleteSheet: %s", res.Error)
	}

	state = svc.GetState("tab1")
	if len(state.Sheets) != 1 {
		t.Errorf("sheet count after delete = %d, want 1", len(state.Sheets))
	}
	svc.Close("tab1")
}

func TestRenameSheet(t *testing.T) {
	svc := newTestService()
	svc.NewWorkbook("tab1")

	res := svc.RenameSheet("tab1", "Sheet1", "MyData")
	if !res.Success {
		t.Fatalf("RenameSheet: %s", res.Error)
	}

	state := svc.GetState("tab1")
	if state.Sheets[0].Name != "MyData" {
		t.Errorf("sheet name = %q, want MyData", state.Sheets[0].Name)
	}
	svc.Close("tab1")
}

func TestExportCSV(t *testing.T) {
	path := createTestXlsx(t)

	svc := newTestService()
	svc.OpenFile("tab1", path)

	csvPath := filepath.Join(t.TempDir(), "export.csv")
	res := svc.ExportCSV("tab1", "Sheet1", csvPath)
	if !res.Success {
		t.Fatalf("ExportCSV: %s", res.Error)
	}

	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("CSV is empty")
	}
	svc.Close("tab1")
}

func TestSaveAs(t *testing.T) {
	svc := newTestService()
	svc.NewWorkbook("tab1")
	svc.SetCellValue("tab1", "Sheet1", "A1", "test")

	path := filepath.Join(t.TempDir(), "saved.xlsx")
	res := svc.SaveAs("tab1", path)
	if !res.Success {
		t.Fatalf("SaveAs: %s", res.Error)
	}

	state := svc.GetState("tab1")
	if state.IsDirty {
		t.Error("IsDirty should be false after save")
	}
	if state.FilePath != path {
		t.Errorf("FilePath = %q, want %q", state.FilePath, path)
	}
	svc.Close("tab1")
}

func TestCloseNonExistent(t *testing.T) {
	svc := newTestService()
	// Should not panic
	svc.Close("nonexistent")
}

func TestGetStateNil(t *testing.T) {
	svc := newTestService()
	state := svc.GetState("nonexistent")
	if state != nil {
		t.Error("GetState for nonexistent should return nil")
	}
}
