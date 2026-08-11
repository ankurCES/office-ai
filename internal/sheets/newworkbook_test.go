package sheets

import (
	"testing"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

func TestNewWorkbookFlow(t *testing.T) {
	svc := New(i18n.New(), projectstore.New(), agentcore.New(nil))
	result := svc.NewWorkbook("test-wb-1")
	if result["success"] != true {
		t.Fatalf("NewWorkbook failed: %v", result)
	}
	t.Logf("NewWorkbook: success=%v title=%v", result["success"], result["title"])

	// Set a cell value
	setResult := svc.SetCellValue("test-wb-1", "Sheet1", "A1", "Hello")
	if !setResult.Success {
		t.Fatalf("SetCellValue failed: %s", setResult.Error)
	}

	// Get cell range
	cells := svc.GetCellRange("test-wb-1", "Sheet1", 0, 0, 5, 5)
	if len(cells) == 0 {
		t.Fatal("GetCellRange returned empty")
	}
	t.Logf("Cell A1 value: %v", cells[0][0].Value)

	// Save
	path := t.TempDir() + "/test.xlsx"
	saveResult := svc.SaveAs("test-wb-1", path)
	if !saveResult.Success {
		t.Fatalf("SaveAs failed: %s", saveResult.Error)
	}
	t.Logf("Saved to: %s", saveResult.FilePath)
}
