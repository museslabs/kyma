package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/config"
)

var (
	gotoModalTitleStyle = lipgloss.NewStyle().
				MarginBottom(1)
	gotoModalInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1)
	veryMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3C3C3C"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5C5C5C"))
)

type GoTo struct {
	input    textinput.Model
	choice   int
	quitting bool
	maxSlide int
	showing  bool
}

func NewGoTo(maxSlides int) GoTo {
	ti := textinput.New()
	ti.Placeholder = "Enter slide number..."
	ti.Focus()
	ti.CharLimit = 10
	ti.Width = 20

	return GoTo{
		input:    ti,
		choice:   -1,
		maxSlide: maxSlides,
	}
}

func (m GoTo) Init() tea.Cmd {
	return textinput.Blink
}

func (m GoTo) Update(msg tea.Msg) (GoTo, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if slideNum, err := strconv.Atoi(m.input.Value()); err == nil && slideNum > 0 {
				m.choice = slideNum
			}
			m.quitting = true
			m.showing = false
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.quitting = true
			m.showing = false
			return m, tea.Quit
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m GoTo) View() string {
	title := gotoModalTitleStyle.Render("Go to Slide")

	input := gotoModalInputStyle.Render(m.input.View())

	enterHelp := lipgloss.JoinHorizontal(
		lipgloss.Center,
		mutedStyle.Render("enter"),
		" ",
		veryMutedStyle.MarginTop(1).Render("• go"),
	)

	escHelp := lipgloss.JoinHorizontal(
		lipgloss.Center,
		mutedStyle.Render("esc"),
		" ",
		veryMutedStyle.MarginTop(1).Render("• cancel"),
	)

	help := veryMutedStyle.MarginTop(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, enterHelp, "  ", escHelp),
	)

	maxInfo := mutedStyle.Render(fmt.Sprintf("(1-%d slides available)", m.maxSlide))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		input,
		help,
		maxInfo,
	)
}

func (m GoTo) Show(slideView string, width, height int) string {
	view := m.View()
	modalContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(config.DefaultBorderColor)).
		Padding(3).
		Render(view)

	_, modalWidth := getLines(modalContent)
	modalHeight := strings.Count(modalContent, "\n") + 1

	centerX := (width - modalWidth) / 2
	centerY := (height - modalHeight) / 2

	return placeOverlay(centerX, centerY, modalContent, slideView)
}

func (m GoTo) IsShowing() bool {
	return m.showing
}

func (m GoTo) SetShowing(showing bool) GoTo {
	m.showing = showing
	return m
}

func (m GoTo) Choice() int {
	return m.choice
}

func (m GoTo) Quitting() bool {
	return m.quitting
}
