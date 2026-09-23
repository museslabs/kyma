package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/museslabs/kyma/internal/config"
)

// TestSlide_ViewFitsTheFrame is the invariant a presenter actually notices: the
// slide, border included, is exactly as wide as the terminal, so a grid never
// pushes content past the frame.
func TestSlide_ViewFitsTheFrame(t *testing.T) {
	bodies := map[string]string{
		"plain":   "# Title\n\nsome text",
		"columns": "[row]\n[col]left[/col]\n[col]right[/col]\n[/row]",
		"master": "# Title\n\n[grid]\n[col span=2]\n## Main\n\nbody\n[/col]\n" +
			"[col]\n[row]one[/row]\n[row]two[/row]\n[/col]\n[/grid]",
		"bordered columns": "[row gap=2]\n[col border=rounded pad=1]a[/col]\n" +
			"[col border=rounded pad=1]b[/col]\n[/row]",
	}

	for _, size := range [][2]int{{80, 24}, {120, 40}, {60, 20}} {
		width, height := size[0], size[1]

		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				slide, err := NewSlide(body, config.Properties{})
				if err != nil {
					t.Fatalf("NewSlide() failed: %v", err)
				}
				slide.Style = config.StyleConfig{}.Apply(width, height)

				for i, line := range strings.Split(slide.View(false, width, height), "\n") {
					if got := ansi.StringWidth(line); got != width {
						t.Errorf(
							"%dx%d: line %d is %d cells wide, want %d:\n%q",
							width, height, i, got, width, ansi.Strip(line),
						)
					}
				}
			})
		}
	}
}

// TestSlide_MasterLayout runs a slide through the templating layer the way the
// presentation loader does.
func TestSlide_MasterLayout(t *testing.T) {
	config.GlobalConfig.Masters = map[string]string{
		"two-col": "[row]\n[col span=2]\n[slot main]\n[/col]\n[col]\n[slot side]\n[/col]\n[/row]",
	}
	t.Cleanup(func() { config.GlobalConfig.Masters = nil })

	slide, err := NewSlide(
		"[slot main]\n# Headline\n[/slot]\n[slot side]\n- a note\n[/slot]",
		config.Properties{Master: "two-col"},
	)
	if err != nil {
		t.Fatalf("NewSlide() failed: %v", err)
	}
	slide.Style = config.StyleConfig{}.Apply(80, 24)

	out := ansi.Strip(slide.View(false, 80, 24))
	for _, want := range []string{"Headline", "a note"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered slide is missing %q:\n%s", want, out)
		}
	}

	// The layout put them side by side, so both land on the same line.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Headline") && strings.Contains(line, "a note") {
			return
		}
	}
	t.Errorf("the slots were not laid out side by side:\n%s", out)
}

func TestSlide_UnknownMasterLayout(t *testing.T) {
	if _, err := NewSlide("# Title", config.Properties{Master: "nope"}); err == nil {
		t.Error("NewSlide() accepted a layout that does not exist")
	}
}
