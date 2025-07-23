package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/markdown"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type Slide struct {
	Data             string
	Prev             *Slide
	Next             *Slide
	Style            config.SlideStyle
	ActiveTransition transitions.Transition
	Properties       config.Properties
	Timer            Timer

	renderer *markdown.Renderer
}

type UpdateSlidesMsg struct {
	NewRoot *Slide
}

func NewSlide(data string, props config.Properties) (*Slide, error) {
	themeName := "dark"
	if props.Style.Theme.Name != "" {
		themeName = props.Style.Theme.Name
	}

	r, err := markdown.NewRenderer(themeName, markdown.WithImageBackend(props.ImageBackend))
	if err != nil {
		return nil, err

	}

	return &Slide{
		Data:       data,
		Properties: props,
		renderer:   r,
	}, nil
}

func (s *Slide) Update() (*Slide, tea.Cmd) {
	transition, cmd := s.ActiveTransition.Update()
	s.ActiveTransition = transition

	return s, cmd
}

func (s *Slide) View(animating bool) string {
	var b strings.Builder

	out, _ := s.renderer.Render(
		s.Data,
		(s.ActiveTransition != nil && s.ActiveTransition.Animating()) || animating,
	)

	if s.ActiveTransition != nil && s.ActiveTransition.Animating() {
		direction := s.ActiveTransition.Direction()
		if direction == transitions.Backwards {
			if s.Next == nil {
				panic("backwards transition at the last slide")
			} else {
				b.WriteString(s.ActiveTransition.View(s.Next.View(true), s.Style.LipGlossStyle.Render(out)))
			}
		} else {
			if s.Prev != nil {
				b.WriteString(s.ActiveTransition.View(s.Prev.View(true), s.Style.LipGlossStyle.Render(out)))
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
