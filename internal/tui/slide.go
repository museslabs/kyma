package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/process"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type Slide struct {
	Data             string
	Prev             *Slide
	Next             *Slide
	Style            config.SlideStyle
	ActiveTransition transitions.Transition
	Properties       config.Properties
	Title            string
	Timer            Timer

	processors []process.PreProcessor

	imageProcessor process.PreProcessor
}

type Options struct {
	Animating bool
}

func WithAnimating() func(o *Options) {
	return func(o *Options) {
		o.Animating = true
	}
}

type UpdateSlidesMsg struct {
	NewRoot *Slide
}

func (s *Slide) Update() (*Slide, tea.Cmd) {
	if s.imageProcessor == nil {
		s.imageProcessor = process.NewImageProcessor()
	}

	transition, cmd := s.ActiveTransition.Update()
	s.ActiveTransition = transition

	// Update timer
	var timerCmd tea.Cmd
	s.Timer, timerCmd = s.Timer.Update(TimerTickMsg{})

	return s, tea.Batch(cmd, timerCmd)
}

func (s *Slide) View(opts ...func(o *Options)) string {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}

	var b strings.Builder

	themeName := "dark"

	if s.Style.Theme.Name != "" {
		themeName = s.Style.Theme.Name
	}

	if s.imageProcessor == nil {
		s.imageProcessor = process.NewImageProcessor()
	}

	out, _ := s.imageProcessor.Pre(
		s.Data,
		themeName,
		(s.ActiveTransition != nil && s.ActiveTransition.Animating()) || options.Animating,
	)

	// // Pre-process markdown to handle custom highlighting syntax
	// out, err := processMarkdownWithHighlighting(s.Data, themeName)
	// if err != nil {
	// 	// If preprocessing fails, fall back to regular Glamour rendering
	// 	out, err = glamour.Render(s.Data, themeName)
	// 	if err != nil {
	// 		b.WriteString("\n\n" + lipgloss.NewStyle().
	// 			Foreground(lipgloss.Color("9")). // Red
	// 			Render("Error: "+err.Error()))
	// 		return b.String()
	// 	}
	// }

	if s.ActiveTransition != nil && s.ActiveTransition.Animating() {
		direction := s.ActiveTransition.Direction()
		if direction == transitions.Backwards {
			if s.Next == nil {
				panic("backwards transition at the last slide")
			} else {
				b.WriteString(s.ActiveTransition.View(s.Next.View(WithAnimating()), s.Style.LipGlossStyle.Render(out)))
			}
		} else {
			if s.Prev != nil {
				b.WriteString(s.ActiveTransition.View(s.Prev.View(WithAnimating()), s.Style.LipGlossStyle.Render(out)))
			} else {
				b.WriteString(s.Style.LipGlossStyle.Render(out))
			}
		}
	} else {
		b.WriteString(s.Style.LipGlossStyle.Render(out))
	}

	return b.String()
}

func (s *Slide) First() *Slide {
	current := s
	for current.Prev != nil {
		current = current.Prev
	}
	return current
}

func (s *Slide) Last() *Slide {
	current := s
	for current.Next != nil {
		current = current.Next
	}
	return current
}
