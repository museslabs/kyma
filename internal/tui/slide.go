package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/config"
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

	preRenderedFrame string
}

type UpdateSlidesMsg struct {
	NewRoot *Slide
}

func (s *Slide) Update() (*Slide, tea.Cmd) {
	transition, cmd := s.ActiveTransition.Update()
	s.ActiveTransition = transition
	s.preRenderedFrame = s.view()
	if cmd == nil {
		s.preRenderedFrame = ""
	}
	return s, cmd
}

func (s Slide) View() string {
	if s.preRenderedFrame == "" {
		return s.view()
	}
	return s.preRenderedFrame
}

func (s Slide) view() string {
	var b strings.Builder

	themeName := "dark"

	if s.Style.Theme.Name != "" {
		themeName = s.Style.Theme.Name
	}

	// Pre-process markdown to handle custom highlighting syntax
	out, err := processMarkdownWithHighlighting(s.Data, themeName)
	if err != nil {
		// If preprocessing fails, fall back to regular Glamour rendering
		out, err = glamour.Render(s.Data, themeName)
		if err != nil {
			b.WriteString("\n\n" + lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")). // Red
				Render("Error: "+err.Error()))
			return b.String()
		}
	}

	if s.ActiveTransition != nil && s.ActiveTransition.Animating() {
		direction := s.ActiveTransition.Direction()
		if direction == transitions.Backwards {
			if s.Next == nil {
				panic("backwards transition at the last slide")
			} else {
				b.WriteString(s.ActiveTransition.View(s.Next.View(), s.Style.LipGlossStyle.Render(out)))
			}
		} else {
			if s.Prev != nil {
				b.WriteString(s.ActiveTransition.View(s.Prev.View(), s.Style.LipGlossStyle.Render(out)))
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
