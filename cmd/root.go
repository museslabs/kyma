package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/ploMP4/kyma/internal/tui"
	"github.com/ploMP4/kyma/internal/tui/transitions"
)

var watch bool

func init() {
	rootCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes in the input file")
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
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		root, err := parseSlides(string(data))
		if err != nil {
			return err
		}

		p := tea.NewProgram(tui.New(root), tea.WithAltScreen(), tea.WithMouseAllMotion())

		if watch {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
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
				return nil
			}

			go func() {
				var debounceTimer *time.Timer

				for {
					select {
					case event, ok := <-watcher.Events:
						if !ok {
							return
						}

						if event.Name == absPath || event.Name == filename ||
							strings.HasSuffix(event.Name, "~") ||
							strings.HasPrefix(event.Name, absPath+".") {
							if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
								if debounceTimer != nil {
									debounceTimer.Stop()
								}
								debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
									data, err := os.ReadFile(filename)
									if err != nil {
										p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "slideUp")})
										return
									}

									newRoot, err := parseSlides(string(data))
									if err != nil {
										p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "slideUp")})
										return
									}

									p.Send(tui.UpdateSlidesMsg{NewRoot: newRoot})
								})
							}
						}
					case err, ok := <-watcher.Errors:
						if !ok {
							return
						}
						p.Send(tui.UpdateSlidesMsg{NewRoot: createErrorSlide(err, "slideUp")})
					}
				}
			}()
		}

		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseSlides(data string) (*tui.Slide, error) {
	slides := strings.Split(string(data), "----\n")

	rootSlide, properties := parseSlide(slides[0])
	p, err := tui.NewProperties(properties)
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
		p, err := tui.NewProperties(properties)
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
		Data: fmt.Sprintf("# Error while updating\n\n%s\n\nIf you believe this is our fault, please open up an issue on GitHub", err.Error()),
		Properties: tui.Properties{
			Transition: transitions.Get(transition, tui.Fps),
		},
	}
}
