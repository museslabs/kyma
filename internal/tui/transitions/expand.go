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

type expand struct {
	width     int
	fps       int
	spring    harmonica.Spring
	progress  float64
	vel       float64
	animating bool
	direction direction
}

func newExpand(fps int) expand {
	const frequency = 7.0
	const damping = 0.6

	return expand{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t expand) Start(width, _ int, direction direction) Transition {
	t.width = width
	t.animating = true
	t.progress = 0
	t.vel = 0
	t.direction = direction
	return t
}

func (t expand) Animating() bool {
	return t.animating
}

func (t expand) Update() (Transition, tea.Cmd) {
	targetProgress := 1.0

	t.progress, t.vel = t.spring.Update(t.progress, t.vel, targetProgress)

	if t.progress >= 0.99 {
		t.animating = false
		t.progress = 1.0
		return t, nil
	}

	return t, Animate(time.Duration(t.fps))
}

func (t expand) View(prev, next string) string {
	var s strings.Builder

	// Calculate how much should expand from center outward
	expandWidth := int(math.Round(t.progress * float64(t.width) / 2))
	centerStart := t.width/2 - expandWidth
	centerEnd := t.width/2 + expandWidth

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
		if expandWidth >= t.width/2 {
			// Animation complete, show next content
			line = truncate.String(nextLine, uint(t.width))
		} else {
			// Build the line with expand effect from center outward
			// Left and right parts: show prev content
			leftPrev := truncate.String(prevLine, uint(centerStart))

			rightPrev := ""
			if centerEnd < t.width {
				rightPrev = charmansi.TruncateLeft(prevLine, centerEnd, "")
			}

			// Center portion: show next content expanding from center
			var center string
			if centerEnd > centerStart {
				truncatedNext := truncate.String(nextLine, uint(centerEnd))
				center = skip.String(truncatedNext, uint(centerStart))
			}

			line = leftPrev + center + rightPrev
			line = truncate.String(line, uint(t.width))
		}

		s.WriteString(line)
		if i < maxLines-1 {
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (t expand) Name() string {
	return "expand"
}

func (t expand) Opposite() Transition {
	return newExpand(t.fps)
}

func (t expand) Direction() direction {
	return t.direction
}
