package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StoreFile is the on-disk format for providers.json.
type StoreFile struct {
	Version int             `json:"version"`
	Active  string          `json:"active"`
	Providers []ProviderConfig `json:"providers"`
}

// Store persists provider configurations to a JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	data StoreFile
}

// NewStore loads or creates a provider store at the given path.
func NewStore(path string) (*Store, error) {
	s := &Store{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, fmt.Errorf("create store dir: %w", err)
			}
			s.data = StoreFile{
				Version:   1,
				Active:    "default",
				Providers: []ProviderConfig{},
			}
			s.save()
			return s, nil
		}
		return nil, fmt.Errorf("read store file: %w", err)
	}

	if err := json.Unmarshal(data, &s.data); err != nil {
		return nil, fmt.Errorf("parse store file: %w", err)
	}

	// Default active provider
	if s.data.Active == "" {
		s.data.Active = "default"
	}

	return s, nil
}

// List returns all custom providers.
func (s *Store) List() []ProviderConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ProviderConfig, len(s.data.Providers))
	copy(result, s.data.Providers)
	return result
}

// Get returns a specific provider by name.
func (s *Store) Get(name string) (ProviderConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.data.Providers {
		if p.Name == name {
			return p, true
		}
	}
	return ProviderConfig{}, false
}

// Add creates a new provider.
func (s *Store) Add(p ProviderConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	for _, existing := range s.data.Providers {
		if existing.Name == p.Name {
			return fmt.Errorf("provider %q already exists", p.Name)
		}
	}

	s.data.Providers = append(s.data.Providers, p)
	return s.save()
}

// Update modifies an existing provider.
func (s *Store) Update(name string, update ProviderConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.data.Providers {
		if p.Name == name {
			if update.Name != "" && update.Name != name {
				s.data.Providers[i].Name = update.Name
			}
			if update.APIKey != "" {
				s.data.Providers[i].APIKey = update.APIKey
			}
			if update.BaseURL != "" {
				s.data.Providers[i].BaseURL = update.BaseURL
			}
			if update.Model != "" {
				s.data.Providers[i].Model = update.Model
			}
			return s.save()
		}
	}

	return fmt.Errorf("provider %q not found", name)
}

// Remove deletes a provider by name.
func (s *Store) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.data.Providers {
		if p.Name == name {
			s.data.Providers = append(s.data.Providers[:i], s.data.Providers[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("provider %q not found", name)
}

// GetActive returns the active provider name.
func (s *Store) GetActive() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Active
}

// SetActive changes the active provider.
func (s *Store) SetActive(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Active = name
	s.save()
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal store: %w", err)
	}

	// Ensure parent directory exists
	os.MkdirAll(filepath.Dir(s.path), 0o755)

	// Atomic write: write to temp file then rename
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write store: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename store: %w", err)
	}

	return nil
}
