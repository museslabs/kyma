package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
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

	defaultConfigDir := filepath.Join(tmpDir, ".config")
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

	configFile := filepath.Join(tmpDir, ".config", "kyma.yaml")
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
