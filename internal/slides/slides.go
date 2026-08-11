// Package slides implements the presentation editor with real OOXML pptx parsing.
package slides

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// ── OOXML types ─────────────────────────────────────────────────────

type Presentation struct {
	XMLName xml.Name      `xml:"presentation"`
	SldSz   *SlideSize    `xml:"sldSz,omitempty"`
	SldIdLst *SlideIdList `xml:"sldIdLst,omitempty"`
}

type SlideSize struct {
	Cx string `xml:"cx,attr,omitempty"`
	Cy string `xml:"cy,attr,omitempty"`
}

type SlideIdList struct {
	SldIds []SlideId `xml:"sldId"`
}

type SlideId struct {
	Id   string `xml:"id,attr,omitempty"`
	RId  string `xml:"r:id,attr,omitempty"`
}

type SlideXML struct {
	XMLName xml.Name `xml:"sld"`
	CSld    CSld     `xml:"cSld"`
}

type CSld struct {
	SpTree SpTree `xml:"spTree"`
}

type SpTree struct {
	Shapes []Shape `xml:"sp"`
	Pics   []Pic   `xml:"pic"`
}

type Shape struct {
	NvSpPr NvSpPr `xml:"nvSpPr"`
	SpPr   SpPr   `xml:"spPr"`
	TxBody *TxBody `xml:"txBody,omitempty"`
}

type NvSpPr struct {
	CNvPr  CNvPr  `xml:"cNvPr"`
	NvPr   *NvPr  `xml:"nvPr,omitempty"`
}

type CNvPr struct {
	Id   string `xml:"id,attr,omitempty"`
	Name string `xml:"name,attr,omitempty"`
}

type NvPr struct {
	Ph *Placeholder `xml:"ph,omitempty"`
}

type Placeholder struct {
	Type string `xml:"type,attr,omitempty"`
	Idx  string `xml:"idx,attr,omitempty"`
}

type SpPr struct {
	Xfrm *Transform `xml:"xfrm,omitempty"`
}

type Transform struct {
	Off Offset `xml:"off"`
	Ext Extent `xml:"ext"`
}

type Offset struct {
	X string `xml:"x,attr,omitempty"`
	Y string `xml:"y,attr,omitempty"`
}

type Extent struct {
	Cx string `xml:"cx,attr,omitempty"`
	Cy string `xml:"cy,attr,omitempty"`
}

type TxBody struct {
	Paragraphs []TxParagraph `xml:"p"`
}

type TxParagraph struct {
	PPr  *TxPPr  `xml:"pPr,omitempty"`
	Runs []TxRun `xml:"r"`
}

type TxPPr struct {
	Algn string `xml:"algn,attr,omitempty"`
}

type TxRun struct {
	RPr  *TxRPr `xml:"rPr,omitempty"`
	Text string `xml:"t"`
}

type TxRPr struct {
	Lang string `xml:"lang,attr,omitempty"`
	Sz   string `xml:"sz,attr,omitempty"`
	B    string `xml:"b,attr,omitempty"`
	I    string `xml:"i,attr,omitempty"`
}

type Pic struct {
	NvPicPr NvPicPr `xml:"nvPicPr"`
	BlipFill *BlipFill `xml:"blipFill,omitempty"`
	SpPr    SpPr    `xml:"spPr"`
}

type NvPicPr struct {
	CNvPr CNvPr `xml:"cNvPr"`
}

type BlipFill struct {
	Blip *Blip `xml:"blip,omitempty"`
}

type Blip struct {
	Embed string `xml:"embed,attr,omitempty"`
}

// ── Application model ───────────────────────────────────────────────

// SlideElement is a JSON-friendly element on a slide.
type SlideElement struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "text", "image", "shape"
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"w"`
	H       int    `json:"h"`
	Text    string `json:"text,omitempty"`
	Bold    bool   `json:"bold,omitempty"`
	Italic  bool   `json:"italic,omitempty"`
	Align   string `json:"align,omitempty"`
	PhType  string `json:"ph_type,omitempty"` // placeholder type: title, body, etc.
	ImageID string `json:"image_id,omitempty"`
}

// SlideInfo is a JSON-friendly slide representation.
type SlideInfo struct {
	Index    int            `json:"index"`
	Elements []SlideElement `json:"elements"`
}

// DeckState holds the in-memory state of an open presentation.
type DeckState struct {
	FilePath   string      `json:"file_path"`
	IsDirty    bool        `json:"is_dirty"`
	Title      string      `json:"title"`
	SlideCount int         `json:"slide_count"`
	Slides     []SlideInfo `json:"slides"`
	Width      int         `json:"width"`  // in pixels (scaled from EMU)
	Height     int         `json:"height"` // in pixels

	// Internal
	slideXMLs  map[string][]byte   // "ppt/slides/slide1.xml" → raw XML
	otherParts map[string][]byte   // everything else in the ZIP
	history    []deckSnapshot
	historyPos int
}

type deckSnapshot struct {
	desc   string
	slides []SlideInfo
}

type OpenResult struct {
	Success    bool        `json:"success"`
	FilePath   string      `json:"file_path,omitempty"`
	Title      string      `json:"title"`
	Error      string      `json:"error,omitempty"`
	SlideCount int         `json:"slide_count"`
	Slides     []SlideInfo `json:"slides,omitempty"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
}

type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Service is the slides module bound to Wails.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	store     *projectstore.Store
	agent     *agentcore.Loop
	mu        sync.RWMutex
	openDecks map[string]*DeckState
}

func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:   i18nSvc,
		store:     store,
		agent:     agent,
		openDecks: make(map[string]*DeckState),
	}
}

// EMU to pixel conversion (96 DPI: 1 inch = 914400 EMU = 96 px)
const emuPerPx = 914400 / 96

func emuToPixels(emu string) int {
	v, _ := strconv.Atoi(emu)
	if v == 0 {
		return 0
	}
	return v / emuPerPx
}

// ── Parsing ─────────────────────────────────────────────────────────

var slideFileRe = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

func parsePptx(data []byte) (*DeckState, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a valid ZIP: %w", err)
	}

	state := &DeckState{
		slideXMLs:  make(map[string][]byte),
		otherParts: make(map[string][]byte),
		Width:      960,  // default 10" @ 96dpi
		Height:     540,  // default 7.5" @ 96dpi
	}

	var presXML []byte

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

		if slideFileRe.MatchString(f.Name) {
			state.slideXMLs[f.Name] = content
		} else if f.Name == "ppt/presentation.xml" {
			presXML = content
			state.otherParts[f.Name] = content
		} else {
			state.otherParts[f.Name] = content
		}
	}

	// Parse presentation.xml for slide size
	if presXML != nil {
		var pres Presentation
		if err := xml.Unmarshal(presXML, &pres); err == nil {
			if pres.SldSz != nil {
				w := emuToPixels(pres.SldSz.Cx)
				h := emuToPixels(pres.SldSz.Cy)
				if w > 0 {
					state.Width = w
				}
				if h > 0 {
					state.Height = h
				}
			}
		}
	}

	// Parse each slide XML
	slideNames := make([]string, 0, len(state.slideXMLs))
	for name := range state.slideXMLs {
		slideNames = append(slideNames, name)
	}
	sort.Slice(slideNames, func(i, j int) bool {
		ni, _ := strconv.Atoi(slideFileRe.FindStringSubmatch(slideNames[i])[1])
		nj, _ := strconv.Atoi(slideFileRe.FindStringSubmatch(slideNames[j])[1])
		return ni < nj
	})

	for idx, name := range slideNames {
		slideInfo := parseSlideXML(state.slideXMLs[name], idx)
		state.Slides = append(state.Slides, slideInfo)
	}

	state.SlideCount = len(state.Slides)
	return state, nil
}

func parseSlideXML(data []byte, index int) SlideInfo {
	info := SlideInfo{Index: index}

	var slide SlideXML
	if err := xml.Unmarshal(data, &slide); err != nil {
		return info
	}

	elemID := 0

	// Parse shapes
	for _, sp := range slide.CSld.SpTree.Shapes {
		elem := SlideElement{
			ID:   fmt.Sprintf("sp-%d-%d", index, elemID),
			Kind: "text",
		}
		elemID++

		if sp.SpPr.Xfrm != nil {
			elem.X = emuToPixels(sp.SpPr.Xfrm.Off.X)
			elem.Y = emuToPixels(sp.SpPr.Xfrm.Off.Y)
			elem.W = emuToPixels(sp.SpPr.Xfrm.Ext.Cx)
			elem.H = emuToPixels(sp.SpPr.Xfrm.Ext.Cy)
		}

		// Placeholder type
		if sp.NvSpPr.NvPr != nil && sp.NvSpPr.NvPr.Ph != nil {
			elem.PhType = sp.NvSpPr.NvPr.Ph.Type
		}

		// Text content
		if sp.TxBody != nil {
			var lines []string
			for _, p := range sp.TxBody.Paragraphs {
				var parts []string
				for _, r := range p.Runs {
					parts = append(parts, r.Text)
					if r.RPr != nil {
						elem.Bold = r.RPr.B == "1"
						elem.Italic = r.RPr.I == "1"
					}
				}
				lines = append(lines, strings.Join(parts, ""))
				if p.PPr != nil && p.PPr.Algn != "" {
					elem.Align = p.PPr.Algn
				}
			}
			elem.Text = strings.Join(lines, "\n")
		}

		info.Elements = append(info.Elements, elem)
	}

	// Parse pictures
	for _, pic := range slide.CSld.SpTree.Pics {
		elem := SlideElement{
			ID:   fmt.Sprintf("pic-%d-%d", index, elemID),
			Kind: "image",
		}
		elemID++

		if pic.SpPr.Xfrm != nil {
			elem.X = emuToPixels(pic.SpPr.Xfrm.Off.X)
			elem.Y = emuToPixels(pic.SpPr.Xfrm.Off.Y)
			elem.W = emuToPixels(pic.SpPr.Xfrm.Ext.Cx)
			elem.H = emuToPixels(pic.SpPr.Xfrm.Ext.Cy)
		}

		if pic.BlipFill != nil && pic.BlipFill.Blip != nil {
			elem.ImageID = pic.BlipFill.Blip.Embed
		}

		info.Elements = append(info.Elements, elem)
	}

	return info
}

// ── File operations ─────────────────────────────────────────────────

func (s *Service) OpenFile(tabID, path string) OpenResult {
	if path == "" {
		return OpenResult{Success: false, Error: "no file path"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return OpenResult{Success: false, Error: fmt.Sprintf("read: %v", err)}
	}

	state, err := parsePptx(data)
	if err != nil {
		return OpenResult{Success: false, Error: fmt.Sprintf("parse: %v", err)}
	}

	state.FilePath = path
	state.Title = filepath.Base(path)
	state.pushHistory("open")

	s.mu.Lock()
	s.openDecks[tabID] = state
	s.mu.Unlock()

	return OpenResult{
		Success:    true,
		FilePath:   path,
		Title:      state.Title,
		SlideCount: state.SlideCount,
		Slides:     state.Slides,
		Width:      state.Width,
		Height:     state.Height,
	}
}

func (s *Service) NewBlank(tabID string) OpenResult {
	state := &DeckState{
		Title:      "Untitled Presentation",
		Width:      960,
		Height:     540,
		slideXMLs:  make(map[string][]byte),
		otherParts: buildBlankPptxParts(),
		Slides: []SlideInfo{
			{
				Index: 0,
				Elements: []SlideElement{
					{ID: "sp-0-0", Kind: "text", X: 60, Y: 40, W: 840, H: 80, Text: "Click to add title", PhType: "ctrTitle"},
					{ID: "sp-0-1", Kind: "text", X: 60, Y: 160, W: 840, H: 360, Text: "Click to add content", PhType: "subTitle"},
				},
			},
		},
		SlideCount: 1,
	}
	state.pushHistory("new")

	s.mu.Lock()
	s.openDecks[tabID] = state
	s.mu.Unlock()

	return OpenResult{
		Success:    true,
		Title:      state.Title,
		SlideCount: 1,
		Slides:     state.Slides,
		Width:      state.Width,
		Height:     state.Height,
	}
}

func buildBlankPptxParts() map[string][]byte {
	return map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`),
		"_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`),
		"ppt/presentation.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldSz cx="9144000" cy="5143500"/>
  <p:sldIdLst><p:sldId id="256" r:id="rId2"/></p:sldIdLst>
</p:presentation>`),
		"ppt/_rels/presentation.xml.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`),
	}
}

// Save saves the presentation to its current path.
func (s *Service) Save(tabID string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDecks[tabID]
	s.mu.RUnlock()

	if !ok {
		return SaveResult{Success: false, Error: "presentation not found"}
	}
	if state.FilePath == "" {
		return SaveResult{Success: false, Error: "no file path - use SaveAs"}
	}

	if err := savePptx(state); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = false
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: state.FilePath}
}

func (s *Service) SaveAs(tabID, path string) SaveResult {
	s.mu.Lock()
	state, ok := s.openDecks[tabID]
	if ok {
		state.FilePath = path
		state.Title = filepath.Base(path)
	}
	s.mu.Unlock()

	if !ok {
		return SaveResult{Success: false, Error: "presentation not found"}
	}

	if err := savePptx(state); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("save: %v", err)}
	}

	s.mu.Lock()
	state.IsDirty = false
	s.mu.Unlock()

	return SaveResult{Success: true, FilePath: path}
}

func savePptx(state *DeckState) error {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Write slide XMLs (generate from model if modified, else use original)
	for name, data := range state.slideXMLs {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(data); err != nil {
			return err
		}
	}

	// Write other parts
	for name, data := range state.otherParts {
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

	return os.WriteFile(state.FilePath, buf.Bytes(), 0644)
}

// ── Slide manipulation ──────────────────────────────────────────────

func (s *Service) AddSlide(tabID string) SlideInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok {
		return SlideInfo{}
	}

	idx := len(state.Slides)
	slide := SlideInfo{
		Index: idx,
		Elements: []SlideElement{
			{ID: fmt.Sprintf("sp-%d-0", idx), Kind: "text", X: 60, Y: 40, W: 840, H: 80, Text: "Click to add title", PhType: "ctrTitle"},
			{ID: fmt.Sprintf("sp-%d-1", idx), Kind: "text", X: 60, Y: 160, W: 840, H: 360, Text: "Click to add content", PhType: "subTitle"},
		},
	}

	state.Slides = append(state.Slides, slide)
	state.SlideCount = len(state.Slides)
	state.IsDirty = true
	state.pushHistory("add slide")
	return slide
}

func (s *Service) DeleteSlide(tabID string, index int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || index < 0 || index >= len(state.Slides) || len(state.Slides) <= 1 {
		return false
	}

	state.Slides = append(state.Slides[:index], state.Slides[index+1:]...)
	// Re-index
	for i := range state.Slides {
		state.Slides[i].Index = i
	}
	state.SlideCount = len(state.Slides)
	state.IsDirty = true
	state.pushHistory("delete slide")
	return true
}

func (s *Service) DuplicateSlide(tabID string, index int) SlideInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || index < 0 || index >= len(state.Slides) {
		return SlideInfo{}
	}

	// Deep copy elements
	src := state.Slides[index]
	newSlide := SlideInfo{
		Index:    index + 1,
		Elements: make([]SlideElement, len(src.Elements)),
	}
	copy(newSlide.Elements, src.Elements)
	for i := range newSlide.Elements {
		newSlide.Elements[i].ID = fmt.Sprintf("sp-%d-%d", index+1, i)
	}

	// Insert after source
	state.Slides = append(state.Slides[:index+1], append([]SlideInfo{newSlide}, state.Slides[index+1:]...)...)
	for i := range state.Slides {
		state.Slides[i].Index = i
	}
	state.SlideCount = len(state.Slides)
	state.IsDirty = true
	state.pushHistory("duplicate slide")
	return newSlide
}

func (s *Service) MoveSlide(tabID string, from, to int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || from < 0 || from >= len(state.Slides) || to < 0 || to >= len(state.Slides) {
		return false
	}

	slide := state.Slides[from]
	state.Slides = append(state.Slides[:from], state.Slides[from+1:]...)
	state.Slides = append(state.Slides[:to], append([]SlideInfo{slide}, state.Slides[to:]...)...)
	for i := range state.Slides {
		state.Slides[i].Index = i
	}
	state.IsDirty = true
	state.pushHistory("move slide")
	return true
}

func (s *Service) UpdateElement(tabID string, slideIndex int, elemID, text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || slideIndex < 0 || slideIndex >= len(state.Slides) {
		return false
	}

	for i := range state.Slides[slideIndex].Elements {
		if state.Slides[slideIndex].Elements[i].ID == elemID {
			state.Slides[slideIndex].Elements[i].Text = text
			state.IsDirty = true
			state.pushHistory("edit element")
			return true
		}
	}
	return false
}

func (s *Service) GetSlides(tabID string) []SlideInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDecks[tabID]; ok {
		return state.Slides
	}
	return nil
}

func (s *Service) GetState(tabID string) *DeckState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDecks[tabID]; ok {
		return &DeckState{
			FilePath:   state.FilePath,
			IsDirty:    state.IsDirty,
			Title:      state.Title,
			SlideCount: state.SlideCount,
			Slides:     state.Slides,
			Width:      state.Width,
			Height:     state.Height,
		}
	}
	return nil
}

func (s *Service) IsDirty(tabID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.openDecks[tabID]; ok {
		return state.IsDirty
	}
	return false
}

func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openDecks, tabID)
}

// ── Undo/Redo ───────────────────────────────────────────────────────

func (state *DeckState) pushHistory(desc string) {
	if state.historyPos < len(state.history) {
		state.history = state.history[:state.historyPos]
	}
	snap := make([]SlideInfo, len(state.Slides))
	copy(snap, state.Slides)
	state.history = append(state.history, deckSnapshot{desc: desc, slides: snap})
	state.historyPos = len(state.history)
	if len(state.history) > 100 {
		state.history = state.history[len(state.history)-100:]
		state.historyPos = len(state.history)
	}
}

func (s *Service) Undo(tabID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.openDecks[tabID]
	if !ok || state.historyPos <= 1 {
		return false
	}
	state.historyPos--
	entry := state.history[state.historyPos-1]
	state.Slides = make([]SlideInfo, len(entry.slides))
	copy(state.Slides, entry.slides)
	state.SlideCount = len(state.Slides)
	state.IsDirty = true
	return true
}

func (s *Service) Redo(tabID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.openDecks[tabID]
	if !ok || state.historyPos >= len(state.history) {
		return false
	}
	entry := state.history[state.historyPos]
	state.Slides = make([]SlideInfo, len(entry.slides))
	copy(state.Slides, entry.slides)
	state.historyPos++
	state.SlideCount = len(state.Slides)
	state.IsDirty = true
	return true
}

// ── AI Skill ────────────────────────────────────────────────────────

func (s *Service) GetSlidesSkill() *agentcore.Skill {
	return &agentcore.Skill{
		ID: "slides",
		SystemPrompt: `You are an AI assistant for presentations. You can:
- Add, delete, duplicate, and reorder slides
- Edit text on slides (title, body, etc.)
- Insert shapes, images, and tables
- Apply themes and animations
Always describe what you changed.`,
		Tools: []agentcore.ToolDef{
			{Name: "add_slide", Description: "Add a new blank slide", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
			{Name: "delete_slide", Description: "Delete a slide by index", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"index": map[string]interface{}{"type": "integer"}}, "required": []string{"index"}}},
			{Name: "edit_text", Description: "Edit text on a slide element", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"slide": map[string]interface{}{"type": "integer"}, "element_id": map[string]interface{}{"type": "string"}, "text": map[string]interface{}{"type": "string"}}, "required": []string{"slide", "element_id", "text"}}},
		},
	}
}

// Suppress unused import warnings
var _ = time.Now
