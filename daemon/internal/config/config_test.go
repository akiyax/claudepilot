package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesNewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	SetDir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != currentVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, currentVersion)
	}
	if cfg.DaemonID == "" {
		t.Error("DaemonID is empty")
	}
	if len(cfg.DaemonID) != 36 {
		t.Errorf("DaemonID len = %d, want 36 (UUID format)", len(cfg.DaemonID))
	}

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, configFileName))
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("config file not valid JSON: %v", err)
	}
	if saved.DaemonID != cfg.DaemonID {
		t.Errorf("saved DaemonID = %s, want %s", saved.DaemonID, cfg.DaemonID)
	}
}

func TestLoadExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	SetDir(tmpDir)

	existing := Config{Version: 1, DaemonID: "test-daemon-id-1234"}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, configFileName), data, 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DaemonID != "test-daemon-id-1234" {
		t.Errorf("DaemonID = %s, want test-daemon-id-1234", cfg.DaemonID)
	}
}

func TestMigrationBackup(t *testing.T) {
	tmpDir := t.TempDir()
	SetDir(tmpDir)

	// Write a v0 config (simulating old version)
	oldConfig := map[string]any{"version": 0, "daemonId": "old-id"}
	data, _ := json.Marshal(oldConfig)
	os.WriteFile(filepath.Join(tmpDir, configFileName), data, 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != currentVersion {
		t.Errorf("Version after migration = %d, want %d", cfg.Version, currentVersion)
	}

	// Check backup was created
	backupPath := filepath.Join(tmpDir, "config.json.bak.v0")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file not created")
	}
}

func TestLoadIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	SetDir(tmpDir)

	cfg1, err := Load()
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if cfg1.DaemonID != cfg2.DaemonID {
		t.Errorf("DaemonID changed between loads: %s vs %s", cfg1.DaemonID, cfg2.DaemonID)
	}
}
