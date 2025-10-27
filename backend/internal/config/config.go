package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Database DatabaseConfig         `yaml:"database"`
	Server   ServerConfig           `yaml:"server"`
	Services map[string]ServiceConfig `yaml:"services"`
	Scraper  ScraperConfig          `yaml:"scraper"`
	TMDB     TMDBConfig             `yaml:"tmdb"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// Cookie represents a browser cookie
type Cookie struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// ServiceConfig holds configuration for a streaming service
type ServiceConfig struct {
	Enabled bool     `yaml:"enabled"`
	Cookies []Cookie `yaml:"cookies"`
	Email   string   `yaml:"email"` // For non-Netflix services
	Password string  `yaml:"password"` // For non-Netflix services
	UseOAuth bool    `yaml:"use_oauth"` // For non-Netflix services
}

// ScraperConfig holds scraper configuration
type ScraperConfig struct {
	Schedule  string `yaml:"schedule"`   // Cron format
	Headless  bool   `yaml:"headless"`
	Timeout   int    `yaml:"timeout"`    // seconds
	UserAgent string `yaml:"user_agent"`
}

// TMDBConfig holds The Movie Database API configuration
type TMDBConfig struct {
	APIKey string `yaml:"api_key"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/streamtime.db"
	}
	if cfg.Scraper.Schedule == "" {
		cfg.Scraper.Schedule = "0 3 * * *" // 3 AM daily
	}
	if cfg.Scraper.Timeout == 0 {
		cfg.Scraper.Timeout = 300 // 5 minutes
	}
	if cfg.Scraper.UserAgent == "" {
		cfg.Scraper.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}

	return &cfg, nil
}

// GetEnabledServices returns a list of enabled service names
func (c *Config) GetEnabledServices() []string {
	var enabled []string
	for name, svc := range c.Services {
		if svc.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}
