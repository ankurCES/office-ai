// Package slides implements the presentation editor module,
// mirroring GenOffice's apps/slides. Handles pptx parsing, Konva-style
// rendering data, element editing, and AI slide generation.
package slides

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ankurCES/office-ai/pkg/agentcore"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
)

// SlideElement represents a shape/text/image on a slide.
type SlideElement struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "shape", "textbox", "image", "table", "chart"
	X        float64                `json:"x"`
	Y        float64                `json:"y"`
	Width    float64                `json:"width"`
	Height   float64                `json:"height"`
	Rotation float64                `json:"rotation,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
}

// Slide represents a single slide with its elements.
type Slide struct {
	ID         string         `json:"id"`
	Index      int            `json:"index"`
	LayoutName string         `json:"layout_name"`
	Elements   []SlideElement `json:"elements"`
	Notes      string         `json:"notes,omitempty"`
}

// PresentationState holds the in-memory state of an open presentation.
type PresentationState struct {
	FilePath    string  `json:"file_path"`
	Title       string  `json:"title"`
	IsDirty     bool    `json:"is_dirty"`
	Slides      []Slide `json:"slides"`
	ActiveSlide int     `json:"active_slide"`
	SlideWidth  float64 `json:"slide_width"`
	SlideHeight float64 `json:"slide_height"`
}

// Service is the slides module service bound to Wails.
type Service struct {
	ctx       context.Context
	i18nSvc   *i18n.Service
	store     *projectstore.Store
	agent     *agentcore.Loop
	mu        sync.RWMutex
	openDecks map[string]*PresentationState
}

// New creates a new slides Service.
func New(i18nSvc *i18n.Service, store *projectstore.Store, agent *agentcore.Loop) *Service {
	return &Service{
		i18nSvc:   i18nSvc,
		store:     store,
		agent:     agent,
		openDecks: make(map[string]*PresentationState),
	}
}

// OpenFile opens a .pptx file and returns its parsed state.
func (s *Service) OpenFile(tabID, path string) map[string]interface{} {
	result := map[string]interface{}{"success": false}

	data, err := os.ReadFile(path)
	if err != nil {
		result["error"] = fmt.Sprintf("read file: %v", err)
		return result
	}
	if len(data) < 4 || string(data[:2]) != "PK" {
		result["error"] = "not a valid pptx file"
		return result
	}

	// TODO: Use pptxengine for full XML parsing
	state := &PresentationState{
		FilePath:    path,
		Title:       filepath.Base(path),
		SlideWidth:  9144000, // EMU (10" standard)
		SlideHeight: 6858000, // EMU (7.5" standard)
		Slides: []Slide{
			{ID: "slide1", Index: 0, LayoutName: "Blank", Elements: nil},
		},
	}

	s.mu.Lock()
	s.openDecks[tabID] = state
	s.mu.Unlock()

	result["success"] = true
	result["file_path"] = path
	result["title"] = state.Title
	result["state"] = state
	return result
}

// NewBlank creates a new blank presentation.
func (s *Service) NewBlank(tabID string) map[string]interface{} {
	state := &PresentationState{
		Title:       "Untitled Presentation",
		SlideWidth:  9144000,
		SlideHeight: 6858000,
		Slides: []Slide{
			{ID: "slide1", Index: 0, LayoutName: "Title Slide", Elements: nil},
		},
	}

	s.mu.Lock()
	s.openDecks[tabID] = state
	s.mu.Unlock()

	return map[string]interface{}{
		"success": true,
		"title":   state.Title,
		"state":   state,
	}
}

// AddSlide inserts a new slide at the given index.
func (s *Service) AddSlide(tabID string, afterIndex int, layout string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok {
		return
	}

	newSlide := Slide{
		ID:         fmt.Sprintf("slide%d", len(state.Slides)+1),
		Index:      afterIndex + 1,
		LayoutName: layout,
	}

	// Insert after the specified index
	idx := afterIndex + 1
	if idx >= len(state.Slides) {
		state.Slides = append(state.Slides, newSlide)
	} else {
		state.Slides = append(state.Slides[:idx+1], state.Slides[idx:]...)
		state.Slides[idx] = newSlide
	}

	// Re-index
	for i := range state.Slides {
		state.Slides[i].Index = i
	}
	state.IsDirty = true
}

// DeleteSlide removes a slide by index.
func (s *Service) DeleteSlide(tabID string, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || index < 0 || index >= len(state.Slides) || len(state.Slides) <= 1 {
		return
	}

	state.Slides = append(state.Slides[:index], state.Slides[index+1:]...)
	for i := range state.Slides {
		state.Slides[i].Index = i
	}
	state.IsDirty = true
}

// AddElement adds an element to a slide.
func (s *Service) AddElement(tabID string, slideIndex int, elem SlideElement) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || slideIndex < 0 || slideIndex >= len(state.Slides) {
		return
	}

	if elem.ID == "" {
		elem.ID = fmt.Sprintf("elem%d", len(state.Slides[slideIndex].Elements)+1)
	}
	state.Slides[slideIndex].Elements = append(state.Slides[slideIndex].Elements, elem)
	state.IsDirty = true
}

// DeleteElement removes an element from a slide.
func (s *Service) DeleteElement(tabID string, slideIndex int, elemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.openDecks[tabID]
	if !ok || slideIndex < 0 || slideIndex >= len(state.Slides) {
		return
	}

	elems := state.Slides[slideIndex].Elements
	for i, e := range elems {
		if e.ID == elemID {
			state.Slides[slideIndex].Elements = append(elems[:i], elems[i+1:]...)
			state.IsDirty = true
			return
		}
	}
}

// Save saves the presentation.
func (s *Service) Save(tabID string) map[string]interface{} {
	s.mu.RLock()
	state, ok := s.openDecks[tabID]
	s.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"success": false, "error": "presentation not found"}
	}
	if state.FilePath == "" {
		return map[string]interface{}{"success": false, "error": "no file path - use SaveAs"}
	}

	// TODO: Use pptxengine to write back changes
	return map[string]interface{}{"success": true, "file_path": state.FilePath}
}

// Close closes an open presentation.
func (s *Service) Close(tabID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.openDecks, tabID)
}
