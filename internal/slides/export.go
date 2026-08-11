package slides

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
)

// ExportHTML exports the presentation as an HTML slideshow file.
func (s *Service) ExportHTML(tabID, path string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDecks[tabID]
	s.mu.RUnlock()

	if !ok {
		return SaveResult{Success: false, Error: "presentation not found"}
	}

	htmlContent := deckToHTML(state)

	if err := os.WriteFile(path, []byte(htmlContent), 0644); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("write: %v", err)}
	}

	return SaveResult{Success: true, FilePath: path}
}

// GetSlideSVG returns an SVG representation of a slide for export.
func (s *Service) GetSlideSVG(tabID string, slideIndex int) string {
	s.mu.RLock()
	state, ok := s.openDecks[tabID]
	s.mu.RUnlock()

	if !ok || slideIndex < 0 || slideIndex >= len(state.Slides) {
		return ""
	}

	slide := state.Slides[slideIndex]
	return slideToSVG(slide, state.Width, state.Height)
}

// GetSlideData returns raw slide data as JSON (for frontend rendering/export).
func (s *Service) GetSlideData(tabID string, slideIndex int) string {
	s.mu.RLock()
	state, ok := s.openDecks[tabID]
	s.mu.RUnlock()

	if !ok || slideIndex < 0 || slideIndex >= len(state.Slides) {
		return "{}"
	}

	data, _ := json.Marshal(state.Slides[slideIndex])
	return string(data)
}

func slideToSVG(slide SlideInfo, w, h int) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, w, h, w, h))
	buf.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>`, w, h))

	for _, elem := range slide.Elements {
		switch elem.Kind {
		case "text":
			fontSize := 24
			if elem.PhType == "ctrTitle" || elem.PhType == "title" {
				fontSize = 36
			}
			anchor := "start"
			if elem.Align == "ctr" {
				anchor = "middle"
			} else if elem.Align == "r" {
				anchor = "end"
			}
			x := elem.X
			if anchor == "middle" {
				x = elem.X + elem.W/2
			}
			buf.WriteString(fmt.Sprintf(
				`<text x="%d" y="%d" font-size="%d" font-family="sans-serif" text-anchor="%s" fill="#333">`,
				x, elem.Y+fontSize, fontSize, anchor,
			))
			buf.WriteString(html.EscapeString(elem.Text))
			buf.WriteString("</text>")

		case "image":
			buf.WriteString(fmt.Sprintf(
				`<rect x="%d" y="%d" width="%d" height="%d" fill="#eee" stroke="#ccc" rx="4"/>`,
				elem.X, elem.Y, elem.W, elem.H,
			))
			buf.WriteString(fmt.Sprintf(
				`<text x="%d" y="%d" font-size="14" text-anchor="middle" fill="#999">Image</text>`,
				elem.X+elem.W/2, elem.Y+elem.H/2,
			))

		case "shape":
			buf.WriteString(fmt.Sprintf(
				`<rect x="%d" y="%d" width="%d" height="%d" fill="#ddd" stroke="#aaa" rx="2"/>`,
				elem.X, elem.Y, elem.W, elem.H,
			))
		}
	}

	buf.WriteString("</svg>")
	return buf.String()
}

func deckToHTML(state *DeckState) string {
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>`)
	buf.WriteString(html.EscapeString(state.Title))
	buf.WriteString(`</title>
<style>
  body { margin: 0; background: #222; display: flex; flex-direction: column; align-items: center; gap: 20px; padding: 20px; }
  .slide { background: white; box-shadow: 0 2px 10px rgba(0,0,0,0.3); margin: 10px auto; }
  @media print {
    body { background: white; gap: 0; padding: 0; }
    .slide { box-shadow: none; page-break-after: always; margin: 0; }
  }
</style>
</head>
<body>
`)

	for _, slide := range state.Slides {
		svg := slideToSVG(slide, state.Width, state.Height)
		buf.WriteString(fmt.Sprintf(`<div class="slide">%s</div>`+"\n", svg))
	}

	buf.WriteString("</body>\n</html>")
	return buf.String()
}
