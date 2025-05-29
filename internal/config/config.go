package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"

	"github.com/museslabs/kyma/internal/tui/transitions"
)

const (
	configName = "kyma"
	configType = "yaml"
)

var GlobalConfig config

type config struct {
	Global  presetConfig            `mapstructure:"global"`
	Presets map[string]presetConfig `mapstructure:"presets"`
}

type presetConfig struct {
	Style      StyleConfig            `mapstructure:"style"`
	Transition transitions.Transition `mapstructure:"transition"`
}

func styleConfigDecodeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to == reflect.TypeOf(StyleConfig{}) {
			m, ok := data.(map[string]any)
			if !ok {
				return data, nil
			}
			var s StyleConfig
			if err := s.DecodeMap(m); err != nil {
				return nil, err
			}
			return s, nil
		}
		return data, nil
	}
}

func transitionDecodeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() == reflect.String &&
			to == reflect.TypeOf((*transitions.Transition)(nil)).Elem() {
			name := data.(string)
			return transitions.Get(name, transitions.Fps), nil
		}
		return data, nil
	}
}

func Load(configPath string) error {
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")

		// Check XDG_CONFIG_HOME first, then fall back to ~/.config
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			viper.AddConfigPath(filepath.Join(xdgConfigHome, "kyma"))
		}
		viper.AddConfigPath(filepath.Join(home, ".config", "kyma"))

		if err := createDefaultConfig(home); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			styleConfigDecodeHook(),
			transitionDecodeHook(),
		),
		Result:  &GlobalConfig,
		TagName: "mapstructure",
	})
	if err != nil {
		return err
	}

	if err := decoder.Decode(viper.AllSettings()); err != nil {
		return err
	}

	return nil
}

func createDefaultConfig(home string) error {
	configDir := filepath.Join(home, ".config", "kyma")
	configFile := filepath.Join(configDir, fmt.Sprintf("%s.%s", configName, configType))

	if _, err := os.Stat(configFile); err == nil {
		return nil
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	defaultConfig := `global:
  style:
    border: rounded
    border_color: "#9999CC"
    layout: center
    theme: dracula
  transition: none

presets:
  minimal:
    style:
      border: hidden
      theme: notty
    transition: none
  dark:
    style:
      border: rounded
      theme: dracula
    transition: swipeLeft
  animated:
    transition: slideUp
`

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}
