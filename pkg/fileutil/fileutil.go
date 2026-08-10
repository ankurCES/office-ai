// Package fileutil provides file I/O utilities mirroring GenOffice's file handling:
// native open/save dialogs, file type detection, temp dir management, and safe writes.
package fileutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileKind identifies a file type by extension.
type FileKind string

const (
	KindDocs     FileKind = "docs"
	KindSheets   FileKind = "sheets"
	KindSlides   FileKind = "slides"
	KindPDF      FileKind = "pdf"
	KindMarkdown FileKind = "markdown"
	KindUnknown  FileKind = "unknown"
)

// FileInfo holds metadata about a file.
type FileInfo struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Extension string    `json:"extension"`
	Kind      FileKind  `json:"kind"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	IsDir     bool      `json:"is_dir"`
}

// extension → kind map
var extKindMap = map[string]FileKind{
	".doc":  KindDocs,
	".docx": KindDocs,
	".odt":  KindDocs,
	".rtf":  KindDocs,
	".txt":  KindDocs,

	".xls":  KindSheets,
	".xlsx": KindSheets,
	".ods":  KindSheets,
	".csv":  KindSheets,
	".tsv":  KindSheets,

	".ppt":  KindSlides,
	".pptx": KindSlides,
	".odp":  KindSlides,

	".pdf": KindPDF,

	".md":       KindMarkdown,
	".markdown": KindMarkdown,
	".mdx":      KindMarkdown,
}

// DetectKind returns the FileKind for a path based on extension.
func DetectKind(path string) FileKind {
	ext := strings.ToLower(filepath.Ext(path))
	if kind, ok := extKindMap[ext]; ok {
		return kind
	}
	return KindUnknown
}

// Stat returns FileInfo for a path.
func Stat(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	return &FileInfo{
		Path:      path,
		Name:      filepath.Base(path),
		Extension: ext,
		Kind:      DetectKind(path),
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		IsDir:     info.IsDir(),
	}, nil
}

// ReadText reads a file as UTF-8 text. Returns content and error.
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteText writes text to a file, creating parent directories.
func WriteText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// SafeWrite writes atomically: write to temp file then rename.
func SafeWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".office-ai-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}

// CopyFile copies src to dst.
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Hash returns the SHA-256 hex hash of a file.
func Hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// TempDir creates a temporary directory for Office AI operations.
func TempDir(prefix string) (string, error) {
	return os.MkdirTemp("", "office-ai-"+prefix+"-")
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ListDir lists files in a directory, optionally filtered by kind.
func ListDir(dir string, kindFilter *FileKind) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []FileInfo
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		info, err := Stat(fullPath)
		if err != nil {
			continue
		}
		if kindFilter != nil && info.Kind != *kindFilter && !info.IsDir {
			continue
		}
		result = append(result, *info)
	}
	return result, nil
}

// FileFilter describes file types for open/save dialogs.
type FileFilter struct {
	DisplayName string   `json:"display_name"` // e.g. "Word Documents"
	Pattern     string   `json:"pattern"`       // e.g. "*.docx;*.doc"
	Extensions  []string `json:"extensions"`
}

// FiltersForKind returns dialog filters for a FileKind.
func FiltersForKind(kind FileKind) []FileFilter {
	switch kind {
	case KindDocs:
		return []FileFilter{
			{DisplayName: "Word Documents", Pattern: "*.docx;*.doc", Extensions: []string{"docx", "doc"}},
			{DisplayName: "OpenDocument", Pattern: "*.odt", Extensions: []string{"odt"}},
			{DisplayName: "Rich Text", Pattern: "*.rtf", Extensions: []string{"rtf"}},
			{DisplayName: "Text Files", Pattern: "*.txt", Extensions: []string{"txt"}},
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	case KindSheets:
		return []FileFilter{
			{DisplayName: "Excel Files", Pattern: "*.xlsx;*.xls", Extensions: []string{"xlsx", "xls"}},
			{DisplayName: "OpenDocument Spreadsheet", Pattern: "*.ods", Extensions: []string{"ods"}},
			{DisplayName: "CSV Files", Pattern: "*.csv", Extensions: []string{"csv"}},
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	case KindSlides:
		return []FileFilter{
			{DisplayName: "PowerPoint Files", Pattern: "*.pptx;*.ppt", Extensions: []string{"pptx", "ppt"}},
			{DisplayName: "OpenDocument Presentation", Pattern: "*.odp", Extensions: []string{"odp"}},
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	case KindPDF:
		return []FileFilter{
			{DisplayName: "PDF Files", Pattern: "*.pdf", Extensions: []string{"pdf"}},
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	case KindMarkdown:
		return []FileFilter{
			{DisplayName: "Markdown Files", Pattern: "*.md;*.markdown;*.mdx", Extensions: []string{"md", "markdown", "mdx"}},
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	default:
		return []FileFilter{
			{DisplayName: "All Files", Pattern: "*.*", Extensions: []string{"*"}},
		}
	}
}

// Exists returns true if the path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// BaseName returns the filename without extension.
func BaseName(path string) string {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	if ext != "" {
		name = name[:len(name)-len(ext)]
	}
	return name
}
