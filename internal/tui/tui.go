package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type keyMap struct {
	Quit    key.Binding
	Next    key.Binding
	Prev    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Command key.Binding
	GoTo    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return nil
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q, esc, ctrl+c", "quit"),
	),
	Next: key.NewBinding(
		key.WithKeys("right", "l", " "),
		key.WithHelp(">, l, <SPC>", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("<, h", "previous"),
	),
	Top: key.NewBinding(
		key.WithKeys("home", "shift+up", "0"),
		key.WithHelp("home, shift+up, 0", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("end", "shift+down", "$"),
		key.WithHelp("end, shift+down, $", "bottom"),
	),
	Command: key.NewBinding(
		key.WithKeys("/", "p"),
		key.WithHelp("/, p", "command palette"),
	),
	GoTo: key.NewBinding(
		key.WithKeys("g", ":"),
		key.WithHelp("g, :", "go to slide"),
	),
}

func style(width, height int, styleConfig config.StyleConfig) config.SlideStyle {
	return styleConfig.Apply(width, height)
}

type model struct {
	width  int
	height int

	slide     *Slide
	keys      keyMap
	help      help.Model
	command   *Command
	goTo      *GoTo
	rootSlide *Slide
}

func New(rootSlide *Slide) model {
	return model{
		slide:     rootSlide,
		keys:      keys,
		help:      help.New(),
		rootSlide: rootSlide,
	}
}

func (m model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.command != nil && m.command.IsShowing() {
		command, cmd := m.command.Update(msg)
		m.command = &command

		if command.quitting || command.Choice() != nil {
			if command.Choice() != nil {
				m.slide = command.Choice()
			}
			m.command = nil
			return m, nil
		}
		return m, cmd
	}

	if m.goTo != nil && m.goTo.IsShowing() {
		goTo, cmd := m.goTo.Update(msg)
		m.goTo = &goTo

		if goTo.Quitting() {
			if choice := goTo.Choice(); choice > 0 {
				// Find the slide at the specified position
				slide := m.rootSlide
				for i := 1; i < choice && slide != nil; i++ {
					slide = slide.Next
				}
				if slide != nil {
					m.slide = slide
				}
			}
			m.goTo = nil
			return m, nil
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case UpdateSlidesMsg:
		// Find current position in the slide list
		currentPosition := 0
		for currentSlide := m.slide; currentSlide != nil && currentSlide.Prev != nil; currentSlide = currentSlide.Prev {
			currentPosition++
		}

		// Update root and navigate to the same position in the new list
		m.slide = msg.NewRoot
		m.rootSlide = msg.NewRoot
		for i := 0; i < currentPosition && m.slide != nil; i++ {
			m.slide = m.slide.Next
		}

		// Reset state for all slides in the new list
		for currentSlide := m.slide; currentSlide != nil; currentSlide = currentSlide.Next {
			currentSlide.ActiveTransition = nil
			currentSlide.Style = style(m.width, m.height, currentSlide.Properties.Style)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		slide := m.slide
		for slide != nil {
			slide.Style = style(m.width, m.height, slide.Properties.Style)
			slide = slide.Next
		}
		return m, nil
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		} else if key.Matches(msg, m.keys.Command) {
			command := NewCommand(m.rootSlide)
			command = command.SetShowing(true)
			m.command = &command
			return m, nil
		} else if key.Matches(msg, m.keys.GoTo) {
			// Count total slides
			count := 0
			for slide := m.rootSlide; slide != nil; slide = slide.Next {
				count++
			}
			goTo := NewGoTo(count)
			goTo = goTo.SetShowing(true)
			m.goTo = &goTo
			return m, nil
		} else if key.Matches(msg, m.keys.Next) {
			if m.slide.Next == nil || m.slide.ActiveTransition != nil && m.slide.ActiveTransition.Animating() {
				return m, nil
			}
			m.slide = m.slide.Next
			m.slide.ActiveTransition = m.slide.Properties.Transition.Start(m.width, m.height, transitions.Forwards)
			return m, transitions.Animate(transitions.Fps)
		} else if key.Matches(msg, m.keys.Prev) {
			if m.slide.Prev == nil || m.slide.ActiveTransition != nil && m.slide.ActiveTransition.Animating() {
				return m, nil
			}
			m.slide = m.slide.Prev
			m.slide.ActiveTransition = m.slide.
				Next.
				Properties.
				Transition.
				Opposite().
				Start(m.width, m.height, transitions.Backwards)

			return m, transitions.Animate(transitions.Fps)
		} else if key.Matches(msg, m.keys.Top) {
			m.slide = m.slide.First()
			return m, nil
		} else if key.Matches(msg, m.keys.Bottom) {
			m.slide = m.slide.Last()
			return m, nil
		}
	case transitions.FrameMsg:
		slide, cmd := m.slide.Update()
		m.slide = slide
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	m.slide.Style = style(m.width, m.height, m.slide.Properties.Style)

	slideView := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.slide.View(),
	)

	if m.command != nil && m.command.IsShowing() {
		return m.command.Show(slideView, m.width, m.height)
	}

	if m.goTo != nil && m.goTo.IsShowing() {
		return m.goTo.Show(slideView, m.width, m.height)
	}

	return slideView
}
