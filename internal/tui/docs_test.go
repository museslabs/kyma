package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/museslabs/kyma/docs"
	"github.com/museslabs/kyma/internal/config"
)

// TestDocsPresentationRenders keeps the bundled documentation honest: every
// slide of it has to parse and fit its frame at the sizes people present at.
func TestDocsPresentationRenders(t *testing.T) {
	data, err := docs.FS.ReadFile("presentation.md")
	if err != nil {
		t.Fatalf("could not read the docs presentation: %v", err)
	}

	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		width, height := size[0], size[1]

		for i, raw := range strings.Split(string(data), "----\n") {
			body, properties := splitSlide(raw)

			props, err := config.NewProperties(properties)
			if err != nil {
				t.Fatalf("slide %d: %v", i+1, err)
			}
			props.ImageBackend = "docs"

			slide, err := NewSlide(body, props)
			if err != nil {
				t.Fatalf("slide %d: %v", i+1, err)
			}
			slide.Style = props.Style.Apply(width, height)

			lines := strings.Split(slide.View(false, width, height), "\n")
			for n, line := range lines {
				if got := ansi.StringWidth(line); got != width {
					t.Errorf(
						"%dx%d slide %d: line %d is %d cells wide, want %d:\n%q",
						width, height, i+1, n, got, width, ansi.Strip(line),
					)
				}
			}

			// Several slides are longer than a small terminal can show, which
			// the app reports on its own. Only hold the deck to fitting at the
			// larger size.
			if width >= 120 && len(lines) > height {
				t.Errorf(
					"%dx%d slide %d: rendered %d lines, which does not fit",
					width, height, i+1, len(lines),
				)
			}

			if out := ansi.Strip(slide.View(false, width, height)); strings.Contains(out, "⚠") {
				t.Errorf("%dx%d slide %d: layout error on the slide:\n%s", width, height, i+1, out)
			}
		}
	}
}

// splitSlide mirrors how the presentation loader separates front matter from a
// slide's body.
func splitSlide(s string) (body, properties string) {
	if !strings.HasPrefix(strings.TrimSpace(s), "---\n") {
		return s, ""
	}

	parts := strings.SplitN(s, "---\n", 3)
	if len(parts) < 3 {
		return s, ""
	}

	return parts[2], parts[1]
}
