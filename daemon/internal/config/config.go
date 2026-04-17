package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

const (
	configDirName    = ".claudepilot"
	configFileName   = "config.json"
	logsDirName      = "logs"
	logFileName      = "claudepilot.log"
	currentVersion   = 1
	maxLogSizeMB     = 10
	maxLogBackups    = 5
)

// Config represents the daemon's persistent configuration.
type Config struct {
	Version  int    `json:"version"`
	DaemonID string `json:"daemonId"`
	Port     int    `json:"port,omitempty"`
}

var (
	configOnce sync.Once
	configDir  string
)

// SetDir overrides the config directory (for testing only).
func SetDir(dir string) {
	configDir = dir
	configOnce = sync.Once{}
	configOnce.Do(func() {}) // consume the once so Dir() returns configDir
}

// Dir returns the configuration directory path (~/.claudepilot).
func Dir() string {
	configOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(fmt.Sprintf("failed to get home directory: %v", err))
		}
		configDir = filepath.Join(home, configDirName)
	})
	return configDir
}

// Load reads the config file, creating it with defaults if it doesn't exist.
// If the config version is older than current, migrations are applied.
func Load() (*Config, error) {
	dir := Dir()
	configPath := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// First run — create new config
			cfg := &Config{
				Version:  currentVersion,
				DaemonID: uuid.New().String(),
			}
			if err := cfg.save(); err != nil {
				return nil, fmt.Errorf("failed to create config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Run migrations if needed
	if cfg.Version < currentVersion {
		if err := migrate(&cfg, configPath, data); err != nil {
			return nil, fmt.Errorf("config migration failed: %w", err)
		}
	}

	return &cfg, nil
}

// Save persists the config to disk.
func (c *Config) save() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(dir, configFileName)
	tmpPath := configPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename config: %w", err)
	}

	return nil
}

// InitLogger sets up structured logging to file and optionally to stderr.
func InitLogger(cfg *Config, verbose bool) (*os.File, error) {
	dir := Dir()
	logsDir := filepath.Join(dir, logsDirName)

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs dir: %w", err)
	}

	logPath := filepath.Join(logsDir, logFileName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer: file always, stderr if verbose
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	fileHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: level})

	if verbose {
		stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		slog.SetDefault(slog.New(&multiHandler{handlers: []slog.Handler{fileHandler, stderrHandler}}))
	} else {
		slog.SetDefault(slog.New(fileHandler))
	}

	slog.Info("Logger initialized",
		"daemonId", cfg.DaemonID,
		"version", cfg.Version,
		"logPath", logPath,
		"verbose", verbose,
	)

	return logFile, nil
}

// multiHandler dispatches log records to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, record.Level) {
			if err := h.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
