package config

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/museslabs/kyma/internal/tui/transitions"
	"github.com/spf13/viper"
)

// MergeProperties merges style and transition configurations in order of precedence:
// 1. Global config
// 2. Named preset (if specified)
// 3. Slide-specific config
func MergeProperties(v *viper.Viper, slideStyle *StyleConfig, slideTransition *transitions.Transition) (*Properties, error) {
	// Parse global config from viper
	globalConfigAux := struct {
		Style struct {
			Layout      string `mapstructure:"layout"`
			Border      string `mapstructure:"border"`
			BorderColor string `mapstructure:"border_color"`
			Theme       string `mapstructure:"theme"`
			Preset      string `mapstructure:"preset"`
		} `mapstructure:"style"`
		Transition string `mapstructure:"transition"`
	}{}

	if err := v.UnmarshalKey("global", &globalConfigAux); err != nil {
		return nil, err
	}

	globalStyle, err := parseStyleConfig(
		globalConfigAux.Style.Layout,
		globalConfigAux.Style.Border,
		globalConfigAux.Style.BorderColor,
		globalConfigAux.Style.Theme,
		globalConfigAux.Style.Preset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	result := &Properties{
		Style: StyleConfig{
			Border:      globalStyle.Border,
			BorderColor: globalStyle.BorderColor,
			Layout:      globalStyle.Layout,
			Theme:       globalStyle.Theme,
			Preset:      slideStyle.Preset,
		},
		Transition: transitions.Get(globalConfigAux.Transition, transitions.Fps),
	}

	if slideStyle.Preset != "" {
		presetConfigAux := struct {
			Style struct {
				Layout      string `mapstructure:"layout"`
				Border      string `mapstructure:"border"`
				BorderColor string `mapstructure:"border_color"`
				Theme       string `mapstructure:"theme"`
				Preset      string `mapstructure:"preset"`
			} `mapstructure:"style"`
			Transition string `mapstructure:"transition"`
		}{}

		if err := v.UnmarshalKey("presets."+slideStyle.Preset, &presetConfigAux); err != nil {
			return nil, err
		}

		presetStyle, err := parseStyleConfig(
			presetConfigAux.Style.Layout,
			presetConfigAux.Style.Border,
			presetConfigAux.Style.BorderColor,
			presetConfigAux.Style.Theme,
			presetConfigAux.Style.Preset,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preset config: %w", err)
		}

		if presetStyle.Border != (lipgloss.Border{}) {
			result.Style.Border = presetStyle.Border
		}
		if presetStyle.BorderColor != "" {
			result.Style.BorderColor = presetStyle.BorderColor
		}
		if presetStyle.Layout.GetAlignHorizontal() != lipgloss.Left || presetStyle.Layout.GetAlignVertical() != lipgloss.Top {
			result.Style.Layout = presetStyle.Layout
		}
		if presetStyle.Theme != (GlamourTheme{}) {
			result.Style.Theme = presetStyle.Theme
		}

		if presetConfigAux.Transition != "" {
			result.Transition = transitions.Get(presetConfigAux.Transition, transitions.Fps)
		}
	}

	result.Merge(Properties{
		Style:      *slideStyle,
		Transition: *slideTransition,
	})

	return result, nil
}
