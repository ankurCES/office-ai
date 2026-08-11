package slides

import (
	"testing"
	"github.com/ankurCES/office-ai/pkg/i18n"
	"github.com/ankurCES/office-ai/pkg/projectstore"
	"github.com/ankurCES/office-ai/pkg/agentcore"
)

func TestNewBlankFlow(t *testing.T) {
	svc := New(i18n.New(), projectstore.New(), agentcore.New(nil))
	result := svc.NewBlank("test-sl-1")
	if !result.Success {
		t.Fatalf("NewBlank failed")
	}
	if result.Title == "" {
		t.Fatal("Title empty")
	}
	if result.SlideCount != 1 {
		t.Fatalf("Expected 1 slide, got %d", result.SlideCount)
	}
	if len(result.Slides) != 1 {
		t.Fatalf("Expected 1 slide in array, got %d", len(result.Slides))
	}
	t.Logf("NewBlank: title=%q slides=%d elements=%d", result.Title, result.SlideCount, len(result.Slides[0].Elements))

	// Add a slide
	newSlide := svc.AddSlide("test-sl-1")
	t.Logf("AddSlide: index=%d", newSlide.Index)

	// GetSlides should now return 2
	slides := svc.GetSlides("test-sl-1")
	if len(slides) != 2 {
		t.Fatalf("Expected 2 slides, got %d", len(slides))
	}
	t.Logf("After AddSlide: %d slides", len(slides))
}
