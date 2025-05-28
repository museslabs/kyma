package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type Slide struct {
	Data             string
	Prev             *Slide
	Next             *Slide
	Style            config.SlideStyle
	ActiveTransition transitions.Transition
	Properties       Properties

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

	out, err := glamour.Render(s.Data, themeName)
	if err != nil {
		b.WriteString("\n\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Render("Error: "+err.Error()))
		return b.String()
	}

	if s.ActiveTransition != nil && s.ActiveTransition.Animating() {
		direction := s.ActiveTransition.Direction()
		if direction == transitions.Backwards {
			b.WriteString(s.ActiveTransition.View(s.Next.View(), s.Style.LipGlossStyle.Render(out)))
		} else {
			b.WriteString(s.ActiveTransition.View(s.Prev.View(), s.Style.LipGlossStyle.Render(out)))
		}
	} else {
		b.WriteString(s.Style.LipGlossStyle.Render(out))
	}
	return b.String()
}

type Properties struct {
	Style      config.StyleConfig     `yaml:"style"`
	Transition transitions.Transition `yaml:"transition"`
}

func (p *Properties) UnmarshalYAML(bytes []byte) error {
	aux := struct {
		Style      config.StyleConfig `yaml:"style"`
		Transition string             `yaml:"transition"`
	}{}

	if err := yaml.Unmarshal(bytes, &aux); err != nil {
		return err
	}
	p.Transition = transitions.Get(aux.Transition, Fps)
	p.Style = aux.Style

	return nil
}

func NewProperties(properties string) (Properties, error) {
	if properties == "" {
		return Properties{Transition: transitions.Get("default", Fps)}, nil
	}

	var p Properties
	if err := yaml.Unmarshal([]byte(properties), &p); err != nil {
		return Properties{}, err
	}

	return p, nil
}
