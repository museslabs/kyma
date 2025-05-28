package config

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/tui/transitions"
	"github.com/spf13/viper"
)

func TestMergeProperties(t *testing.T) {
	tests := []struct {
		name             string
		globalStyle      StyleConfig
		globalTransition transitions.Transition
		presetStyle      StyleConfig
		presetTransition transitions.Transition
		slideStyle       StyleConfig
		slideTransition  transitions.Transition
		want             Properties
	}{
		{
			name: "global config only",
			globalStyle: StyleConfig{
				Border:      getBorder("rounded"),
				BorderColor: "#9999CC",
				Layout:      mustGetLayout(t, "center"),
				Theme:       getTheme("dracula"),
			},
			globalTransition: transitions.Get("none", transitions.Fps),
			slideStyle:       StyleConfig{},
			slideTransition:  nil,
			want: Properties{
				Style: StyleConfig{
					Border:      getBorder("rounded"),
					BorderColor: "#9999CC",
					Layout:      mustGetLayout(t, "center"),
					Theme:       getTheme("dracula"),
				},
				Transition: transitions.Get("none", transitions.Fps),
			},
		},
		{
			name: "global and preset config",
			globalStyle: StyleConfig{
				Border:      getBorder("rounded"),
				BorderColor: "#9999CC",
				Layout:      mustGetLayout(t, "center"),
				Theme:       getTheme("dracula"),
			},
			globalTransition: transitions.Get("none", transitions.Fps),
			presetStyle: StyleConfig{
				Border: getBorder("hidden"),
				Theme:  getTheme("notty"),
			},
			presetTransition: transitions.Get("slideUp", transitions.Fps),
			slideStyle: StyleConfig{
				Preset: "minimal",
			},
			slideTransition: nil,
			want: Properties{
				Style: StyleConfig{
					Border:      getBorder("hidden"),
					BorderColor: "#9999CC",
					Layout:      mustGetLayout(t, "center"),
					Theme:       getTheme("notty"),
					Preset:      "minimal",
				},
				Transition: transitions.Get("slideUp", transitions.Fps),
			},
		},
		{
			name: "global, preset and slide config",
			globalStyle: StyleConfig{
				Border:      getBorder("rounded"),
				BorderColor: "#9999CC",
				Layout:      mustGetLayout(t, "center"),
				Theme:       getTheme("dracula"),
			},
			globalTransition: transitions.Get("none", transitions.Fps),
			presetStyle: StyleConfig{
				Border: getBorder("hidden"),
				Theme:  getTheme("notty"),
			},
			presetTransition: transitions.Get("slideUp", transitions.Fps),
			slideStyle: StyleConfig{
				Preset:      "minimal",
				BorderColor: "#FF0000",
				Layout:      mustGetLayout(t, "right"),
			},
			slideTransition: transitions.Get("swipeLeft", transitions.Fps),
			want: Properties{
				Style: StyleConfig{
					Border:      getBorder("hidden"),
					BorderColor: "#FF0000",
					Layout:      mustGetLayout(t, "right"),
					Theme:       getTheme("notty"),
					Preset:      "minimal",
				},
				Transition: transitions.Get("swipeLeft", transitions.Fps),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()

			// Set global style config
			v.Set("global.style.border", "rounded")
			v.Set("global.style.border_color", "#9999CC")
			v.Set("global.style.layout", "center")
			v.Set("global.style.theme", "dracula")
			v.Set("global.transition", "none")

			if tt.slideStyle.Preset != "" {
				// Set preset style config
				v.Set("presets."+tt.slideStyle.Preset+".style.border", "hidden")
				v.Set("presets."+tt.slideStyle.Preset+".style.theme", "notty")
				v.Set("presets."+tt.slideStyle.Preset+".transition", "slideUp")
			}

			got, err := MergeProperties(v, &tt.slideStyle, &tt.slideTransition)
			if err != nil {
				t.Fatalf("MergeProperties() error = %v", err)
			}

			if got.Style.Border != tt.want.Style.Border {
				t.Errorf("Border = %v, want %v", got.Style.Border, tt.want.Style.Border)
			}
			if got.Style.BorderColor != tt.want.Style.BorderColor {
				t.Errorf("BorderColor = %v, want %v", got.Style.BorderColor, tt.want.Style.BorderColor)
			}
			if got.Style.Layout.GetAlignHorizontal() != tt.want.Style.Layout.GetAlignHorizontal() {
				t.Errorf("Layout horizontal alignment = %v, want %v", got.Style.Layout.GetAlignHorizontal(), tt.want.Style.Layout.GetAlignHorizontal())
			}
			if got.Style.Theme.Name != tt.want.Style.Theme.Name {
				t.Errorf("Theme = %v, want %v", got.Style.Theme.Name, tt.want.Style.Theme.Name)
			}
			if got.Style.Preset != tt.want.Style.Preset {
				t.Errorf("Preset = %v, want %v", got.Style.Preset, tt.want.Style.Preset)
			}
			if got.Transition.Name() != tt.want.Transition.Name() {
				t.Errorf("Transition = %v, want %v", got.Transition.Name(), tt.want.Transition.Name())
			}
		})
	}
}

func mustGetLayout(t *testing.T, layout string) lipgloss.Style {
	t.Helper()
	style, err := getLayout(layout)
	if err != nil {
		t.Fatalf("failed to get layout: %v", err)
	}
	return style
}
