package transitions

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/muesli/reflow/truncate"

	"github.com/ploMP4/kyma/internal/skip"
	"github.com/ploMP4/kyma/internal/tui/messages"
)

type swipeRight struct {
	width     int
	fps       int
	spring    harmonica.Spring
	x         float64
	xVel      float64
	animating bool
	direction direction
}

func newSwipeRight(fps int) swipeRight {
	const frequency = 7.0
	const damping = 0.75

	return swipeRight{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t swipeRight) Start(width int, _ int) Transition {
	t.width = width
	t.animating = true
	t.x = 0
	t.xVel = 0
	t.direction = direction
	return t
}

func (t swipeRight) Animating() bool {
	return t.animating
}

func (t swipeRight) Update() (Transition, tea.Cmd) {
	targetX := -float64(t.width)

	t.x, t.xVel = t.spring.Update(t.x, t.xVel, targetX)

	if t.x <= targetX {
		t.animating = false
		return t, nil
	}

	return t, messages.Animate(time.Duration(t.fps))
}

func (t swipeRight) View(prev string, next string) string {
	var s strings.Builder

	x := int(math.Round(t.x))

	prevLines := strings.Split(prev, "\n")
	nextLines := strings.Split(next, "\n")

	// assert that slides are always equal height
	if len(nextLines) != len(prevLines) {
		panic("Slides of not equal height")
	}

	for i := range nextLines {
		prev := truncate.String(prevLines[i], uint(t.width+x))
		next := skip.String(nextLines[i], uint(t.width+x))
		s.WriteString(next + " " + prev)
		if i < len(nextLines)-1 {
			s.WriteString("\n")
		}
	}
	return s.String()
}

func (t swipeRight) Name() string {
	return "swipeRight"
}

func (t swipeRight) Opposite() Transition {
	return newSwipeLeft(t.fps)
}

func (t swipeRight) Direction() direction {
	return t.direction
}
