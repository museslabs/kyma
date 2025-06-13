package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/logger"
	"github.com/museslabs/kyma/internal/tui"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

var (
	static     bool
	configPath string
	logPath    string
)

func init() {
	rootCmd.Flags().BoolVarP(&static, "static", "s", false, "Disable live reload (watch mode is enabled by default)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	rootCmd.Flags().StringVarP(&logPath, "log", "l", "", "Path to log file (default: ~/.config/kyma/logs/<timestamp>.kyma.log)")
	rootCmd.AddCommand(versionCmd)
}

var rootCmd = &cobra.Command{
	Use: "kyma <filename>",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}

		if filepath.Ext(args[0]) != ".md" {
			return fmt.Errorf("expected markdown file got: %v", args[0])
		}
		return nil
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logger.Load(logPath); err != nil {
			return fmt.Errorf("failed to initialize slog: %w", err)
		}

		slog.Info("Starting Kyma")

		if err := config.Load(configPath); err != nil {
			slog.Error("Failed to load config", "error", err, "config_path", configPath)
			return err
		}

		filename := args[0]
		slog.Info("Loading presentation", "filename", filename)

		data, err := os.ReadFile(filename)
		if err != nil {
			slog.Error("Failed to read presentation file", "error", err, "filename", filename)
			return err
		}

		root, err := parseSlides(string(data))
		if err != nil {
			slog.Error("Failed to parse slides", "error", err, "filename", filename)
			return err
		}

		slog.Info("Successfully parsed presentation")

		p := tea.NewProgram(tui.New(root), tea.WithAltScreen(), tea.WithMouseAllMotion())

		if !static {
			slog.Info("Starting file watcher for live reload")
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				slog.Error("Failed to create file watcher", "error", err)
				p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
				return nil
			}
			defer watcher.Close()

			absPath, err := filepath.Abs(filename)
			if err != nil {
				p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
				return nil
			}

			if err := watcher.Add(filepath.Dir(absPath)); err != nil {
				p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
			}

			if configPath != "" {
				configDir := filepath.Dir(configPath)
				if err := watcher.Add(configDir); err != nil {
					p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
				}
			} else {
				home, err := os.UserHomeDir()
				if err == nil {
					configDir := filepath.Join(home, ".config")
					if err := watcher.Add(configDir); err != nil {
						p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
					}
				}
			}

			go watchFileChanges(watcher, p, filename, absPath, configPath)
		}

		slog.Info("Starting TUI program")
		if _, err := p.Run(); err != nil {
			slog.Error("TUI program failed", "error", err)
			return err
		}

		slog.Info("Kyma session ended")
		return nil
	},
}

func watchFileChanges(
	watcher *fsnotify.Watcher,
	p *tea.Program,
	filename, absPath string,
	configPath string,
) {
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Name == absPath || event.Name == filename ||
				strings.HasSuffix(event.Name, "~") ||
				strings.HasPrefix(event.Name, absPath+".") ||
				event.Name == configPath {

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
						slog.Info("File changed, reloading presentation", "file", event.Name)

						data, err := os.ReadFile(filename)
						if err != nil {
							slog.Error("Failed to read file during reload", "error", err, "filename", filename)
							p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
							return
						}

						if err := config.Load(configPath); err != nil {
							slog.Error("Failed to reload config", "error", err, "config_path", configPath)
							p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
							return
						}

						newRoot, err := parseSlides(string(data))
						if err != nil {
							slog.Error("Failed to parse slides during reload", "error", err, "filename", filename)
							p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
							return
						}

						slog.Info("Successfully reloaded presentation")
						p.Send(tui.UpdateSlidesMsg{NewRoot: newRoot})
					})
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "none")})
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Application failed", "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseSlides(data string) (*tui.Slide, error) {
	slides := strings.Split(string(data), "----\n")

	rootSlide, properties := parseSlide(slides[0])
	p, err := config.NewProperties(properties)

	if err != nil {
		return nil, err
	}

	root := &tui.Slide{
		Data:       rootSlide,
		Properties: p,
	}

	curr := root
	for _, slide := range slides[1:] {
		slide, properties := parseSlide(slide)
		p, err := config.NewProperties(properties)
		if err != nil {
			return nil, err
		}

		curr.Next = &tui.Slide{
			Data:       slide,
			Prev:       curr,
			Properties: p,
		}
		curr = curr.Next
	}

	return root, nil
}

func parseSlide(s string) (slide, properties string) {
	slide = s

	if strings.HasPrefix(strings.TrimSpace(s), "---\n") {
		parts := strings.Split(s, "---\n")
		properties = parts[1]
		slide = parts[2]
	}

	return slide, properties
}

func createErrorSlide(err error, transition string) *tui.Slide {
	return &tui.Slide{
		Data: fmt.Sprintf(
			"# Error while updating\n\n%s\n\nIf you believe this is our fault, please open up an issue on GitHub",
			err.Error(),
		),
		Properties: config.Properties{
			Transition: transitions.Get(transition, transitions.Fps),
		},
	}
}
