package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// migrate applies configuration migrations from oldVersion to currentVersion.
func migrate(cfg *Config, configPath string, originalData []byte) error {
	dir := Dir()

	// Backup original file before migration
	backupPath := filepath.Join(dir, fmt.Sprintf("config.json.bak.v%d", cfg.Version))
	if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
		return fmt.Errorf("failed to backup config: %w", err)
	}

	// Apply migration chain
	// v1 → v2: (future migrations go here)
	// if cfg.Version < 2 { ... }

	cfg.Version = currentVersion

	// Save migrated config
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write migrated config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename migrated config: %w", err)
	}

	return nil
}
