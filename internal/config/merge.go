package config

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// MergeConfigs merges configurations in order of precedence:
// 1. Global config
// 2. Named preset (if specified)
// 3. Slide-specific config
func MergeConfigs(v *viper.Viper, slideConfig *StyleConfig) (*StyleConfig, error) {
	// Parse global config from viper
	globalConfigAux := struct {
		Layout      string `mapstructure:"layout"`
		Border      string `mapstructure:"border"`
		BorderColor string `mapstructure:"border_color"`
		Theme       string `mapstructure:"theme"`
		Preset      string `mapstructure:"preset"`
	}{}

	if err := v.UnmarshalKey("global.style", &globalConfigAux); err != nil {
		return nil, err
	}

	globalConfig, err := ParseStyleConfig(
		globalConfigAux.Layout,
		globalConfigAux.Border,
		globalConfigAux.BorderColor,
		globalConfigAux.Theme,
		globalConfigAux.Preset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	result := &StyleConfig{
		Border:      globalConfig.Border,
		BorderColor: globalConfig.BorderColor,
		Layout:      globalConfig.Layout,
		Theme:       globalConfig.Theme,
		Preset:      slideConfig.Preset,
	}

	if slideConfig.Preset != "" {
		presetConfigAux := struct {
			Layout      string `mapstructure:"layout"`
			Border      string `mapstructure:"border"`
			BorderColor string `mapstructure:"border_color"`
			Theme       string `mapstructure:"theme"`
			Preset      string `mapstructure:"preset"`
		}{}

		if err := v.UnmarshalKey("presets."+slideConfig.Preset+".style", &presetConfigAux); err != nil {
			return nil, err
		}

		presetConfig, err := ParseStyleConfig(
			presetConfigAux.Layout,
			presetConfigAux.Border,
			presetConfigAux.BorderColor,
			presetConfigAux.Theme,
			presetConfigAux.Preset,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preset config: %w", err)
		}

		if presetConfig.Border != (lipgloss.Border{}) {
			result.Border = presetConfig.Border
		}
		if presetConfig.BorderColor != "" {
			result.BorderColor = presetConfig.BorderColor
		}
		if presetConfig.Layout.GetAlignHorizontal() != lipgloss.Left || presetConfig.Layout.GetAlignVertical() != lipgloss.Top {
			result.Layout = presetConfig.Layout
		}
		if presetConfig.Theme != (GlamourTheme{}) {
			result.Theme = presetConfig.Theme
		}
	}

	result.Merge(*slideConfig)

	return result, nil
}
