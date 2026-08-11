package docs

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"strings"
)

// ExportHTML exports the document as an HTML file.
func (s *Service) ExportHTML(tabID, path string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok || state.doc == nil {
		return SaveResult{Success: false, Error: "document not found"}
	}

	htmlContent := docToHTML(state)

	if err := os.WriteFile(path, []byte(htmlContent), 0644); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("write: %v", err)}
	}

	return SaveResult{Success: true, FilePath: path}
}

// ExportText exports the document as plain text.
func (s *Service) ExportText(tabID, path string) SaveResult {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok || state.doc == nil {
		return SaveResult{Success: false, Error: "document not found"}
	}

	var buf bytes.Buffer
	for _, p := range state.doc.Body.Paragraphs {
		for _, r := range p.Runs {
			for _, t := range r.Text {
				buf.WriteString(t.Value)
			}
			if r.Br != nil {
				buf.WriteByte('\n')
			}
		}
		buf.WriteByte('\n')
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return SaveResult{Success: false, Error: fmt.Sprintf("write: %v", err)}
	}

	return SaveResult{Success: true, FilePath: path}
}

// GetHTMLPreview returns the document as HTML for in-app rendering/printing.
func (s *Service) GetHTMLPreview(tabID string) string {
	s.mu.RLock()
	state, ok := s.openDocs[tabID]
	s.mu.RUnlock()

	if !ok || state.doc == nil {
		return "<html><body><p>No document open</p></body></html>"
	}
	return docToHTML(state)
}

func docToHTML(state *DocState) string {
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>`)
	buf.WriteString(html.EscapeString(state.Title))
	buf.WriteString(`</title>
<style>
  body {
    font-family: 'Segoe UI', -apple-system, sans-serif;
    max-width: 800px;
    margin: 40px auto;
    padding: 0 20px;
    line-height: 1.6;
    color: #333;
  }
  @media print {
    body { margin: 0; max-width: none; }
  }
  h1 { font-size: 2em; margin-top: 1em; }
  h2 { font-size: 1.5em; margin-top: 0.8em; }
  h3 { font-size: 1.17em; margin-top: 0.6em; }
  p { margin: 0.5em 0; }
  .align-center { text-align: center; }
  .align-right { text-align: right; }
  .align-justify { text-align: justify; }
</style>
</head>
<body>
`)

	for _, p := range state.doc.Body.Paragraphs {
		tag := "p"
		className := ""
		if p.PPr != nil {
			if p.PPr.PStyle != nil {
				switch p.PPr.PStyle.Val {
				case "Heading1":
					tag = "h1"
				case "Heading2":
					tag = "h2"
				case "Heading3":
					tag = "h3"
				}
			}
			if p.PPr.Jc != nil {
				switch p.PPr.Jc.Val {
				case "center":
					className = "align-center"
				case "right":
					className = "align-right"
				case "both", "justify":
					className = "align-justify"
				}
			}
		}

		if className != "" {
			buf.WriteString(fmt.Sprintf("<%s class=\"%s\">", tag, className))
		} else {
			buf.WriteString(fmt.Sprintf("<%s>", tag))
		}

		for _, r := range p.Runs {
			openTags := ""
			closeTags := ""
			if r.RPr != nil {
				if r.RPr.Bold != nil {
					openTags += "<strong>"
					closeTags = "</strong>" + closeTags
				}
				if r.RPr.Italic != nil {
					openTags += "<em>"
					closeTags = "</em>" + closeTags
				}
				if r.RPr.Underline != nil {
					openTags += "<u>"
					closeTags = "</u>" + closeTags
				}
				if r.RPr.Strike != nil {
					openTags += "<s>"
					closeTags = "</s>" + closeTags
				}
			}

			for _, t := range r.Text {
				text := html.EscapeString(t.Value)
				if text == "" {
					continue
				}
				buf.WriteString(openTags)
				buf.WriteString(text)
				buf.WriteString(closeTags)
			}

			if r.Br != nil {
				buf.WriteString("<br>")
			}
		}

		// If paragraph is empty, add &nbsp; to prevent collapse
		allText := ""
		for _, r := range p.Runs {
			for _, t := range r.Text {
				allText += t.Value
			}
		}
		if strings.TrimSpace(allText) == "" {
			buf.WriteString("&nbsp;")
		}

		buf.WriteString(fmt.Sprintf("</%s>\n", tag))
	}

	buf.WriteString("</body>\n</html>")
	return buf.String()
}
