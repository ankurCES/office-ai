package docs

import (
	"testing"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

func TestNewBlankFlow(t *testing.T) {
	svc := New(i18n.New(), projectstore.New(), agentcore.New(nil))
	result := svc.NewBlank("test-tab-1")
	if !result.Success {
		t.Fatalf("NewBlank failed: success=false")
	}
	if result.Title == "" {
		t.Fatal("NewBlank returned empty title")
	}
	t.Logf("NewBlank: success=%v title=%q paragraphs=%d", result.Success, result.Title, len(result.Paragraphs))

	// Verify GetState works after NewBlank
	state := svc.GetState("test-tab-1")
	if state == nil {
		t.Fatal("GetState returned nil after NewBlank")
	}
	t.Logf("GetState: title=%q wordCount=%d pageCount=%d", state.Title, state.WordCount, state.PageCount)

	// Verify InsertParagraph works
	ok := svc.InsertParagraph("test-tab-1", 0, "Hello World")
	if !ok {
		t.Fatal("InsertParagraph failed")
	}

	// Verify state updated
	state2 := svc.GetState("test-tab-1")
	t.Logf("WordCount after insert: %d", state2.WordCount)
}

func TestNewBlankAndSave(t *testing.T) {
	svc := New(i18n.New(), projectstore.New(), agentcore.New(nil))
	result := svc.NewBlank("save-tab")
	if !result.Success {
		t.Fatalf("NewBlank failed")
	}

	// SaveAs to temp file
	path := t.TempDir() + "/test.docx"
	saveResult := svc.SaveAs("save-tab", path)
	if !saveResult.Success {
		t.Fatalf("SaveAs failed: %s", saveResult.Error)
	}
	t.Logf("Saved to: %s", saveResult.FilePath)
}
