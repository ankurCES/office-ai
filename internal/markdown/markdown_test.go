package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

func newTestSvc() *Service {
	return New(i18n.New(), projectstore.New(), nil)
}

func TestOpenFile(t *testing.T) {
	svc := newTestSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Hello\n\nBody"), 0644)

	result := svc.OpenFile("tab1", path)
	if result["success"] != true {
		t.Fatalf("OpenFile failed: %v", result["error"])
	}
	if result["content"] != "# Hello\n\nBody" {
		t.Errorf("content = %q", result["content"])
	}
	if result["title"] != "test.md" {
		t.Errorf("title = %q", result["title"])
	}
}

func TestNewBlank(t *testing.T) {
	svc := newTestSvc()
	result := svc.NewBlank("tab1")
	if result["success"] != true {
		t.Fatalf("NewBlank failed: %v", result)
	}
}

func TestUpdateContent(t *testing.T) {
	svc := newTestSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("old"), 0644)

	svc.OpenFile("tab1", path)
	svc.UpdateContent("tab1", "# New content")

	// Verify by saving and reading back
	res := svc.Save("tab1")
	if res["success"] != true {
		t.Fatalf("Save: %v", res["error"])
	}
	data, _ := os.ReadFile(path)
	if string(data) != "# New content" {
		t.Errorf("content = %q, want '# New content'", string(data))
	}
}

func TestSave(t *testing.T) {
	svc := newTestSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("original"), 0644)

	svc.OpenFile("tab1", path)
	svc.UpdateContent("tab1", "modified")
	res := svc.Save("tab1")
	if res["success"] != true {
		t.Fatalf("Save: %v", res["error"])
	}

	data, _ := os.ReadFile(path)
	if string(data) != "modified" {
		t.Errorf("saved = %q, want 'modified'", string(data))
	}
}

func TestSaveAs(t *testing.T) {
	svc := newTestSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("original"), 0644)
	svc.OpenFile("tab1", path)

	newPath := filepath.Join(dir, "copy.md")
	res := svc.SaveAs("tab1", newPath)
	if res["success"] != true {
		t.Fatalf("SaveAs: %v", res["error"])
	}

	data, _ := os.ReadFile(newPath)
	if string(data) != "original" {
		t.Errorf("content = %q", string(data))
	}
}

func TestClose(t *testing.T) {
	svc := newTestSvc()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("test"), 0644)

	svc.OpenFile("tab1", path)
	svc.Close("tab1")

	// Verify save fails after close
	res := svc.Save("tab1")
	if res["success"] == true {
		t.Error("Save should fail after close")
	}
}

func TestSaveNoPath(t *testing.T) {
	svc := newTestSvc()
	svc.NewBlank("tab1")

	res := svc.Save("tab1")
	if res["success"] == true {
		t.Error("Save with no path should fail")
	}
}
