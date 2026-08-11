// Package docs implements the document editor module with real OOXML docx parsing.
// Uses archive/zip + encoding/xml to parse/generate word/document.xml.
package docs

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// ── OOXML types ─────────────────────────────────────────────────────

// Document is the root <w:document> element.
type Document struct {
	XMLName xml.Name `xml:"document"`
	Body    Body     `xml:"body"`
}

// Body is <w:body>.
type Body struct {
	Paragraphs []Paragraph `xml:"p"`
	SectPr     *SectPr     `xml:"sectPr,omitempty"`
}

// Paragraph is <w:p>.
type Paragraph struct {
	PPr  *ParagraphProperties `xml:"pPr,omitempty"`
	Runs []Run                `xml:"r"`
}

// ParagraphProperties is <w:pPr>.
type ParagraphProperties struct {
	PStyle    *StringVal    `xml:"pStyle,omitempty"`
	Jc        *StringVal    `xml:"jc,omitempty"`
	NumPr     *NumPr        `xml:"numPr,omitempty"`
	Spacing   *Spacing      `xml:"spacing,omitempty"`
	Ind       *Indentation  `xml:"ind,omitempty"`
	KeepNext  *EmptyElement `xml:"keepNext,omitempty"`
	PageBreak *EmptyElement `xml:"pageBreakBefore,omitempty"`
}

// Run is <w:r>.
type Run struct {
	RPr  *RunProperties `xml:"rPr,omitempty"`
	Text []Text         `xml:"t"`
	Tab  *EmptyElement  `xml:"tab,omitempty"`
	Br   *EmptyElement  `xml:"br,omitempty"`
}

// RunProperties is <w:rPr>.
type RunProperties struct {
	Bold       *EmptyElement `xml:"b,omitempty"`
	Italic     *EmptyElement `xml:"i,omitempty"`
	Underline  *StringVal    `xml:"u,omitempty"`
	Strike     *EmptyElement `xml:"strike,omitempty"`
	FontSize   *StringVal    `xml:"sz,omitempty"`
	FontSizeCs *StringVal    `xml:"szCs,omitempty"`
	Color      *StringVal    `xml:"color,omitempty"`
	RFonts     *RFonts       `xml:"rFonts,omitempty"`
	Highlight  *StringVal    `xml:"highlight,omitempty"`
	VertAlign  *StringVal    `xml:"vertAlign,omitempty"`
}

// Text is <w:t>.
type Text struct {
	XMLName xml.Name `xml:"t"`
	Space   string   `xml:"space,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

// SectPr is <w:sectPr> (section properties).
type SectPr struct {
	PgSz    *PageSize    `xml:"pgSz,omitempty"`
	PgMar   *PageMargins `xml:"pgMar,omitempty"`
	Cols    *Columns     `xml:"cols,omitempty"`
	DocGrid *DocGrid     `xml:"docGrid,omitempty"`
}

type PageSize struct {
	W      string `xml:"w,attr,omitempty"`
	H      string `xml:"h,attr,omitempty"`
	Orient string `xml:"orient,attr,omitempty"`
}

type PageMargins struct {
	Top    string `xml:"top,attr,omitempty"`
	Right  string `xml:"right,attr,omitempty"`
	Bottom string `xml:"bottom,attr,omitempty"`
	Left   string `xml:"left,attr,omitempty"`
	Header string `xml:"header,attr,omitempty"`
	Footer string `xml:"footer,attr,omitempty"`
	Gutter string `xml:"gutter,attr,omitempty"`
}

type Columns struct {
	Space string `xml:"space,attr,omitempty"`
}

type DocGrid struct {
	LinePitch string `xml:"linePitch,attr,omitempty"`
}

type StringVal struct {
	Val string `xml:"val,attr,omitempty"`
}

type EmptyElement struct{}

type RFonts struct {
	Ascii    string `xml:"ascii,attr,omitempty"`
	HAnsi   string `xml:"hAnsi,attr,omitempty"`
	EastAsia string `xml:"eastAsia,attr,omitempty"`
	Cs       string `xml:"cs,attr,omitempty"`
}

type NumPr struct {
	ILvl  *StringVal `xml:"ilvl,omitempty"`
	NumId *StringVal `xml:"numId,omitempty"`
}

type Spacing struct {
	Before string `xml:"before,attr,omitempty"`
	After  string `xml:"after,attr,omitempty"`
	Line   string `xml:"line,attr,omitempty"`
}

type Indentation struct {
	Left      string `xml:"left,attr,omitempty"`
	Right     string `xml:"right,attr,omitempty"`
	FirstLine string `xml:"firstLine,attr,omitempty"`
	Hanging   string `xml:"hanging,attr,omitempty"`
}

// ── Application model ───────────────────────────────────────────────

// ParaInfo is the JSON-friendly paragraph representation exposed to the frontend.
type ParaInfo struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	Style     string `json:"style,omitempty"`
	Alignment string `json:"alignment,omitempty"`
	IsBold    bool   `json:"is_bold,omitempty"`
	IsItalic  bool   `json:"is_italic,omitempty"`
}

// DocState holds the in-memory state of an open document.
type DocState struct {
	FilePath   string     `json:"file_path"`
	IsDirty    bool       `json:"is_dirty"`
	Title      string     `json:"title"`
	WordCount  int        `json:"word_count"`
	CharCount  int        `json:"char_count"`
	PageCount  int        `json:"page_count"`
	Paragraphs []ParaInfo `json:"paragraphs,omitempty"`

	// Internal (not JSON-serialized to frontend)
	doc        *Document
	zipData    []byte            // original ZIP bytes for round-trip
	otherParts map[string][]byte // non-document.xml parts preserved for save
	history    []historyEntry
	historyPos int
	autosaveTimer *time.Timer
	lastSaved  time.Time
}

type historyEntry struct {
	description string
	paragraphs  []Paragraph // snapshot
}

// OpenFileResult describes the outcome of opening a document.
type OpenFileResult struct {
	Success    bool       `json:"success"`
	FilePath   string     `json:"file_path"`
	Title      string     `json:"title"`
	Error      string     `json:"error,omitempty"`
	WordCount  int        `json:"word_count"`
	CharCount  int        `json:"char_count"`
	PageCount  int        `json:"page_count"`
	Paragraphs []ParaInfo `json:"paragraphs,omitempty"`
}

// SaveResult describes the outcome of saving a document.
type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
}

// FindReplaceResult describes the outcome of a find/replace operation.
type FindReplaceResult struct {
	Count   int    `json:"count"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Service is the docs module service bound to Wails.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	store     *projectstore.Store
	agent     *agentcore.Loop
	mu        sync.RWMutex
	openDocs  map[string]*DocState
}

// New creates a new docs Service.
func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:  i18nSvc,
		store:    store,
		agent:    agent,
		openDocs: make(map[string]*DocState),
	}
}

// ── Parsing ─────────────────────────────────────────────────────────

// parseDocx reads a docx ZIP and extracts the document model.
func parseDocx(data []byte) (*Document, map[string][]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("not a valid ZIP: %w", err)
	}

	var docXML []byte
	otherParts := make(map[string][]byte)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		if f.Name == "word/document.xml" {
			docXML = content
		} else {
			otherParts[f.Name] = content
		}
	}

	if docXML == nil {
		return nil, nil, fmt.Errorf("word/document.xml not found in archive")
	}

	var doc Document
	if err := xml.Unmarshal(docXML, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse document.xml: %w", err)
	}

	return &doc, otherParts, nil
}

// extractParagraphs converts XML paragraphs to the frontend model.
func extractParagraphs(doc *Document) []ParaInfo {
	result := make([]ParaInfo, 0, len(doc.Body.Paragraphs))
	for i, p := range doc.Body.Paragraphs {
		info := ParaInfo{Index: i}

		// Extract text from all runs
		var texts []string
		for _, r := range p.Runs {
			for _, t := range r.Text {
				texts = append(texts, t.Value)
			}
			if r.Tab != nil {
				texts = append(texts, "\t")
			}
			if r.Br != nil {
				texts = append(texts, "\n")
			}
		}
		info.Text = strings.Join(texts, "")

		// Extract formatting
		if p.PPr != nil {
			if p.PPr.PStyle != nil {
				info.Style = p.PPr.PStyle.Val
			}
			if p.PPr.Jc != nil {
				info.Alignment = p.PPr.Jc.Val
			}
		}

		// Check if first run has bold/italic
		if len(p.Runs) > 0 && p.Runs[0].RPr != nil {
			info.IsBold = p.Runs[0].RPr.Bold != nil
			info.IsItalic = p.Runs[0].RPr.Italic != nil
		}

		result = append(result, info)
	}
	return result
}

// countWords counts words in the document.
func countWords(doc *Document) int {
	count := 0
	for _, p := range doc.Body.Paragraphs {
		for _, r := range p.Runs {
			for _, t := range r.Text {
				words := strings.Fields(t.Value)
				count += len(words)
			}
		}
	}
	return count
}

// countChars counts characters in the document.
func countChars(doc *Document) int {
	count := 0
	for _, p := range doc.Body.Paragraphs {
		for _, r := range p.Runs {
			for _, t := range r.Text {
				count += utf8.RuneCountInString(t.Value)
			}
		}
	}
	return count
}

// estimatePages estimates page count (rough: ~3000 chars per page).
func estimatePages(charCount int) int {
	if charCount == 0 {
		return 1
	}
	pages := charCount / 3000
	if charCount%3000 > 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	return pages
}

// ── File operations ─────────────────────────────────────────────────

// OpenFile opens a .docx file and returns its parsed state.
func (s *Service) OpenFile(tabID, path string) OpenFileResult {
	if path == "" {
		return OpenFileResult{Success: false, Error: "no file path provided"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return OpenFileResult{Success: false, Error: fmt.Sprintf("read file: %v", err)}
	}

	doc, otherParts, err := parseDocx(data)
	if err != nil {
		return OpenFileResult{Success: false, Error: fmt.Sprintf("parse docx: %v", err)}
	}

	paras := extractParagraphs(doc)
	wc := countWords(doc)
	cc := countChars(doc)
	pc := estimatePages(cc)

	state := &DocState{
		FilePath:   path,
		Title:      filepath.Base(path),
		WordCount:  wc,
		CharCount:  cc,
		PageCount:  pc,
		Paragraphs: paras,
		doc:        doc,
		zipData:    data,
		otherParts: otherParts,
		lastSaved:  time.Now(),
	}
	state.pushHistory("open")

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	return OpenFileResult{
		Success:    true,
		FilePath:   path,
		Title:      state.Title,
		WordCount:  wc,
		CharCount:  cc,
		PageCount:  pc,
		Paragraphs: paras,
	}
}

// NewBlank creates a new blank document with proper OOXML structure.
func (s *Service) NewBlank(tabID string) OpenFileResult {
	doc := &Document{
		Body: Body{
			Paragraphs: []Paragraph{
				{Runs: []Run{{Text: []Text{{Value: "", Space: "preserve"}}}}},
			},
			SectPr: &SectPr{
				PgSz:    &PageSize{W: "12240", H: "15840"},  // Letter size
				PgMar:   &PageMargins{Top: "1440", Right: "1440", Bottom: "1440", Left: "1440", Header: "720", Footer: "720", Gutter: "0"},
				Cols:    &Columns{Space: "720"},
				DocGrid: &DocGrid{LinePitch: "360"},
			},
		},
	}

	state := &DocState{
		Title:      "Untitled Document",
		WordCount:  0,
		CharCount:  0,
		PageCount:  1,
		Paragraphs: []ParaInfo{{Index: 0, Text: ""}},
		doc:        doc,
		otherParts: buildBlankDocxParts(),
	}
	state.pushHistory("new")

	s.mu.Lock()
	s.openDocs[tabID] = state
	s.mu.Unlock()

	return OpenFileResult{
		Success:    true,
		Title:      state.Title,
		WordCount:  0,
		CharCount:  0,
		PageCount:  1,
		Paragraphs: state.Paragraphs,
	}
}

// buildBlankDocxParts returns the minimal set of parts for a valid docx.
func buildBlankDocxParts() map[string][]byte {
	return map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`),
		"_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`),
		"word/_rels/document.xml.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`),
		"word/styles.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:rPr><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:pPr><w:keepNext/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="48"/><w:szCs w:val="48"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:pPr><w:keepNext/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="36"/><w:szCs w:val="36"/></w:rPr>
  </w:style>
</w:styles>`),
	}
}

// ── Save ────────────────────────────────────────────────────────────

// generateDocXML serializes the document model back to XML.
func generateDocXML(doc *Document) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	buf.WriteString(`<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:w10="urn:schemas-microsoft-com:office:word" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" xmlns:wpg="http://schemas.microsoft.com/office/word/2010/wordprocessingGroup" xmlns:wpi="http://schemas.microsoft.com/office/word/2010/wordprocessingInk" xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml" xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape" mc:Ignorable="w14 wp14">`)
	buf.WriteString("<w:body>")

	for _, p := range doc.Body.Paragraphs {
		buf.WriteString("<w:p>")
		if p.PPr != nil {
			writePPr(&buf, p.PPr)
		}
		for _, r := range p.Runs {
			buf.WriteString("<w:r>")
			if r.RPr != nil {
				writeRPr(&buf, r.RPr)
			}
			if r.Tab != nil {
				buf.WriteString("<w:tab/>")
			}
			if r.Br != nil {
				buf.WriteString("<w:br/>")
			}
			for _, t := range r.Text {
				space := ""
				if t.Space != "" {
					space = ` xml:space="` + t.Space + `"`
				}
				buf.WriteString("<w:t" + space + ">")
				xml.Escape(&buf, []byte(t.Value))
				buf.WriteString("</w:t>")
			}
			buf.WriteString("</w:r>")
		}
		buf.WriteString("</w:p>")
	}

	if doc.Body.SectPr != nil {
		writeSectPr(&buf, doc.Body.SectPr)
	}

	buf.WriteString("</w:body></w:document>")
	return buf.Bytes(), nil
}

func writePPr(buf *bytes.Buffer, ppr *ParagraphProperties) {
	buf.WriteString("<w:pPr>")
	if ppr.PStyle != nil {
		buf.WriteString(`<w:pStyle w:val="` + ppr.PStyle.Val + `"/>`)
	}
	if ppr.Jc != nil {
		buf.WriteString(`<w:jc w:val="` + ppr.Jc.Val + `"/>`)
	}
	if ppr.KeepNext != nil {
		buf.WriteString("<w:keepNext/>")
	}
	if ppr.PageBreak != nil {
		buf.WriteString("<w:pageBreakBefore/>")
	}
	if ppr.Spacing != nil {
		buf.WriteString("<w:spacing")
		if ppr.Spacing.Before != "" {
			buf.WriteString(` w:before="` + ppr.Spacing.Before + `"`)
		}
		if ppr.Spacing.After != "" {
			buf.WriteString(` w:after="` + ppr.Spacing.After + `"`)
		}
		if ppr.Spacing.Line != "" {
			buf.WriteString(` w:line="` + ppr.Spacing.Line + `"`)
		}
		buf.WriteString("/>")
	}
	if ppr.Ind != nil {
		buf.WriteString("<w:ind")
		if ppr.Ind.Left != "" {
			buf.WriteString(` w:left="` + ppr.Ind.Left + `"`)
		}
		if ppr.Ind.Right != "" {
			buf.WriteString(` w:right="` + ppr.Ind.Right + `"`)
		}
		if ppr.Ind.FirstLine != "" {
			buf.WriteString(` w:firstLine="` + ppr.Ind.FirstLine + `"`)
		}
		buf.WriteString("/>")
	}
	if ppr.NumPr != nil {
		buf.WriteString("<w:numPr>")
		if ppr.NumPr.ILvl != nil {
			buf.WriteString(`<w:ilvl w:val="` + ppr.NumPr.ILvl.Val + `"/>`)
		}
		if ppr.NumPr.NumId != nil {
			buf.WriteString(`<w:numId w:val="` + ppr.NumPr.NumId.Val + `"/>`)
		}
		buf.WriteString("</w:numPr>")
	}
	buf.WriteString("</w:pPr>")
}

func writeRPr(buf *bytes.Buffer, rpr *RunProperties) {
	buf.WriteString("<w:rPr>")
	if rpr.RFonts != nil {
		buf.WriteString("<w:rFonts")
		if rpr.RFonts.Ascii != "" {
			buf.WriteString(` w:ascii="` + rpr.RFonts.Ascii + `"`)
		}
		if rpr.RFonts.HAnsi != "" {
			buf.WriteString(` w:hAnsi="` + rpr.RFonts.HAnsi + `"`)
		}
		if rpr.RFonts.EastAsia != "" {
			buf.WriteString(` w:eastAsia="` + rpr.RFonts.EastAsia + `"`)
		}
		buf.WriteString("/>")
	}
	if rpr.Bold != nil {
		buf.WriteString("<w:b/>")
	}
	if rpr.Italic != nil {
		buf.WriteString("<w:i/>")
	}
	if rpr.Underline != nil {
		buf.WriteString(`<w:u w:val="` + rpr.Underline.Val + `"/>`)
	}
	if rpr.Strike != nil {
		buf.WriteString("<w:strike/>")
	}
	if rpr.FontSize != nil {
		buf.WriteString(`<w:sz w:val="` + rpr.FontSize.Val + `"/>`)
	}
	if rpr.FontSizeCs != nil {
		buf.WriteString(`<w:szCs w:val="` + rpr.FontSizeCs.Val + `"/>`)
	}
	if rpr.Color != nil {
		buf.WriteString(`<w:color w:val="` + rpr.Color.Val + `"/>`)
	}
	if rpr.Highlight != nil {
		buf.WriteString(`<w:highlight w:val="` + rpr.Highlight.Val + `"/>`)
	}
	buf.WriteString("</w:rPr>")
}

func writeSectPr(buf *bytes.Buffer, sp *SectPr) {
	buf.WriteString("<w:sectPr>")
	if sp.PgSz != nil {
		buf.WriteString("<w:pgSz")
		if sp.PgSz.W != "" {
			buf.WriteString(` w:w="` + sp.PgSz.W + `"`)
		}
		if sp.PgSz.H != "" {
			buf.WriteString(` w:h="` + sp.PgSz.H + `"`)
		}
		if sp.PgSz.Orient != "" {
			buf.WriteString(` w:orient="` + sp.PgSz.Orient + `"`)
		}
		buf.WriteString("/>")
	}
	if sp.PgMar != nil {
		buf.WriteString("<w:pgMar")
		if sp.PgMar.Top != "" {
			buf.WriteString(` w:top="` + sp.PgMar.Top + `"`)
		}
		if sp.PgMar.Right != "" {
			buf.WriteString(` w:right="` + sp.PgMar.Right + `"`)
		}
		if sp.PgMar.Bottom != "" {
			buf.WriteString(` w:bottom="` + sp.PgMar.Bottom + `"`)
		}
		if sp.PgMar.Left != "" {
			buf.WriteString(` w:left="` + sp.PgMar.Left + `"`)
		}
		buf.WriteString("/>")
	}
	buf.WriteString("</w:sectPr>")
}

// saveDocx writes the document model back to a docx ZIP file.
func saveDocx(doc *Document, otherParts map[string][]byte, path string) error {
	docXML, err := generateDocXML(doc)
	if err != nil {
		return fmt.Errorf("generate XML: %w", err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Write the document.xml part
	fw, err := w.Create("word/document.xml")
	if err != nil {
		return err
	}
	if _, err := fw.Write(docXML); err != nil {
		return err
	}

	// Write all other preserved parts
	for name, data := range otherParts {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(data); err != nil {
			return err
		}
	}

	if err := w.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// Save saves the document to its current path.
func (s *Service) Save(tabID string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok {
		return SaveResult{Success: false, Error: "document not found"}
	}
	if state.FilePath == "" {
		return SaveResult{Success: false, Error: "no file path - use SaveAs"}
	}

	if err := saveDocx(state.doc, state.otherParts, state.FilePath); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = false
	state.lastSaved = time.Now()
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: state.FilePath}
}

// SaveAs saves the document to a new path.
func (s *Service) SaveAs(tabID, path string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok {
		return SaveResult{Success: false, Error: "document not found"}
	}

	if err := saveDocx(state.doc, state.otherParts, path); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save: %v", err)}
	}

	s.mu.Lock()
	state.FilePath = path
	state.Title = filepath.Base(path)
	state.IsDirty = false
	state.lastSaved = time.Now()
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: path}
}

// ── Editing operations ──────────────────────────────────────────────

// UpdateParagraph updates the text and formatting of a paragraph.
func (s *Service) UpdateParagraph(tabID string, index int, text string, bold, italic bool, alignment string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil {
		return false
	}

	// Extend paragraphs if needed
	for len(state.doc.Body.Paragraphs) <= index {
		state.doc.Body.Paragraphs = append(state.doc.Body.Paragraphs, Paragraph{
			Runs: []Run{{Text: []Text{{Value: "", Space: "preserve"}}}},
		})
	}

	p := &state.doc.Body.Paragraphs[index]

	// Update text
	if len(p.Runs) == 0 {
		p.Runs = []Run{{Text: []Text{{Value: text, Space: "preserve"}}}}
	} else {
		p.Runs = []Run{{
			RPr:  p.Runs[0].RPr,
			Text: []Text{{Value: text, Space: "preserve"}},
		}}
	}

	// Update formatting
	if p.Runs[0].RPr == nil {
		p.Runs[0].RPr = &RunProperties{}
	}
	if bold {
		p.Runs[0].RPr.Bold = &EmptyElement{}
	} else {
		p.Runs[0].RPr.Bold = nil
	}
	if italic {
		p.Runs[0].RPr.Italic = &EmptyElement{}
	} else {
		p.Runs[0].RPr.Italic = nil
	}

	if alignment != "" {
		if p.PPr == nil {
			p.PPr = &ParagraphProperties{}
		}
		p.PPr.Jc = &StringVal{Val: alignment}
	}

	state.IsDirty = true
	s.refreshCounts(state)
	state.pushHistory("edit paragraph")
	return true
}

// InsertParagraph inserts a new paragraph at the given index.
func (s *Service) InsertParagraph(tabID string, index int, text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil {
		return false
	}

	newPara := Paragraph{
		Runs: []Run{{Text: []Text{{Value: text, Space: "preserve"}}}},
	}

	paras := state.doc.Body.Paragraphs
	if index >= len(paras) {
		state.doc.Body.Paragraphs = append(paras, newPara)
	} else {
		state.doc.Body.Paragraphs = append(paras[:index+1], paras[index:]...)
		state.doc.Body.Paragraphs[index] = newPara
	}

	state.IsDirty = true
	s.refreshCounts(state)
	state.pushHistory("insert paragraph")
	return true
}

// DeleteParagraph removes a paragraph at the given index.
func (s *Service) DeleteParagraph(tabID string, index int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil || index < 0 || index >= len(state.doc.Body.Paragraphs) {
		return false
	}

	state.doc.Body.Paragraphs = append(
		state.doc.Body.Paragraphs[:index],
		state.doc.Body.Paragraphs[index+1:]...,
	)

	state.IsDirty = true
	s.refreshCounts(state)
	state.pushHistory("delete paragraph")
	return true
}

// FindReplace performs find and replace across the entire document.
func (s *Service) FindReplace(tabID, search, replace string, replaceAll bool) FindReplaceResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil {
		return FindReplaceResult{Success: false, Error: "document not found"}
	}

	count := 0
	for i := range state.doc.Body.Paragraphs {
		p := &state.doc.Body.Paragraphs[i]
		for j := range p.Runs {
			for k := range p.Runs[j].Text {
				t := &p.Runs[j].Text[k]
				if strings.Contains(t.Value, search) {
					if replaceAll {
						old := t.Value
						t.Value = strings.ReplaceAll(t.Value, search, replace)
						count += strings.Count(old, search)
					} else if count == 0 {
						t.Value = strings.Replace(t.Value, search, replace, 1)
						count = 1
					}
				}
			}
		}
	}

	if count > 0 {
		state.IsDirty = true
		s.refreshCounts(state)
		state.pushHistory(fmt.Sprintf("replace '%s' → '%s' (%d)", search, replace, count))
	}

	return FindReplaceResult{Count: count, Success: true}
}

// Find returns all paragraph indices containing the search term.
func (s *Service) Find(tabID, search string) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil {
		return nil
	}

	var results []int
	searchLower := strings.ToLower(search)
	for i, p := range state.doc.Body.Paragraphs {
		for _, r := range p.Runs {
			for _, t := range r.Text {
				if strings.Contains(strings.ToLower(t.Value), searchLower) {
					results = append(results, i)
					break
				}
			}
		}
	}
	return results
}

// ── Undo/Redo ───────────────────────────────────────────────────────

func (state *DocState) pushHistory(desc string) {
	// Truncate any redo future
	if state.historyPos < len(state.history) {
		state.history = state.history[:state.historyPos]
	}
	// Snapshot current paragraphs
	snap := make([]Paragraph, len(state.doc.Body.Paragraphs))
	copy(snap, state.doc.Body.Paragraphs)
	state.history = append(state.history, historyEntry{
		description: desc,
		paragraphs:  snap,
	})
	state.historyPos = len(state.history)

	// Cap history at 100
	if len(state.history) > 100 {
		state.history = state.history[len(state.history)-100:]
		state.historyPos = len(state.history)
	}
}

// Undo reverts to the previous state.
func (s *Service) Undo(tabID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil || state.historyPos <= 1 {
		return false
	}

	state.historyPos--
	entry := state.history[state.historyPos-1]
	state.doc.Body.Paragraphs = make([]Paragraph, len(entry.paragraphs))
	copy(state.doc.Body.Paragraphs, entry.paragraphs)
	state.IsDirty = true
	s.refreshCounts(state)
	return true
}

// Redo re-applies the next state.
func (s *Service) Redo(tabID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok || state.doc == nil || state.historyPos >= len(state.history) {
		return false
	}

	entry := state.history[state.historyPos]
	state.doc.Body.Paragraphs = make([]Paragraph, len(entry.paragraphs))
	copy(state.doc.Body.Paragraphs, entry.paragraphs)
	state.historyPos++
	state.IsDirty = true
	s.refreshCounts(state)
	return true
}

// ── Autosave ────────────────────────────────────────────────────────

// StartAutosave begins periodic autosaving for a document.
func (s *Service) StartAutosave(tabID string, intervalMs int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDocs[tabID]
	if !ok {
		return
	}

	if state.autosaveTimer != nil {
		state.autosaveTimer.Stop()
	}

	if intervalMs <= 0 {
		intervalMs = 2000
	}

	state.autosaveTimer = time.AfterFunc(time.Duration(intervalMs)*time.Millisecond, func() {
		s.doAutosave(tabID)
	})
}

func (s *Service) doAutosave(tabID string) {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok || !state.IsDirty || state.doc == nil {
		// Reschedule
		s.mu.Lock()
		if st, ok := s.openDocs[tabID]; ok && st.autosaveTimer != nil {
			st.autosaveTimer.Reset(2 * time.Second)
		}
		s.mu.Unlock()
		return
	}

	// Save to autosave file
	autosavePath := state.FilePath
	if autosavePath == "" {
		dir := os.TempDir()
		autosavePath = filepath.Join(dir, fmt.Sprintf("office-ai-autosave-%s.docx", tabID))
	} else {
		dir := filepath.Dir(autosavePath)
		base := filepath.Base(autosavePath)
		autosavePath = filepath.Join(dir, ".~"+base)
	}

	_ = saveDocx(state.doc, state.otherParts, autosavePath)

	// Reschedule
	s.mu.Lock()
	if st, ok := s.openDocs[tabID]; ok && st.autosaveTimer != nil {
		st.autosaveTimer.Reset(2 * time.Second)
	}
	s.mu.Unlock()
}

// ── Query methods ───────────────────────────────────────────────────

func (s *Service) refreshCounts(state *DocState) {
	state.WordCount = countWords(state.doc)
	state.CharCount = countChars(state.doc)
	state.PageCount = estimatePages(state.CharCount)
	state.Paragraphs = extractParagraphs(state.doc)
}

// GetState returns the current state of an open document.
func (s *Service) GetState(tabID string) *DocState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.openDocs[tabID]
	if state != nil {
		// Return a copy without internal fields
		return &DocState{
			FilePath:   state.FilePath,
			IsDirty:    state.IsDirty,
			Title:      state.Title,
			WordCount:  state.WordCount,
			CharCount:  state.CharCount,
			PageCount:  state.PageCount,
			Paragraphs: state.Paragraphs,
		}
	}
	return nil
}

// IsDirty returns whether the document has unsaved changes.
func (s *Service) IsDirty(tabID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDocs[tabID]; ok {
		return state.IsDirty
	}
	return false
}

// Close closes an open document and cleans up resources.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.openDocs[tabID]; ok {
		if state.autosaveTimer != nil {
			state.autosaveTimer.Stop()
		}
	}
	delete(s.openDocs, tabID)
}

// GetParagraphs returns all paragraphs for the frontend.
func (s *Service) GetParagraphs(tabID string) []ParaInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDocs[tabID]; ok {
		return state.Paragraphs
	}
	return nil
}

// ── AI Skill ────────────────────────────────────────────────────────

// GetDocsSkill returns the agent skill for document operations.
func (s *Service) GetDocsSkill() *agentcore.Skill {
	return &agentcore.Skill{
		ID: "docs",
		SystemPrompt: `You are an AI assistant for document editing. You can:
- Insert, replace, and delete text in the document
- Format paragraphs (alignment, spacing, indentation)
- Insert tables, images, and other elements
- Apply styles and themes
- Manage headers, footers, and page numbering
Always describe what you changed after each edit.`,
		Tools: []agentcore.ToolDef{
			{
				Name:        "insert_text",
				Description: "Insert text at a paragraph index",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text":      map[string]interface{}{"type": "string"},
						"paragraph": map[string]interface{}{"type": "integer"},
					},
					"required": []string{"text"},
				},
			},
			{
				Name:        "replace_text",
				Description: "Replace text matching a search string",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search":  map[string]interface{}{"type": "string"},
						"replace": map[string]interface{}{"type": "string"},
					},
					"required": []string{"search", "replace"},
				},
			},
			{
				Name:        "format_paragraph",
				Description: "Apply formatting to a paragraph",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"paragraph": map[string]interface{}{"type": "integer"},
						"alignment": map[string]interface{}{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
						"bold":      map[string]interface{}{"type": "boolean"},
						"italic":    map[string]interface{}{"type": "boolean"},
					},
					"required": []string{"paragraph"},
				},
			},
			{
				Name:        "delete_paragraph",
				Description: "Delete a paragraph by index",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"paragraph": map[string]interface{}{"type": "integer"},
					},
					"required": []string{"paragraph"},
				},
			},
			{
				Name:        "find_replace",
				Description: "Find and replace text across the document",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search":      map[string]interface{}{"type": "string"},
						"replace":     map[string]interface{}{"type": "string"},
						"replace_all": map[string]interface{}{"type": "boolean"},
					},
					"required": []string{"search", "replace"},
				},
			},
		},
	}
}
