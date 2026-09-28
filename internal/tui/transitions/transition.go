package transitions

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type direction byte

const (
	Forwards  direction = 0
	Backwards direction = 1
)

type FrameMsg time.Time

const Fps = 60

// RefreshRate is how many times a second the screen is redrawn, the most
// bubbletea allows. Transitions that run on the clock animate at this rate.
const RefreshRate = 120

// Animate asks for the next frame. Frames land on the ticks of the clock
// rather than an interval after the last one finished, so they come evenly
// spaced and in step with the screen being redrawn at the same rate.
func Animate(fps time.Duration) tea.Cmd {
	fmt.Print("\x1b_Ga=d\x1b\\")
	return tea.Every(time.Second/fps, func(t time.Time) tea.Msg {
		return FrameMsg(t)
	})
}

type Transition interface {
	Start(width, height int, direction direction) Transition
	Animating() bool
	Update() (Transition, tea.Cmd)
	View(prev, next string) string
	Opposite() Transition
	Name() string
	Direction() direction
}

func Get(name string, fps int) Transition {
	switch name {
	case "slideUp":
		return newSlideUp(fps)
	case "slideDown":
		return newSlideDown(fps)
	case "swipeLeft":
		return newSwipeLeft(fps)
	case "swipeRight":
		return newSwipeRight(fps)
	case "flip":
		return newFlipRight(fps)
	case "collapse":
		return newCollapse(fps)
	case "expand":
		return newExpand(fps)
	case "fade":
		return newFade(fps)
	case "morph":
		return newMorph(fps)
	default:
		return newNoTransition(fps)
	}
}
