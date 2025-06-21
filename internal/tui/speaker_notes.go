package tui

import (
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// ConnectionStatus represents the state of the sync connection
type ConnectionStatus int

const (
	StatusDisconnectedWaiting ConnectionStatus = iota
	StatusConnected
	StatusDisconnectedClosed
	StatusReconnecting
)

func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnectedWaiting:
		return "Disconnected - waiting for presentation to start"
	case StatusConnected:
		return "Connected"
	case StatusDisconnectedClosed:
		return "Disconnected - presentation closed"
	case StatusReconnecting:
		return "Reconnecting..."
	default:
		return "Unknown"
	}
}

type SpeakerNotesModel struct {
	width            int
	height           int
	currentSlide     int
	slides           []*Slide
	syncClient       *SyncClient
	slideChangeChan  chan int
	connectionStatus ConnectionStatus
}

type SlideChangeMsg struct {
	SlideNumber int
}

type ConnectionLostMsg struct{}

type ReconnectAttemptMsg struct{}

type ReconnectedMsg struct {
	Client *SyncClient
}

func NewSpeakerNotes(rootSlide *Slide) SpeakerNotesModel {
	// Create slides array for easier indexing
	var slides []*Slide
	slide := rootSlide
	for slide != nil {
		slides = append(slides, slide)
		slide = slide.Next
	}

	// Attempt to create a sync client to connect to the main presentation
	syncClient, err := NewSyncClient()

	var status ConnectionStatus
	if err != nil {
		slog.Warn("Failed to connect to sync server - run the main presentation first", "error", err)
		status = StatusDisconnectedWaiting
	} else {
		status = StatusConnected
	}

	// Create buffered channel for slide changes
	slideChangeChan := make(chan int)

	return SpeakerNotesModel{
		currentSlide:     0,
		slides:           slides,
		syncClient:       syncClient,
		slideChangeChan:  slideChangeChan,
		connectionStatus: status,
	}
}

func (m SpeakerNotesModel) Init() tea.Cmd {
	if m.syncClient != nil {
		// Start listening for slide changes in a goroutine
		go m.listenForSlideChangesWithReconnect()

		return tea.Batch(
			tea.ClearScreen,
			m.waitForSlideChange(),
		)
	}

	// No connection initially, start trying to reconnect
	return tea.Batch(
		tea.ClearScreen,
		m.attemptReconnect(),
	)
}

func (m SpeakerNotesModel) waitForSlideChange() tea.Cmd {
	return func() tea.Msg {
		slideNum := <-m.slideChangeChan
		return SlideChangeMsg{SlideNumber: slideNum}
	}
}

func (m SpeakerNotesModel) listenForSlideChangesWithReconnect() {
	if m.syncClient == nil {
		return
	}

	// Listen for slide changes and detect disconnection
	m.syncClient.ListenForSlideChanges(m.slideChangeChan)

	// If we reach here, the connection was lost
	m.slideChangeChan <- -1
}

func (m SpeakerNotesModel) attemptReconnect() tea.Cmd {
	return func() tea.Msg {
		syncClient, err := NewSyncClient()
		if err != nil {
			return ReconnectAttemptMsg{}
		}

		return ReconnectedMsg{Client: syncClient}
	}
}

func (m SpeakerNotesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case SlideChangeMsg:
		if msg.SlideNumber == -1 {
			m.connectionStatus = StatusDisconnectedClosed
			if m.syncClient != nil {
				m.syncClient.Close()
				m.syncClient = nil
			}
			// Start reconnection attempts
			return m, m.attemptReconnect()
		}

		if msg.SlideNumber >= 0 && msg.SlideNumber < len(m.slides) {
			m.currentSlide = msg.SlideNumber
			slog.Info("Speaker notes: slide changed", "slide", msg.SlideNumber)
		}
		// Continue waiting for more slide changes
		return m, m.waitForSlideChange()
	case ReconnectAttemptMsg:
		m.connectionStatus = StatusReconnecting
		// Wait a bit before trying again
		return m, tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
			return m.attemptReconnect()()
		})
	case ReconnectedMsg:
		m.syncClient = msg.Client
		m.connectionStatus = StatusConnected
		// Start listening for slide changes again
		go m.listenForSlideChangesWithReconnect()
		return m, m.waitForSlideChange()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.syncClient != nil {
				m.syncClient.Close()
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SpeakerNotesModel) View() string {
	if len(m.slides) == 0 {
		return "No slides available"
	}

	if m.currentSlide >= len(m.slides) {
		m.currentSlide = len(m.slides) - 1
	}

	slide := m.slides[m.currentSlide]
	notes := slide.Properties.Notes

	if notes == "" {
		notes = "No speaker notes for this slide."
	}

	headerText := fmt.Sprintf("Speaker Notes - Slide %d/%d (%s)", m.currentSlide+1, len(m.slides), m.connectionStatus)

	header := lipgloss.NewStyle().
		Bold(true).
		Padding(1).
		Foreground(lipgloss.Color("#9999CC")).
		Render(headerText)

	rendered, err := glamour.Render(notes, "dark")
	if err != nil {
		rendered = notes
	}

	notesStyle := lipgloss.NewStyle().
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		MarginTop(1)

	content := notesStyle.Render(rendered)

	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}
