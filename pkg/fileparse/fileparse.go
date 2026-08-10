// Package fileparse extracts plain text from office documents,
// mirroring GenOffice's file-parse package.
package fileparse

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ParsedFileKind identifies the document type.
type ParsedFileKind string

const (
	KindDocx     ParsedFileKind = "docx"
	KindXlsx     ParsedFileKind = "xlsx"
	KindPptx     ParsedFileKind = "pptx"
	KindPDF      ParsedFileKind = "pdf"
	KindMarkdown ParsedFileKind = "markdown"
	KindText     ParsedFileKind = "text"
	KindUnknown  ParsedFileKind = "unknown"
)

// ParsedFile holds the extracted text and metadata.
type ParsedFile struct {
	Kind ParsedFileKind `json:"kind"`
	Text string         `json:"text"`
	Name string         `json:"name"`
}

// ParseFileToText extracts text from a file based on its extension.
func ParseFileToText(path string) (*ParsedFile, error) {
	ext := strings.ToLower(filepath.Ext(path))
	name := filepath.Base(path)

	switch ext {
	case ".docx":
		text, err := DocxToText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindDocx, Text: text, Name: name}, nil
	case ".xlsx":
		text, err := XlsxToText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindXlsx, Text: text, Name: name}, nil
	case ".pptx":
		text, err := PptxToText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindPptx, Text: text, Name: name}, nil
	case ".pdf":
		text, err := PdfToText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindPDF, Text: text, Name: name}, nil
	case ".md", ".markdown":
		text, err := readFileText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindMarkdown, Text: text, Name: name}, nil
	case ".txt", ".csv", ".tsv", ".log":
		text, err := readFileText(path)
		if err != nil {
			return nil, err
		}
		return &ParsedFile{Kind: KindText, Text: text, Name: name}, nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// DocxToText extracts text from a .docx file by reading word/document.xml.
func DocxToText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			return extractXMLText(rc)
		}
	}
	return "", fmt.Errorf("word/document.xml not found in docx")
}

// PptxToText extracts text from all slides in a .pptx file.
func PptxToText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	var texts []string
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			text, err := extractXMLText(rc)
			rc.Close()
			if err == nil && text != "" {
				texts = append(texts, text)
			}
		}
	}
	return strings.Join(texts, "\n\n"), nil
}

// XlsxToText extracts text from shared strings in a .xlsx file.
func XlsxToText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			return extractXMLText(rc)
		}
	}
	return "", nil // xlsx may have no shared strings
}

// PdfToText is a placeholder — real implementation would use pdfcpu or pdfium.
func PdfToText(path string) (string, error) {
	return fmt.Sprintf("[PDF text extraction from %s - pending pdfcpu integration]", filepath.Base(path)), nil
}

// extractXMLText walks XML tokens and concatenates all character data.
func extractXMLText(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var parts []string
	var inText bool

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return strings.Join(parts, " "), nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" || t.Name.Local == "r" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				parts = append(parts, "\n")
			}
			inText = false
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && inText {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, " ")), nil
}

func readFileText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
