package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectKind(t *testing.T) {
	tests := []struct {
		path string
		want FileKind
	}{
		{"document.docx", KindDocs},
		{"data.xlsx", KindSheets},
		{"slides.pptx", KindSlides},
		{"report.pdf", KindPDF},
		{"readme.md", KindMarkdown},
		{"notes.txt", KindDocs}, // .txt maps to docs
		{"image.png", KindUnknown},
		{"archive.zip", KindUnknown},
		{"data.csv", KindSheets},
		{"doc.odt", KindDocs},
		{"doc.rtf", KindDocs},
		{"slides.odp", KindSlides},
		{"notes.mdx", KindMarkdown},
	}

	for _, tc := range tests {
		got := DetectKind(tc.path)
		if got != tc.want {
			t.Errorf("DetectKind(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestStat(t *testing.T) {
	// Create a temp file
	tmp := filepath.Join(t.TempDir(), "test.docx")
	os.WriteFile(tmp, []byte("hello"), 0644)

	info, err := Stat(tmp)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Kind != KindDocs {
		t.Errorf("expected kind=docs, got %q", info.Kind)
	}
	if info.Size != 5 {
		t.Errorf("expected size=5, got %d", info.Size)
	}
	if info.Name != "test.docx" {
		t.Errorf("expected name=test.docx, got %q", info.Name)
	}
}

func TestReadWriteText(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	content := "Hello, World!"

	err := WriteText(tmp, content)
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	got, err := ReadText(tmp)
	if err != nil {
		t.Fatalf("ReadText failed: %v", err)
	}
	if got != content {
		t.Errorf("expected %q, got %q", content, got)
	}
}

func TestSafeWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "safe.txt")

	err := SafeWrite(path, []byte("safe content"))
	if err != nil {
		t.Fatalf("SafeWrite failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back failed: %v", err)
	}
	if string(data) != "safe content" {
		t.Errorf("expected 'safe content', got %q", string(data))
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	os.WriteFile(src, []byte("copy me"), 0644)

	err := CopyFile(src, dst)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	data, _ := os.ReadFile(dst)
	if string(data) != "copy me" {
		t.Errorf("expected 'copy me', got %q", string(data))
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash.txt")
	os.WriteFile(path, []byte("checksum test"), 0644)

	hash, err := Hash(path)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if hash == "" {
		t.Error("Hash returned empty")
	}
	if len(hash) != 64 { // SHA-256 hex
		t.Errorf("expected 64 char hex, got %d chars", len(hash))
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/doc.docx", "doc"},
		{"simple.txt", "simple"},
		{"noext", "noext"},
		{"/path/to/readme.md", "readme"},
	}

	for _, tc := range tests {
		got := BaseName(tc.path)
		if got != tc.want {
			t.Errorf("BaseName(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "c")
	err := EnsureDir(dir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestTempDir(t *testing.T) {
	dir, err := TempDir("test-prefix")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(dir)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("temp dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestFileFilters(t *testing.T) {
	filters := FiltersForKind(KindDocs)
	if len(filters) == 0 {
		t.Error("GetFilters returned empty for docs")
	}

	found := false
	for _, f := range filters {
		if f.DisplayName != "" && len(f.Extensions) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("no valid filter found")
	}
}
