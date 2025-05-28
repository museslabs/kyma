package config

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// MergeConfigs merges configurations in order of precedence:
// 1. Global config
// 2. Named preset (if specified)
// 3. Slide-specific config
func MergeConfigs(v *viper.Viper, slideConfig *StyleConfig) (*StyleConfig, error) {
	globalConfig := &StyleConfig{}
	if err := v.UnmarshalKey("global.style", globalConfig); err != nil {
		return nil, err
	}

	result := &StyleConfig{
		Border:      globalConfig.Border,
		BorderColor: globalConfig.BorderColor,
		Layout:      globalConfig.Layout,
		Theme:       globalConfig.Theme,
		Preset:      slideConfig.Preset,
	}

	if slideConfig.Preset != "" {
		presetConfig := &StyleConfig{}
		if err := v.UnmarshalKey("presets."+slideConfig.Preset+".style", presetConfig); err != nil {
			return nil, err
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
