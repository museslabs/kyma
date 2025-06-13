package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var Logger *slog.Logger

const maxLogFiles = 3

func Load(logPath string) error {
	var logFilePath string
	var err error

	if logPath != "" {
		logFilePath = logPath
	} else {
		logFilePath, err = getDefaultLogPath()
		if err != nil {
			return fmt.Errorf("failed to get default log path: %w", err)
		}
	}

	return initLogger(logFilePath)
}

func initLogger(logPath string) error {
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	defaultLogPath, err := getDefaultLogPath()
	if err == nil && filepath.Dir(defaultLogPath) == logDir {
		if err := rotateLogFiles(logDir); err != nil {
			return fmt.Errorf("failed to rotate log files: %w", err)
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	Logger = slog.New(handler)

	Info("Logger initialized", "log_file", logPath, "format", "json")
	return nil
}

func rotateLogFiles(logDir string) error {
	pattern := filepath.Join(logDir, "*.kyma.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find log files: %w", err)
	}

	if len(matches) < maxLogFiles {
		return nil
	}

	type logFile struct {
		path    string
		modTime time.Time
	}

	var logFiles []logFile
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		logFiles = append(logFiles, logFile{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.Before(logFiles[j].modTime)
	})

	filesToRemove := len(logFiles) - (maxLogFiles - 1)
	for i := range filesToRemove {
		if err := os.Remove(logFiles[i].path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old log file %s: %v\n", logFiles[i].path, err)
		}
	}

	return nil
}

func getDefaultLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Check XDG_CONFIG_HOME first, then fall back to ~/.config
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	var configDir string
	if xdgConfigHome != "" {
		configDir = filepath.Join(xdgConfigHome, "kyma")
	} else {
		configDir = filepath.Join(home, ".config", "kyma")
	}

	logsDir := filepath.Join(configDir, "logs")
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := fmt.Sprintf("%s.kyma.log", timestamp)

	return filepath.Join(logsDir, logFile), nil
}

func Debug(msg string, keyvals ...any) {
	if Logger != nil {
		Logger.Debug(msg, keyvals...)
	}
}

func Info(msg string, keyvals ...any) {
	if Logger != nil {
		Logger.Info(msg, keyvals...)
	}
}

func Warn(msg string, keyvals ...any) {
	if Logger != nil {
		Logger.Warn(msg, keyvals...)
	}
}

func Error(msg string, keyvals ...any) {
	if Logger != nil {
		Logger.Error(msg, keyvals...)
	}
}
