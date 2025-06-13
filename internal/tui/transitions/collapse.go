package transitions

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	charmansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"

	"github.com/museslabs/kyma/internal/skip"
)

type collapse struct {
	width     int
	fps       int
	spring    harmonica.Spring
	progress  float64
	vel       float64
	animating bool
	direction direction
}

func newCollapse(fps int) collapse {
	const frequency = 7.0
	const damping = 0.6

	return collapse{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t collapse) Start(width, _ int, direction direction) Transition {
	t.width = width
	t.animating = true
	t.progress = 0
	t.vel = 0
	t.direction = direction
	return t
}

func (t collapse) Animating() bool {
	return t.animating
}

func (t collapse) Update() (Transition, tea.Cmd) {
	targetProgress := 1.0

	t.progress, t.vel = t.spring.Update(t.progress, t.vel, targetProgress)

	if t.progress >= 0.99 {
		t.animating = false
		t.progress = 1.0
		return t, nil
	}

	return t, Animate(time.Duration(t.fps))
}

func (t collapse) View(prev, next string) string {
	var s strings.Builder

	// Calculate how much should collapse from edges toward center
	collapseWidth := int(math.Round((1.0 - t.progress) * float64(t.width) / 2))
	centerStart := t.width/2 - collapseWidth
	centerEnd := t.width/2 + collapseWidth

	prevLines := strings.Split(prev, "\n")
	nextLines := strings.Split(next, "\n")

	// Ensure slides are equal height
	maxLines := max(len(nextLines), len(prevLines))

	for i := range maxLines {
		var prevLine, nextLine string

		if i < len(prevLines) {
			prevLine = prevLines[i]
		}
		if i < len(nextLines) {
			nextLine = nextLines[i]
		}

		var line string
		if collapseWidth <= 0 {
			// Animation complete, show next content
			line = truncate.String(nextLine, uint(t.width))
		} else {
			// Build the line with collapse effect from edges toward center
			// Center portion: show prev content that's collapsing
			var center string
			if centerEnd > centerStart {
				truncatedPrev := truncate.String(prevLine, uint(centerEnd))
				center = skip.String(truncatedPrev, uint(centerStart))
			}

			// Left and right parts: show next content appearing from edges
			leftNext := truncate.String(nextLine, uint(centerStart))

			rightNext := ""
			if centerEnd < t.width {
				rightNext = charmansi.TruncateLeft(nextLine, centerEnd, "")
			}

			line = leftNext + center + rightNext
			line = truncate.String(line, uint(t.width))
		}

		s.WriteString(line)
		if i < maxLines-1 {
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (t collapse) Name() string {
	return "collapse"
}

func (t collapse) Opposite() Transition {
	return newExpand(t.fps)
}

func (t collapse) Direction() direction {
	return t.direction
}
