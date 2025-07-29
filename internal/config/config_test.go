package config

import (
	"os"
	"path/filepath"
	"testing"

	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/spf13/viper"

	"github.com/museslabs/kyma/internal/tui/transitions"
)

func TestLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testConfig := `global:
  style:
    border: rounded
    border_color: "#FF0000"
    layout: center
    theme: dracula

presets:
  test:
    style:
      border: hidden
      theme: notty
`
	testConfigPath := filepath.Join(tmpDir, "kyma.yaml")
	if err := os.WriteFile(testConfigPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	defaultConfigDir := filepath.Join(tmpDir, ".config", "kyma")
	if err := os.MkdirAll(defaultConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create .config directory: %v", err)
	}

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
		checkVals  bool
	}{
		{
			name:       "valid config path",
			configPath: testConfigPath,
			wantErr:    false,
			checkVals:  true,
		},
		{
			name:       "invalid config path",
			configPath: filepath.Join(tmpDir, "nonexistent.yaml"),
			wantErr:    true,
			checkVals:  false,
		},
		{
			name:       "empty config path",
			configPath: "",
			wantErr:    false,
			checkVals:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set HOME environment variable for default config
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", oldHome)

			err := Load(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkVals {
				if tt.configPath == testConfigPath {
					if viper.GetString("global.style.border") != "rounded" {
						t.Errorf(
							"global.style.border = %v, want %v",
							viper.GetString("global.style.border"),
							"rounded",
						)
					}
					if viper.GetString("global.style.border_color") != "#FF0000" {
						t.Errorf(
							"global.style.border_color = %v, want %v",
							viper.GetString("global.style.border_color"),
							"#FF0000",
						)
					}
					if viper.GetString("global.style.layout") != "center" {
						t.Errorf(
							"global.style.layout = %v, want %v",
							viper.GetString("global.style.layout"),
							"center",
						)
					}
					if viper.GetString("global.style.theme") != "dracula" {
						t.Errorf(
							"global.style.theme = %v, want %v",
							viper.GetString("global.style.theme"),
							"dracula",
						)
					}

					if viper.GetString("presets.test.style.border") != "hidden" {
						t.Errorf(
							"presets.test.style.border = %v, want %v",
							viper.GetString("presets.test.style.border"),
							"hidden",
						)
					}
					if viper.GetString("presets.test.style.theme") != "notty" {
						t.Errorf(
							"presets.test.style.theme = %v, want %v",
							viper.GetString("presets.test.style.theme"),
							"notty",
						)
					}
				} else {
					if viper.GetString("global.style.border") != "rounded" {
						t.Errorf("global.style.border = %v, want %v", viper.GetString("global.style.border"), "rounded")
					}
					if viper.GetString("global.style.border_color") != "#9999CC" {
						t.Errorf("global.style.border_color = %v, want %v", viper.GetString("global.style.border_color"), "#9999CC")
					}
					if viper.GetString("global.style.layout") != "center" {
						t.Errorf("global.style.layout = %v, want %v", viper.GetString("global.style.layout"), "center")
					}
					if viper.GetString("global.style.theme") != "dracula" {
						t.Errorf("global.style.theme = %v, want %v", viper.GetString("global.style.theme"), "dracula")
					}

					if viper.GetString("presets.minimal.style.border") != "hidden" {
						t.Errorf("presets.minimal.style.border = %v, want %v", viper.GetString("presets.minimal.style.border"), "hidden")
					}
					if viper.GetString("presets.minimal.style.theme") != "notty" {
						t.Errorf("presets.minimal.style.theme = %v, want %v", viper.GetString("presets.minimal.style.theme"), "notty")
					}
					if viper.GetString("presets.dark.style.border") != "rounded" {
						t.Errorf("presets.dark.style.border = %v, want %v", viper.GetString("presets.dark.style.border"), "rounded")
					}
					if viper.GetString("presets.dark.style.theme") != "dracula" {
						t.Errorf("presets.dark.style.theme = %v, want %v", viper.GetString("presets.dark.style.theme"), "dracula")
					}
				}
			}
		})
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = createDefaultConfig(tmpDir)
	if err != nil {
		t.Fatalf("createDefaultConfig() error = %v", err)
	}

	configFile := filepath.Join(tmpDir, ".config", "kyma", "kyma.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Default config file was not created")
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read default config: %v", err)
	}

	if viper.GetString("global.style.border") != "rounded" {
		t.Errorf(
			"global.style.border = %v, want %v",
			viper.GetString("global.style.border"),
			"rounded",
		)
	}
	if viper.GetString("global.style.border_color") != "#9999CC" {
		t.Errorf(
			"global.style.border_color = %v, want %v",
			viper.GetString("global.style.border_color"),
			"#9999CC",
		)
	}
	if viper.GetString("global.style.layout") != "center" {
		t.Errorf(
			"global.style.layout = %v, want %v",
			viper.GetString("global.style.layout"),
			"center",
		)
	}
	if viper.GetString("global.style.theme") != "dracula" {
		t.Errorf(
			"global.style.theme = %v, want %v",
			viper.GetString("global.style.theme"),
			"dracula",
		)
	}

	if viper.GetString("presets.minimal.style.border") != "hidden" {
		t.Errorf(
			"presets.minimal.style.border = %v, want %v",
			viper.GetString("presets.minimal.style.border"),
			"hidden",
		)
	}
	if viper.GetString("presets.minimal.style.theme") != "notty" {
		t.Errorf(
			"presets.minimal.style.theme = %v, want %v",
			viper.GetString("presets.minimal.style.theme"),
			"notty",
		)
	}
	if viper.GetString("presets.dark.style.border") != "rounded" {
		t.Errorf(
			"presets.dark.style.border = %v, want %v",
			viper.GetString("presets.dark.style.border"),
			"rounded",
		)
	}
	if viper.GetString("presets.dark.style.theme") != "dracula" {
		t.Errorf(
			"presets.dark.style.theme = %v, want %v",
			viper.GetString("presets.dark.style.theme"),
			"dracula",
		)
	}
}

func TestPrecedence(t *testing.T) {
	tmpDir := t.TempDir()

	testConfig := `global:
  style:
    border: rounded
    border_color: "#FF0000"
    layout: center
    theme: dark

presets:
  test:
    style:
      border: hidden
      theme: notty
      border_color: "#fff"
      layout: center
`
	testConfigPath := filepath.Join(tmpDir, "kyma.yaml")
	if err := os.WriteFile(testConfigPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	if err := Load(testConfigPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		name       string
		properties string
		want       Properties
	}{
		{
			name: "slide properties should override global ones",
			properties: `style:
border: hidden
border_color: "#000"
layout: left
theme: dracula`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Left, lipgloss.Left)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.HiddenBorder()
						return &b
					}(),
					BorderColor: "#000",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["dracula"],
						Name:  "dracula",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name: "border from default styles",
			properties: `style:
border_color: "#000"
layout: left
theme: dracula`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Left, lipgloss.Left)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.RoundedBorder()
						return &b
					}(),
					BorderColor: "#000",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["dracula"],
						Name:  "dracula",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name: "border color from default styles",
			properties: `style:
border: hidden
layout: left
theme: dracula`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Left, lipgloss.Left)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.HiddenBorder()
						return &b
					}(),
					BorderColor: "#FF0000",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["dracula"],
						Name:  "dracula",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name: "layout from default styles",
			properties: `style:
border: hidden
border_color: "#000"
theme: dracula`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.HiddenBorder()
						return &b
					}(),
					BorderColor: "#000",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["dracula"],
						Name:  "dracula",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name: "theme from default styles",
			properties: `style:
border: hidden
border_color: "#000"
layout: left`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Left, lipgloss.Left)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.HiddenBorder()
						return &b
					}(),
					BorderColor: "#000",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["dark"],
						Name:  "dark",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name:       "use a preset",
			properties: `preset: test`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.HiddenBorder()
						return &b
					}(),
					BorderColor: "#fff",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["notty"],
						Name:  "notty",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
		{
			name: "use a preset and override border",
			properties: `preset: test
style:
  border: rounded`,
			want: Properties{
				Title: "",
				Style: StyleConfig{
					Layout: func() *lipgloss.Style {
						s := lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)
						return &s
					}(),
					Border: func() *lipgloss.Border {
						b := lipgloss.RoundedBorder()
						return &b
					}(),
					BorderColor: "#fff",
					Theme: &GlamourTheme{
						Style: *glamourStyles.DefaultStyles["notty"],
						Name:  "notty",
					},
				},
				Transition:   transitions.Get("none", transitions.Fps),
				Notes:        "",
				ImageBackend: "chafa",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Properties
			if err := yaml.Unmarshal([]byte(tt.properties), &p); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			if p.Title != tt.want.Title {
				t.Errorf("p.Title = %s, want = %s", p.Title, tt.want.Title)
			}

			if p.Transition != tt.want.Transition {
				t.Errorf("p.Transition = %s, want = %s", p.Transition, tt.want.Transition)
			}

			if p.Notes != tt.want.Notes {
				t.Errorf("p.Notes = %s, want = %s", p.Notes, tt.want.Notes)
			}

			if p.ImageBackend != tt.want.ImageBackend {
				t.Errorf("p.ImageBackend = %s, want = %s", p.ImageBackend, tt.want.ImageBackend)
			}

			if p.Style.BorderColor != tt.want.Style.BorderColor {
				t.Errorf(
					"p.Style.BorderColor = %s, want = %s",
					p.Style.BorderColor,
					tt.want.Style.BorderColor,
				)
			}

			if *p.Style.Border != *tt.want.Style.Border {
				t.Errorf("p.Style.Border = %v, want = %v", p.Style.Border, tt.want.Style.Border)
			}

			if *p.Style.Theme != *tt.want.Style.Theme {
				t.Errorf("p.Style.Theme = %v, want = %v", p.Style.Theme, tt.want.Style.Theme)
			}

			if p.Style.Layout.GetAlignHorizontal() != tt.want.Style.Layout.GetAlignHorizontal() ||
				p.Style.Layout.GetAlignVertical() != tt.want.Style.Layout.GetAlignVertical() {
				t.Errorf(
					"p.Style.Layout = %f, %f, want = %f, %f",
					p.Style.Layout.GetAlignHorizontal(),
					p.Style.Layout.GetAlignVertical(),
					tt.want.Style.Layout.GetAlignHorizontal(),
					tt.want.Style.Layout.GetAlignVertical(),
				)
			}
		})
	}
}
