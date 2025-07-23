package transitions

import (
	"log/slog"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wordwrap"
)

type flipRight struct {
	width     int
	fps       int
	spring    harmonica.Spring
	x         float64
	xVel      float64
	animating bool
	direction direction
}

func newFlipRight(fps int) flipRight {
	const frequency = 7.0
	const damping = 0.8

	return flipRight{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t flipRight) Start(width int, _ int, direction direction) Transition {
	t.width = width
	t.animating = true
	t.x = 0
	t.xVel = 0
	t.direction = direction
	return t
}

func (t flipRight) Animating() bool {
	return t.animating
}

func (t flipRight) Update() (Transition, tea.Cmd) {
	targetX := float64(t.width)

	t.x, t.xVel = t.spring.Update(t.x, t.xVel, targetX)

	if t.x >= targetX {
		t.animating = false
		return t, nil
	}

	return t, Animate(time.Duration(t.fps))
}

func (t flipRight) View(prev string, next string) string {
	var s strings.Builder

	x := int(math.Round(t.x))

	prevLines := strings.Split(prev, "\n")
	nextLines := strings.Split(next, "\n")

	// assert that slides are always equal height
	if len(nextLines) != len(prevLines) {
		slog.Error("Slides of not equal height")
		return next
	}

	for i := range nextLines {
		wrappedPrev := strings.Split(wordwrap.String(prevLines[i], x), "\n")
		prev = wrappedPrev[len(wrappedPrev)-1]
		next := truncate.String(nextLines[i], uint(x))
		s.WriteString(next + " " + prev)
		if i < len(nextLines)-1 {
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (t flipRight) Name() string {
	return "flipRight"
}

func (t flipRight) Opposite() Transition {
	return t
}

func (t flipRight) Direction() direction {
	return t.direction
}
