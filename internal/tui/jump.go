package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Jump struct {
	numberInput string
	jumpSteps   int
	quitting    bool
	showing     bool
	width       int
	height      int
}

func NewJump() Jump {
	return Jump{
		numberInput: "",
		showing:     false,
	}
}

func (m Jump) Init() tea.Cmd {
	return nil
}

func (m Jump) Update(msg tea.Msg) (Jump, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quitting = true
			m.showing = false
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if len(m.numberInput) < 4 {
				m.numberInput += msg.String()
			}
			return m, nil
		case "backspace":
			if len(m.numberInput) > 0 {
				m.numberInput = m.numberInput[:len(m.numberInput)-1]
			}
			return m, nil
		case "h", "left":
			if m.numberInput != "" {
				if steps, err := strconv.Atoi(m.numberInput); err == nil {
					m.jumpSteps = -steps
				}
				m.quitting = true
				m.showing = false
				return m, tea.Quit
			}
			return m, nil
		case "l", "right":
			if m.numberInput != "" {
				if steps, err := strconv.Atoi(m.numberInput); err == nil {
					m.jumpSteps = steps
				}
				m.quitting = true
				m.showing = false
				return m, tea.Quit
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Jump) Show(slideView string, width, height int) string {
	displayNumber := m.numberInput
	if displayNumber == "" {
		displayNumber = "_"
	}

	statusContent := lipgloss.NewStyle().
		Background(lipgloss.Color("#3C3C3C")).
		Foreground(lipgloss.Color("#DDDDDD")).
		Padding(0, 1).
		Render(displayNumber)

	statusWidth := lipgloss.Width(statusContent)
	statusX := (width - statusWidth) / 2
	statusY := height - 1

	leftHelp := lipgloss.JoinHorizontal(
		lipgloss.Center,
		mutedStyle.Render("←/h"),
		" ",
		veryMutedStyle.Render("• previous"),
	)

	rightHelp := lipgloss.JoinHorizontal(
		lipgloss.Center,
		mutedStyle.Render("→/l"),
		" ",
		veryMutedStyle.Render("• next"),
	)

	escHelp := lipgloss.JoinHorizontal(
		lipgloss.Center,
		mutedStyle.Render("esc/ctrl+c"),
		" ",
		veryMutedStyle.Render("• cancel"),
	)

	helpText := lipgloss.JoinHorizontal(lipgloss.Center, leftHelp, "  ", rightHelp, "  ", escHelp)
	helpWidth := lipgloss.Width(helpText)
	helpX := (width - helpWidth) / 2
	helpY := height - 3

	viewWithStatus := placeOverlay(statusX, statusY, statusContent, slideView)
	return placeOverlay(helpX, helpY, helpText, viewWithStatus)
}

func (m Jump) IsShowing() bool {
	return m.showing
}

func (m Jump) SetShowing(showing bool) Jump {
	m.showing = showing
	if showing {
		m.numberInput = ""
		m.jumpSteps = 0
		m.quitting = false
	}
	return m
}

func (m Jump) Quitting() bool {
	return m.quitting
}

func (m Jump) JumpSteps() int {
	return m.jumpSteps
}
