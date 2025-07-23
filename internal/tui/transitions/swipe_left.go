package transitions

import (
	"log/slog"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/muesli/reflow/truncate"

	"github.com/museslabs/kyma/internal/skip"
)

type swipeLeft struct {
	width     int
	fps       int
	spring    harmonica.Spring
	x         float64
	xVel      float64
	animating bool
	direction direction
}

func newSwipeLeft(fps int) swipeLeft {
	const frequency = 7.0
	const damping = 0.75

	return swipeLeft{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t swipeLeft) Start(width int, _ int, direction direction) Transition {
	t.width = width
	t.animating = true
	t.x = 0
	t.xVel = 0
	t.direction = direction
	return t
}

func (t swipeLeft) Animating() bool {
	return t.animating
}

func (t swipeLeft) Update() (Transition, tea.Cmd) {
	targetX := float64(t.width)

	t.x, t.xVel = t.spring.Update(t.x, t.xVel, targetX)

	if t.x >= targetX {
		t.animating = false
		return t, nil
	}

	return t, Animate(time.Duration(t.fps))
}

func (t swipeLeft) View(prev string, next string) string {
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
		prev := skip.String(prevLines[i], uint(x))
		next := truncate.String(nextLines[i], uint(x))
		s.WriteString(prev + " " + next)
		if i < len(nextLines)-1 {
			s.WriteString("\n")
		}
	}
	return s.String()
}

func (t swipeLeft) Name() string {
	return "swipeLeft"
}

func (t swipeLeft) Opposite() Transition {
	return newSwipeRight(t.fps)
}

func (t swipeLeft) Direction() direction {
	return t.direction
}
