package cmd

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/museslabs/kyma/docs"
	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/logger"
	"github.com/museslabs/kyma/internal/tui"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Start documentation as a presentation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logger.Load(logPath); err != nil {
			return fmt.Errorf("failed to initialize slog: %w", err)
		}

		slog.Info("Starting Kyma Docs")

		if err := config.Load(configPath); err != nil {
			slog.Error("Failed to load config", "error", err, "config_path", configPath)
			return err
		}

		data, err := docs.FS.ReadFile("presentation.md")
		if err != nil {
			slog.Error(
				"Failed to read presentation file",
				"error",
				err,
				"filename",
				"presentation.md",
			)
			return err
		}

		root, err := parseSlides(string(data))
		if err != nil {
			slog.Error("Failed to parse slides", "error", err, "filename", "presentation.md")
			return err
		}

		slog.Info("Successfully parsed presentation")

		p := tea.NewProgram(
			tui.New(root, "presentation.md"),
			tea.WithAltScreen(),
			tea.WithMouseAllMotion(),
		)

		slog.Info("Starting TUI program")
		if _, err := p.Run(); err != nil {
			slog.Error("TUI program failed", "error", err)
			return err
		}

		slog.Info("Kyma docs session ended")
		return nil
	},
}
