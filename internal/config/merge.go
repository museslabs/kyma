package config

import (
	"github.com/museslabs/kyma/internal/tui"
	"github.com/spf13/viper"
)

// MergeConfigs merges configurations in order of precedence:
// 1. Global config
// 2. Named preset (if specified)
// 3. Slide-specific config
func MergeConfigs(v *viper.Viper, slideConfig *tui.StyleConfig) (*tui.StyleConfig, error) {
	globalConfig := &StyleConfig{}
	if err := v.UnmarshalKey("global.style", globalConfig); err != nil {
		return nil, err
	}

	layout, err := tui.GetLayout(globalConfig.Layout)
	if err != nil {
		return nil, err
	}

	result := &tui.StyleConfig{
		Border:      tui.GetBorder(globalConfig.Border),
		BorderColor: globalConfig.BorderColor,
		Layout:      layout,
		Theme:       tui.GetTheme(globalConfig.Theme),
	}

	if slideConfig.Preset != "" {
		presetConfig := &StyleConfig{}
		if err := v.UnmarshalKey("presets."+slideConfig.Preset+".style", presetConfig); err != nil {
			return nil, err
		}

		if presetConfig.Border != "" {
			result.Border = tui.GetBorder(presetConfig.Border)
		}
		if presetConfig.BorderColor != "" {
			result.BorderColor = presetConfig.BorderColor
		}
		if presetConfig.Layout != "" {
			layout, err := tui.GetLayout(presetConfig.Layout)
			if err != nil {
				return nil, err
			}
			result.Layout = layout
		}
		if presetConfig.Theme != "" {
			result.Theme = tui.GetTheme(presetConfig.Theme)
		}
	}

	result.Merge(*slideConfig)

	return result, nil
}
