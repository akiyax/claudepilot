package provider

import (
	"path/filepath"
	"testing"
)

func TestManagerCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithDir error: %v", err)
	}

	// List should contain default
	providers := mgr.List()
	if len(providers) != 1 {
		t.Fatalf("initial provider count = %d, want 1", len(providers))
	}
	if providers[0].Name != "default" {
		t.Errorf("provider name = %q, want %q", providers[0].Name, "default")
	}
	if !providers[0].IsDefault {
		t.Error("default provider should have IsDefault=true")
	}

	// Add custom provider
	err = mgr.Add(ProviderConfig{
		Name:    "custom-anthropic",
		APIKey:  "sk-test-123",
		BaseURL: "https://api.anthropic.com",
	})
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// List should now have 2
	providers = mgr.List()
	if len(providers) != 2 {
		t.Fatalf("provider count after add = %d, want 2", len(providers))
	}

	// Get custom provider
	p, err := mgr.Get("custom-anthropic")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if p.APIKey != "sk-test-123" {
		t.Errorf("apiKey = %q, want %q", p.APIKey, "sk-test-123")
	}

	// Update provider
	err = mgr.Update("custom-anthropic", ProviderConfig{
		APIKey: "sk-updated-456",
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	p, err = mgr.Get("custom-anthropic")
	if err != nil {
		t.Fatalf("Get after update error: %v", err)
	}
	if p.APIKey != "sk-updated-456" {
		t.Errorf("apiKey after update = %q", p.APIKey)
	}

	// Remove provider
	err = mgr.Remove("custom-anthropic")
	if err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	providers = mgr.List()
	if len(providers) != 1 {
		t.Errorf("provider count after remove = %d, want 1", len(providers))
	}
}

func TestManagerSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManagerWithDir(tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithDir error: %v", err)
	}

	// Default should be active initially
	active := mgr.GetActive()
	if active.Name != "default" {
		t.Errorf("initial active = %q, want %q", active.Name, "default")
	}

	// Add and switch to custom
	mgr.Add(ProviderConfig{
		Name:   "my-provider",
		APIKey: "sk-test",
	})
	err = mgr.Switch("my-provider")
	if err != nil {
		t.Fatalf("Switch error: %v", err)
	}

	active = mgr.GetActive()
	if active.Name != "my-provider" {
		t.Errorf("active after switch = %q, want %q", active.Name, "my-provider")
	}

	// Switch back to default
	err = mgr.Switch("default")
	if err != nil {
		t.Fatalf("Switch to default error: %v", err)
	}
	active = mgr.GetActive()
	if active.Name != "default" {
		t.Errorf("active after switch back = %q", active.Name)
	}
}

func TestManagerSwitchNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManagerWithDir(tmpDir)

	err := mgr.Switch("no-such-provider")
	if err == nil {
		t.Error("expected error switching to non-existent provider")
	}
}

func TestManagerCannotModifyDefault(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManagerWithDir(tmpDir)

	err := mgr.Add(ProviderConfig{Name: "default", APIKey: "test"})
	if err == nil {
		t.Error("should not allow creating provider named 'default'")
	}

	err = mgr.Update("default", ProviderConfig{APIKey: "test"})
	if err == nil {
		t.Error("should not allow updating 'default' provider")
	}

	err = mgr.Remove("default")
	if err == nil {
		t.Error("should not allow removing 'default' provider")
	}
}

func TestManagerEnvVars(t *testing.T) {
	// Default provider should return nil
	p := ProviderConfig{Name: "default", IsDefault: true}
	envs := ProviderEnvVars(p)
	if envs != nil {
		t.Errorf("default provider envs = %v, want nil", envs)
	}

	// Custom provider should inject env vars
	p = ProviderConfig{
		Name:    "custom",
		APIKey:  "sk-test-123",
		BaseURL: "https://custom.api.com",
	}
	envs = ProviderEnvVars(p)
	if len(envs) != 2 {
		t.Fatalf("custom provider envs count = %d, want 2", len(envs))
	}
	if envs[0] != "ANTHROPIC_API_KEY=sk-test-123" {
		t.Errorf("env[0] = %q", envs[0])
	}
	if envs[1] != "ANTHROPIC_BASE_URL=https://custom.api.com" {
		t.Errorf("env[1] = %q", envs[1])
	}

	// Custom without BaseURL
	p = ProviderConfig{Name: "custom2", APIKey: "sk-test"}
	envs = ProviderEnvVars(p)
	if len(envs) != 1 {
		t.Errorf("custom without base URL envs count = %d, want 1", len(envs))
	}
}

func TestStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, ".claudepilot", "providers.json")

	// Create store and add data
	store, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}

	store.Add(ProviderConfig{Name: "test", APIKey: "sk-persist"})
	store.SetActive("test")

	// Reload store from disk
	store2, err := NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore reload error: %v", err)
	}

	p, ok := store2.Get("test")
	if !ok {
		t.Error("provider should persist across store reload")
	}
	if p.APIKey != "sk-persist" {
		t.Errorf("apiKey after reload = %q", p.APIKey)
	}
	if store2.GetActive() != "test" {
		t.Errorf("active after reload = %q, want %q", store2.GetActive(), "test")
	}
}
