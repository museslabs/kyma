package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TimerTickMsg struct{}

type Timer struct {
	startTime time.Time
	duration  time.Duration
	running   bool
}

func NewTimer() Timer {
	return Timer{
		startTime: time.Now(),
		duration:  0,
		running:   false,
	}
}

func (t Timer) Start() Timer {
	if !t.running {
		t.startTime = time.Now()
		t.duration = 0
		t.running = true
	}
	return t
}


func (t Timer) Reset() Timer {
	t.startTime = time.Now()
	t.duration = 0
	return t
}

func (t Timer) Pause() Timer {
	if t.running {
		t.duration = time.Since(t.startTime)
		t.running = false
	}
	return t
}

func (t Timer) Resume() Timer {
	if !t.running {
		t.startTime = time.Now().Add(-t.duration)
		t.running = true
	}
	return t
}

func (t Timer) Update(msg tea.Msg) (Timer, tea.Cmd) {
	switch msg.(type) {
	case TimerTickMsg:
		if t.running {
			t.duration = time.Since(t.startTime)
		}
		return t, tea.Tick(time.Second, func(time.Time) tea.Msg {
			return TimerTickMsg{}
		})
	}
	return t, nil
}

func (t Timer) Duration() time.Duration {
	if t.running {
		return time.Since(t.startTime)
	}
	return t.duration
}

func (t Timer) FormatDuration() string {
	d := t.Duration()
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (t Timer) IsRunning() bool {
	return t.running
}

type TimerDisplay struct {
	visible bool
}

func NewTimerDisplay() TimerDisplay {
	return TimerDisplay{
		visible: false,
	}
}

func (td TimerDisplay) Update(msg tea.Msg) (TimerDisplay, tea.Cmd) {
	// TimerDisplay doesn't need to update timers, just handle display state
	return td, nil
}

func (td TimerDisplay) Show(slideView string, width, height int, globalTimer Timer, slideTimer Timer) string {
	if !td.visible {
		return slideView
	}

	globalTimeStr := globalTimer.FormatDuration()
	slideTimeStr := slideTimer.FormatDuration()

	timerContent := lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2A2A")).
		Foreground(lipgloss.Color("#DDDDDD")).
		Padding(0, 1).
		Render(fmt.Sprintf("Total:  %s\nSlide:  %s", globalTimeStr, slideTimeStr))

	// Position in top left
	timerX := 3
	timerY := 1

	return placeOverlay(timerX, timerY, timerContent, slideView)
}

func (td TimerDisplay) IsVisible() bool {
	return td.visible
}

func (td TimerDisplay) ToggleVisible() TimerDisplay {
	td.visible = !td.visible
	return td
}

func EnsureTimerInitialized(slide *Slide) {
	if slide != nil && slide.Timer.startTime.IsZero() && slide.Timer.duration == 0 && !slide.Timer.running {
		slide.Timer = NewTimer()
	}
}
