package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ProviderConfig defines an AI provider configuration.
type ProviderConfig struct {
	Name     string `json:"name"`           // Unique name (e.g., "default", "custom-openai")
	APIKey   string `json:"apiKey"`         // API key (only for custom providers)
	BaseURL  string `json:"baseUrl"`        // Optional custom base URL
	Model    string `json:"model,omitempty"` // Optional default model override
	IsDefault bool  `json:"isDefault"`      // True for the built-in default provider
}

// Manager handles provider CRUD operations.
type Manager struct {
	mu      sync.RWMutex
	homeDir string
	store   *Store
	active  string // name of the active provider
}

// NewManager creates a new provider manager.
func NewManager() (*Manager, error) {
	home, _ := os.UserHomeDir()
	return NewManagerWithDir(home)
}

// NewManagerWithDir creates a Manager with a custom home directory.
func NewManagerWithDir(homeDir string) (*Manager, error) {
	m := &Manager{
		homeDir: homeDir,
		active:  "default",
	}

	store, err := NewStore(filepath.Join(homeDir, ".claudepilot", "providers.json"))
	if err != nil {
		return nil, fmt.Errorf("init provider store: %w", err)
	}
	m.store = store

	// Load saved active provider
	if saved := store.GetActive(); saved != "" {
		m.active = saved
	}

	return m, nil
}

// List returns all providers (built-in default + custom).
func (m *Manager) List() []ProviderConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []ProviderConfig{
		{
			Name:      "default",
			IsDefault: true,
		},
	}

	custom := m.store.List()
	result = append(result, custom...)
	return result
}

// Get returns a specific provider by name.
func (m *Manager) Get(name string) (ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "default" {
		return ProviderConfig{
			Name:      "default",
			IsDefault: true,
		}, nil
	}

	p, ok := m.store.Get(name)
	if !ok {
		return ProviderConfig{}, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

// Add creates a new custom provider.
func (m *Manager) Add(p ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p.Name == "default" {
		return fmt.Errorf("cannot create provider named \"default\"")
	}
	if p.APIKey == "" {
		return fmt.Errorf("API key is required for custom providers")
	}

	return m.store.Add(p)
}

// Update modifies an existing custom provider.
func (m *Manager) Update(name string, update ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot modify the default provider")
	}

	return m.store.Update(name, update)
}

// Remove deletes a custom provider.
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot remove the default provider")
	}

	if m.active == name {
		m.active = "default"
		m.store.SetActive("default")
	}

	return m.store.Remove(name)
}

// Switch changes the active provider.
func (m *Manager) Switch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "default" {
		m.active = "default"
		m.store.SetActive("default")
		return nil
	}

	if _, ok := m.store.Get(name); !ok {
		return fmt.Errorf("provider %q not found", name)
	}

	m.active = name
	m.store.SetActive(name)
	return nil
}

// GetActive returns the active provider config.
func (m *Manager) GetActive() ProviderConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.active == "default" {
		return ProviderConfig{
			Name:      "default",
			IsDefault: true,
		}
	}

	p, ok := m.store.Get(m.active)
	if !ok {
		// Fallback to default if active provider was removed
		m.active = "default"
		return ProviderConfig{
			Name:      "default",
			IsDefault: true,
		}
	}
	return p
}

// EnvVars returns the environment variables to inject for the active provider.
// Default provider returns nil (uses existing Claude Code config).
func (m *Manager) EnvVars() []string {
	p := m.GetActive()
	return ProviderEnvVars(p)
}

// ProviderEnvVars returns env vars for a given provider config.
// Default provider returns nil (zero-config, uses existing env).
func ProviderEnvVars(p ProviderConfig) []string {
	if p.IsDefault || p.APIKey == "" {
		return nil
	}

	var envs []string
	envs = append(envs, "ANTHROPIC_API_KEY="+p.APIKey)
	if p.BaseURL != "" {
		envs = append(envs, "ANTHROPIC_BASE_URL="+p.BaseURL)
	}
	return envs
}
