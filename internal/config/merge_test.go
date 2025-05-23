package config

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/tui"
	"github.com/spf13/viper"
)

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name        string
		globalStyle StyleConfig
		presetStyle StyleConfig
		slideStyle  tui.StyleConfig
		want        tui.StyleConfig
	}{
		{
			name: "global config only",
			globalStyle: StyleConfig{
				Border:      "rounded",
				BorderColor: "#9999CC",
				Layout:      "center",
				Theme:       "dracula",
			},
			slideStyle: tui.StyleConfig{},
			want: tui.StyleConfig{
				Border:      tui.GetBorder("rounded"),
				BorderColor: "#9999CC",
				Layout:      mustGetLayout(t, "center"),
				Theme:       tui.GetTheme("dracula"),
			},
		},
		{
			name: "global and preset config",
			globalStyle: StyleConfig{
				Border:      "rounded",
				BorderColor: "#9999CC",
				Layout:      "center",
				Theme:       "dracula",
			},
			presetStyle: StyleConfig{
				Border: "hidden",
				Theme:  "notty",
			},
			slideStyle: tui.StyleConfig{
				Preset: "minimal",
			},
			want: tui.StyleConfig{
				Border:      tui.GetBorder("hidden"),
				BorderColor: "#9999CC",
				Layout:      mustGetLayout(t, "center"),
				Theme:       tui.GetTheme("notty"),
				Preset:      "minimal",
			},
		},
		{
			name: "global, preset and slide config",
			globalStyle: StyleConfig{
				Border:      "rounded",
				BorderColor: "#9999CC",
				Layout:      "center",
				Theme:       "dracula",
			},
			presetStyle: StyleConfig{
				Border: "hidden",
				Theme:  "notty",
			},
			slideStyle: tui.StyleConfig{
				Preset:      "minimal",
				BorderColor: "#FF0000",
				Layout:      mustGetLayout(t, "right"),
			},
			want: tui.StyleConfig{
				Border:      tui.GetBorder("hidden"),
				BorderColor: "#FF0000",
				Layout:      mustGetLayout(t, "right"),
				Theme:       tui.GetTheme("notty"),
				Preset:      "minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.Set("global.style", tt.globalStyle)
			if tt.slideStyle.Preset != "" {
				v.Set("presets."+tt.slideStyle.Preset+".style", tt.presetStyle)
			}

			got, err := MergeConfigs(v, &tt.slideStyle)
			if err != nil {
				t.Fatalf("MergeConfigs() error = %v", err)
			}

			if got.Border != tt.want.Border {
				t.Errorf("Border = %v, want %v", got.Border, tt.want.Border)
			}
			if got.BorderColor != tt.want.BorderColor {
				t.Errorf("BorderColor = %v, want %v", got.BorderColor, tt.want.BorderColor)
			}
			if got.Layout.GetAlignHorizontal() != tt.want.Layout.GetAlignHorizontal() {
				t.Errorf("Layout horizontal alignment = %v, want %v", got.Layout.GetAlignHorizontal(), tt.want.Layout.GetAlignHorizontal())
			}
			if got.Theme.Name != tt.want.Theme.Name {
				t.Errorf("Theme = %v, want %v", got.Theme.Name, tt.want.Theme.Name)
			}
			if got.Preset != tt.want.Preset {
				t.Errorf("Preset = %v, want %v", got.Preset, tt.want.Preset)
			}
		})
	}
}

func mustGetLayout(t *testing.T, layout string) lipgloss.Style {
	t.Helper()
	style, err := tui.GetLayout(layout)
	if err != nil {
		t.Fatalf("failed to get layout: %v", err)
	}
	return style
}
