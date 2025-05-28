package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	Style StyleConfig `mapstructure:"style"`
}

type PresetConfig struct {
	Style StyleConfig `mapstructure:"style"`
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
		v.AddConfigPath(filepath.Join(home, ".config"))

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
	configDir := filepath.Join(home, ".config")
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

presets:
  minimal:
    style:
      border: hidden
      theme: notty
  dark:
    style:
      border: rounded
      theme: dracula
`

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}
