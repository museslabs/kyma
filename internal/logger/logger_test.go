package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		logPath   string
		wantErr   bool
		checkFile bool
	}{
		{
			name:      "custom log path",
			logPath:   filepath.Join(tmpDir, "custom.log"),
			wantErr:   false,
			checkFile: true,
		},
		{
			name:      "invalid directory path",
			logPath:   filepath.Join("/invalid/nonexistent/dir", "test.log"),
			wantErr:   true,
			checkFile: false,
		},
		{
			name:      "empty log path (default)",
			logPath:   "",
			wantErr:   false,
			checkFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set HOME environment variable for default config
			oldHome := os.Getenv("HOME")
			oldXDG := os.Getenv("XDG_CONFIG_HOME")
			os.Setenv("HOME", tmpDir)
			os.Unsetenv("XDG_CONFIG_HOME")
			defer func() {
				os.Setenv("HOME", oldHome)
				if oldXDG != "" {
					os.Setenv("XDG_CONFIG_HOME", oldXDG)
				}
			}()

			err := Load(tt.logPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFile && !tt.wantErr {

				// Test that we can actually log
				slog.Info("test message", "key", "value")

				// Check that log file exists
				var logPath string
				if tt.logPath != "" {
					logPath = tt.logPath
				} else {
					defaultPath, err := getDefaultLogPath()
					if err != nil {
						t.Errorf("Failed to get default log path: %v", err)
						return
					}
					logPath = defaultPath
				}

				if _, err := os.Stat(logPath); os.IsNotExist(err) {
					t.Errorf("Log file was not created at %s", logPath)
				}
			}
		})
	}
}

func TestGetDefaultLogPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		setXDGConfig   bool
		xdgConfigValue string
		wantContains   string
	}{
		{
			name:         "default home config",
			setXDGConfig: false,
			wantContains: ".config/kyma/logs",
		},
		{
			name:           "XDG_CONFIG_HOME set",
			setXDGConfig:   true,
			xdgConfigValue: filepath.Join(tmpDir, "xdg-config"),
			wantContains:   "xdg-config/kyma/logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHome := os.Getenv("HOME")
			oldXDG := os.Getenv("XDG_CONFIG_HOME")

			os.Setenv("HOME", tmpDir)
			if tt.setXDGConfig {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigValue)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			defer func() {
				os.Setenv("HOME", oldHome)
				if oldXDG != "" {
					os.Setenv("XDG_CONFIG_HOME", oldXDG)
				} else {
					os.Unsetenv("XDG_CONFIG_HOME")
				}
			}()

			logPath, err := getDefaultLogPath()
			if err != nil {
				t.Errorf("getDefaultLogPath() error = %v", err)
				return
			}

			if !strings.Contains(logPath, tt.wantContains) {
				t.Errorf("getDefaultLogPath() = %v, want path containing %v", logPath, tt.wantContains)
			}

			if !strings.HasSuffix(logPath, ".kyma.log") {
				t.Errorf("getDefaultLogPath() = %v, want path ending with .kyma.log", logPath)
			}
		})
	}
}

func TestRotateLogFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create some test log files with different timestamps
	testFiles := []string{
		"2023-01-01_10-00-00.kyma.log",
		"2023-01-02_10-00-00.kyma.log",
		"2023-01-03_10-00-00.kyma.log",
		"2023-01-04_10-00-00.kyma.log",
		"2023-01-05_10-00-00.kyma.log",
	}

	for i, filename := range testFiles {
		filePath := filepath.Join(logsDir, filename)
		if err := os.WriteFile(filePath, []byte("test log content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}

		// Set different modification times to ensure proper sorting
		modTime := time.Now().Add(time.Duration(i-5) * time.Hour)
		if err := os.Chtimes(filePath, modTime, modTime); err != nil {
			t.Fatalf("Failed to set modification time for %s: %v", filename, err)
		}
	}

	// Test rotation
	err = rotateLogFiles(logsDir)
	if err != nil {
		t.Errorf("rotateLogFiles() error = %v", err)
	}

	// Check that only maxLogFiles-1 files remain (since we're about to create a new one)
	pattern := filepath.Join(logsDir, "*.kyma.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Errorf("Failed to glob log files: %v", err)
	}

	expectedFiles := maxLogFiles - 1
	if len(matches) != expectedFiles {
		t.Errorf("After rotation, got %d files, want %d", len(matches), expectedFiles)
	}

	// Check that the oldest files were removed (the files with earliest timestamps should be gone)
	for _, match := range matches {
		filename := filepath.Base(match)
		if filename == "2023-01-01_10-00-00.kyma.log" || filename == "2023-01-02_10-00-00.kyma.log" {
			t.Errorf("Old file %s should have been removed", filename)
		}
	}
}

func TestRotateLogFilesNoRotationNeeded(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logsDir := filepath.Join(tmpDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatalf("Failed to create logs dir: %v", err)
	}

	// Create fewer than maxLogFiles
	testFiles := []string{
		"2023-01-01_10-00-00.kyma.log",
		"2023-01-02_10-00-00.kyma.log",
	}

	for _, filename := range testFiles {
		filePath := filepath.Join(logsDir, filename)
		if err := os.WriteFile(filePath, []byte("test log content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Test rotation
	err = rotateLogFiles(logsDir)
	if err != nil {
		t.Errorf("rotateLogFiles() error = %v", err)
	}

	// Check that all files remain
	pattern := filepath.Join(logsDir, "*.kyma.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Errorf("Failed to glob log files: %v", err)
	}

	if len(matches) != len(testFiles) {
		t.Errorf("After rotation, got %d files, want %d", len(matches), len(testFiles))
	}
}

func TestLoggerFunctions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kyma-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	err = Load(logPath)
	if err != nil {
		t.Fatalf("Failed to load logger: %v", err)
	}

	// Test all logging functions
	slog.Debug("debug message", "key", "value")
	slog.Info("info message", "key", "value")
	slog.Warn("warn message", "key", "value")
	slog.Error("error message", "key", "value")

	// Check that log file was created and has content
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "debug message") {
		t.Error("Debug message not found in log")
	}
	if !strings.Contains(logContent, "info message") {
		t.Error("Info message not found in log")
	}
	if !strings.Contains(logContent, "warn message") {
		t.Error("Warn message not found in log")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("Error message not found in log")
	}
}

func TestLoggerWithNilLogger(t *testing.T) {
	// These should not panic
	slog.Debug("debug message", "key", "value")
	slog.Info("info message", "key", "value")
	slog.Warn("warn message", "key", "value")
	slog.Error("error message", "key", "value")
}
