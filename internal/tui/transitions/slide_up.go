package transitions

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"

	"github.com/ploMP4/kyma/internal/tui/messages"
)

type slideUp struct {
	height    int
	fps       int
	spring    harmonica.Spring
	y         float64
	yVel      float64
	animating bool
}

func newSlideUp(fps int) slideUp {
	const frequency = 7.0
	const damping = 0.8

	return slideUp{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t slideUp) Start(_, height int) Transition {
	t.height = height
	t.animating = true
	t.y = 0
	t.yVel = 0
	return t
}

func (t slideUp) Animating() bool {
	return t.animating
}

func (t slideUp) Update() (Transition, tea.Cmd) {
	targetY := float64(t.height)

	t.y, t.yVel = t.spring.Update(t.y, t.yVel, targetY)

	if t.y >= targetY {
		t.animating = false
		return t, nil
	}

	return t, messages.Animate(time.Duration(t.fps))
}

func (t slideUp) View(prev, next string) string {
	var s strings.Builder

	y := int(math.Round(t.y))

	s.WriteString(prev)

	lines := strings.Split(next, "\n")
	if y > len(lines) {
		y = len(lines)
	}
	s.WriteString("\n" + strings.Join(lines[:y], "\n"))

	return s.String()
}

func (t slideUp) Name() string {
	return "slideUp"
}
