package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/museslabs/kyma/internal/tui/transitions"
	"github.com/spf13/viper"
)

const (
	configName = "kyma"
	configType = "yaml"
)

var configPath string

func SetConfigPath(path string) {
	configPath = path
}

type Config struct {
	Global  GlobalConfig            `mapstructure:"global"`
	Presets map[string]PresetConfig `mapstructure:"presets"`
}

type GlobalConfig struct {
	Style      StyleConfig            `mapstructure:"style"`
	Transition transitions.Transition `mapstructure:"transition"`
}

type PresetConfig struct {
	Style      StyleConfig            `mapstructure:"style"`
	Transition transitions.Transition `mapstructure:"transition"`
}

func Initialize(configPath string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(configName)
	v.SetConfigType(configType)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")

		// Check XDG_CONFIG_HOME first, then fall back to ~/.config
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			v.AddConfigPath(filepath.Join(xdgConfigHome, "kyma"))
		}
		v.AddConfigPath(filepath.Join(home, ".config", "kyma"))

		if err := createDefaultConfig(home); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return v, nil
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
