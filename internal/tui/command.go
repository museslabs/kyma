package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/config"
)

var (
	titleStyle = lipgloss.NewStyle()
	itemStyle  = lipgloss.NewStyle().
			PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(config.DefaultBorderColor))
	paginationStyle = list.DefaultStyles().PaginationStyle.
			PaddingLeft(2)
	helpStyle = list.DefaultStyles().HelpStyle.
			PaddingLeft(2).
			PaddingBottom(2)
	quitTextStyle = lipgloss.NewStyle().
			Margin(0, 0, 0, 2)
)

type SlideItem struct {
	slide  *Slide
	title  string
	number int
}

func (s SlideItem) FilterValue() string { return s.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func renderItem(str string, isSelected bool) string {
	if isSelected {
		return selectedItemStyle.Render("> " + str)
	}
	return itemStyle.Render(str)
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SlideItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", i.number, i.title)
	fmt.Fprint(w, renderItem(str, index == m.Index()))
}

type Command struct {
	list     list.Model
	choice   *Slide
	quitting bool
	showing  bool
}

func NewCommand(rootSlide *Slide) Command {
	items := []list.Item{}

	current := rootSlide
	slideNumber := 1

	for current != nil {
		title := current.Properties.Title
		if title == "" {
			title = fmt.Sprintf("#%d", slideNumber)
		}

		items = append(items, SlideItem{
			slide:  current,
			title:  title,
			number: slideNumber,
		})

		current = current.Next
		slideNumber++
	}

	const modalWidth = 90
	const listHeight = 10

	l := list.New(items, itemDelegate{}, modalWidth, listHeight)
	l.Title = "Go to Slide"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	l.Styles.PaginationStyle = paginationStyle
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle

	l.Styles.NoItems = lipgloss.NewStyle().
		MarginLeft(0).
		Foreground(lipgloss.Color("240"))

	l.KeyMap.Filter = key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	)

	return Command{list: l, showing: false}
}

func (m Command) Init() tea.Cmd {
	return nil
}

func (m Command) Update(msg tea.Msg) (Command, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			// If currently filtering, let the list handle it (clear filter)
			// Otherwise, quit the modal
			if m.list.FilterState() == list.Filtering || m.list.FilterState() == list.FilterApplied {
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "enter":
			if item := m.list.SelectedItem(); item != nil {
				if i, ok := item.(SlideItem); ok {
					m.choice = i.slide
				}
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Command) View() string {
	if m.choice != nil {
		title := m.choice.Properties.Title
		if title == "" {
			title = "selected slide"
		}
		return quitTextStyle.Render(fmt.Sprintf("Navigating to: %s", title))
	}
	if m.quitting {
		return quitTextStyle.Render("Cancelled.")
	}

	content := lipgloss.Place(
		90,
		15,
		lipgloss.Left,
		lipgloss.Center,
		m.list.View(),
	)

	return "\n" + content
}

func (m Command) Show(slideView string, width, height int) string {
	view := m.View()
	modalContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(90).
		Height(15).
		Padding(0, 4, 0, 4).
		BorderForeground(lipgloss.Color(config.DefaultBorderColor)).
		Render(view)

	_, modalWidth := getLines(modalContent)
	modalHeight := strings.Count(modalContent, "\n") + 1

	centerX := (width - modalWidth) / 2
	centerY := (height - modalHeight) / 2

	return placeOverlay(centerX, centerY, modalContent, slideView)
}

func (m Command) Choice() *Slide {
	return m.choice
}

func (m Command) IsShowing() bool {
	return m.showing
}

func (m Command) SetShowing(showing bool) Command {
	m.showing = showing
	return m
}

type OpenCommandMsg struct{}
type CloseCommandMsg struct {
	SelectedSlide *Slide
}
