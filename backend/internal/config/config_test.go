package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  path: ./test.db

server:
  port: 9090
  host: 127.0.0.1

services:
  netflix:
    enabled: true
    email: test@example.com
    password: testpass
    use_oauth: false

  youtube_tv:
    enabled: false
    email: test2@example.com
    password: testpass2
    use_oauth: true

scraper:
  schedule: "0 2 * * *"
  headless: true
  timeout: 600
  user_agent: "TestAgent/1.0"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify database config
	if cfg.Database.Path != "./test.db" {
		t.Errorf("Expected database path './test.db', got '%s'", cfg.Database.Path)
	}

	// Verify server config
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", cfg.Server.Host)
	}

	// Verify services
	if len(cfg.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(cfg.Services))
	}

	netflix, ok := cfg.Services["netflix"]
	if !ok {
		t.Fatal("Netflix service not found in config")
	}
	if !netflix.Enabled {
		t.Error("Expected Netflix to be enabled")
	}
	if netflix.Email != "test@example.com" {
		t.Errorf("Expected Netflix email 'test@example.com', got '%s'", netflix.Email)
	}

	youtubeTv, ok := cfg.Services["youtube_tv"]
	if !ok {
		t.Fatal("YouTube TV service not found in config")
	}
	if youtubeTv.Enabled {
		t.Error("Expected YouTube TV to be disabled")
	}
	if !youtubeTv.UseOAuth {
		t.Error("Expected YouTube TV to use OAuth")
	}

	// Verify scraper config
	if cfg.Scraper.Schedule != "0 2 * * *" {
		t.Errorf("Expected schedule '0 2 * * *', got '%s'", cfg.Scraper.Schedule)
	}
	if !cfg.Scraper.Headless {
		t.Error("Expected headless to be true")
	}
	if cfg.Scraper.Timeout != 600 {
		t.Errorf("Expected timeout 600, got %d", cfg.Scraper.Timeout)
	}
	if cfg.Scraper.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected user agent 'TestAgent/1.0', got '%s'", cfg.Scraper.UserAgent)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
services:
  netflix:
    enabled: true
    email: test@example.com
    password: testpass
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults are set
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Database.Path != "./data/streamtime.db" {
		t.Errorf("Expected default database path './data/streamtime.db', got '%s'", cfg.Database.Path)
	}
	if cfg.Scraper.Schedule != "0 3 * * *" {
		t.Errorf("Expected default schedule '0 3 * * *', got '%s'", cfg.Scraper.Schedule)
	}
	if cfg.Scraper.Timeout != 300 {
		t.Errorf("Expected default timeout 300, got %d", cfg.Scraper.Timeout)
	}
	if cfg.Scraper.UserAgent == "" {
		t.Error("Expected default user agent to be set")
	}
}

func TestLoadInvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent config file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
this is not: valid: yaml:
  - with random
    indentation
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestGetEnabledServices(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"netflix": {
				Enabled: true,
			},
			"youtube_tv": {
				Enabled: false,
			},
			"amazon_video": {
				Enabled: true,
			},
		},
	}

	enabled := cfg.GetEnabledServices()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled services, got %d", len(enabled))
	}

	// Check that both enabled services are in the list
	enabledMap := make(map[string]bool)
	for _, name := range enabled {
		enabledMap[name] = true
	}

	if !enabledMap["netflix"] {
		t.Error("Expected netflix to be in enabled services")
	}
	if !enabledMap["amazon_video"] {
		t.Error("Expected amazon_video to be in enabled services")
	}
	if enabledMap["youtube_tv"] {
		t.Error("Did not expect youtube_tv to be in enabled services")
	}
}
