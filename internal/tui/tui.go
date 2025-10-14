package tui

import (
	"fmt"
	"log/slog"
	"strings"

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
	Jump    key.Binding
	Timer   key.Binding
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
	Jump: key.NewBinding(
		key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
		key.WithHelp("1-9", "jump slides"),
	),
	Timer: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle timer"),
	),
}

func style(width, height int, styleConfig config.StyleConfig) config.SlideStyle {
	return styleConfig.Apply(width, height)
}

// navigateToSlide handles the common pattern of pausing current timer, switching slides, and resuming
func (m *model) navigateToSlide(newSlide *Slide) {
	if m.slide != nil {
		m.slide.Timer = m.slide.Timer.Pause()
	}
	m.slide = newSlide
	EnsureTimerInitialized(m.slide)
	if m.slide != nil {
		m.slide.Timer = m.slide.Timer.Resume()
	}

	// Sync current slide position with speaker notes
	m.syncCurrentSlide()
}

// syncCurrentSlide broadcasts the current slide number to speaker notes clients
func (m *model) syncCurrentSlide() {
	if m.syncServer == nil {
		return
	}

	// Calculate current slide position
	slidePos := 0
	slide := m.rootSlide
	for slide != nil && slide != m.slide {
		slidePos++
		slide = slide.Next
	}

	// Broadcast slide position to all connected clients
	m.syncServer.BroadcastSlideChange(slidePos)
}

type model struct {
	width  int
	height int

	slide            *Slide
	keys             keyMap
	help             help.Model
	command          *Command
	goTo             *GoTo
	jump             *Jump
	rootSlide        *Slide
	globalTimer      Timer
	timerDisplay     TimerDisplay
	syncServer       *SyncServer
	presentationFile string
}

func New(rootSlide *Slide, presentationFile string) model {
	// Initialize timer only for the first slide
	if rootSlide != nil {
		rootSlide.Timer = NewTimer().Start()
	}

	// Create sync server for speaker notes communication
	syncServer, err := NewSyncServer()
	if err != nil {
		slog.Error("Failed to create sync server", "error", err)
		syncServer = nil
	} else {
		syncServer.Start()
		slog.Info("Sync server ready for speaker notes")
	}

	return model{
		slide:            rootSlide,
		keys:             keys,
		help:             help.New(),
		rootSlide:        rootSlide,
		globalTimer:      NewTimer().Start(),
		timerDisplay:     NewTimerDisplay(),
		syncServer:       syncServer,
		presentationFile: presentationFile,
	}
}

func (m model) Init() tea.Cmd {
	// Initial sync for speaker notes
	m.syncCurrentSlide()

	return tea.Batch(
		tea.ClearScreen,
		// tea.Tick(time.Second, func(time.Time) tea.Msg {
		// 	return TimerTickMsg{}
		// }),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		slog.Info("Key pressed", "key", keyMsg.String())
	}

	if m.command != nil && m.command.IsShowing() {
		command, cmd := m.command.Update(msg)
		m.command = &command

		if command.quitting || command.Choice() != nil {
			if command.Choice() != nil {
				m.navigateToSlide(command.Choice())
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
					m.navigateToSlide(slide)
				}
			}
			m.goTo = nil
			return m, nil
		}
		return m, cmd
	}

	if m.jump != nil && m.jump.IsShowing() {
		jump, cmd := m.jump.Update(msg)
		m.jump = &jump

		if jump.Quitting() {
			if steps := jump.JumpSteps(); steps != 0 {
				newSlide := m.slide
				if steps > 0 {
					// Jump forward
					for i := 0; i < steps && newSlide.Next != nil; i++ {
						newSlide = newSlide.Next
					}
				} else {
					// Jump backward
					for i := 0; i < -steps && newSlide.Prev != nil; i++ {
						newSlide = newSlide.Prev
					}
				}
				m.navigateToSlide(newSlide)
			}
			m.jump = nil
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
			// Clean up sync server before quitting
			if m.syncServer != nil {
				m.syncServer.Stop()
			}
			return m, tea.Quit
		} else if key.Matches(msg, m.keys.Command) {
			command := NewCommand(m.rootSlide)
			command = command.SetShowing(true)
			m.command = &command
			return m, nil
		} else if key.Matches(msg, m.keys.GoTo) {
			count := 0
			for slide := m.rootSlide; slide != nil; slide = slide.Next {
				count++
			}
			goTo := NewGoTo(count)
			goTo = goTo.SetShowing(true)
			m.goTo = &goTo
			return m, nil
		} else if key.Matches(msg, m.keys.Jump) {
			jump := NewJump()
			jump = jump.SetShowing(true)
			m.jump = &jump
			jump, cmd := jump.Update(msg)
			m.jump = &jump
			return m, cmd
		} else if key.Matches(msg, m.keys.Timer) {
			m.timerDisplay = m.timerDisplay.ToggleVisible()
			return m, nil
		} else if key.Matches(msg, m.keys.Next) {
			if m.slide.Next == nil || m.slide.ActiveTransition != nil && m.slide.ActiveTransition.Animating() {
				return m, nil
			}
			m.navigateToSlide(m.slide.Next)
			m.slide.ActiveTransition = m.slide.Properties.Transition.Start(m.width, m.height, transitions.Forwards)
			return m, transitions.Animate(transitions.Fps)
		} else if key.Matches(msg, m.keys.Prev) {
			if m.slide.Prev == nil || m.slide.ActiveTransition != nil && m.slide.ActiveTransition.Animating() {
				return m, nil
			}
			m.navigateToSlide(m.slide.Prev)
			m.slide.ActiveTransition = m.slide.
				Next.
				Properties.
				Transition.
				Opposite().
				Start(m.width, m.height, transitions.Backwards)

			return m, transitions.Animate(transitions.Fps)
		} else if key.Matches(msg, m.keys.Top) {
			m.navigateToSlide(m.slide.First())
			return m, nil
		} else if key.Matches(msg, m.keys.Bottom) {
			m.navigateToSlide(m.slide.Last())
			return m, nil
		}
	case transitions.FrameMsg:
		slide, cmd := m.slide.Update()
		m.slide = slide
		return m, cmd
		// case TimerTickMsg:
		// 	var cmd tea.Cmd
		// 	m.globalTimer, cmd = m.globalTimer.Update(msg)
		// 	return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	m.slide.Style = style(m.width, m.height, m.slide.Properties.Style)

	hasOverlay := (m.command != nil && m.command.IsShowing()) ||
		(m.goTo != nil && m.goTo.IsShowing()) ||
		(m.jump != nil && m.jump.IsShowing())

	slideView := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.slide.View(
			(m.slide.ActiveTransition != nil && m.slide.ActiveTransition.Animating()) || hasOverlay,
			m.width,
			m.height,
		),
	)

	lines := strings.Split(slideView, "\n")
	if len(lines) > m.height {
		fmt.Print("\x1b_Ga=d\x1b\\")
		return m.exceedScreenSizeView()
	}

	if m.command != nil && m.command.IsShowing() {
		return m.command.Show(slideView, m.width, m.height)
	}

	if m.goTo != nil && m.goTo.IsShowing() {
		return m.goTo.Show(slideView, m.width, m.height)
	}

	if m.jump != nil && m.jump.IsShowing() {
		return m.jump.Show(slideView, m.width, m.height)
	}

	if m.timerDisplay.IsVisible() {
		return m.timerDisplay.Show(slideView, m.width, m.height, m.globalTimer, m.slide.Timer)
	}

	return slideView
}

// GetSyncServer returns the sync server instance
func (m model) GetSyncServer() *SyncServer {
	return m.syncServer
}

func (m model) exceedScreenSizeView() string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.NewStyle().
			Width(m.width-2).
			Height(m.height-2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1")).
			Foreground(lipgloss.Color("9")).
			Bold(true).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Unable to render slide: content exceeds terminal size."),
	)
}
