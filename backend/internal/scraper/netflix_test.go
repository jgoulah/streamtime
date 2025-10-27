package scraper

import (
	"testing"
	"time"

	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

func TestNetflixScraperName(t *testing.T) {
	cfg := &config.Config{}
	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	if scraper.Name() != "Netflix" {
		t.Errorf("Expected name 'Netflix', got '%s'", scraper.Name())
	}
}

func TestParseDate(t *testing.T) {
	cfg := &config.Config{}
	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"1/15/25", "2025-01-15", false},
		{"01/15/2025", "2025-01-15", false},
		{"1/5/2025", "2025-01-05", false},
		{"2025-01-15", "2025-01-15", false},
		{"Jan 15, 2025", "2025-01-15", false},
		{"January 15, 2025", "2025-01-15", false},
		{"invalid date", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := scraper.parseDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			resultStr := result.Format("2006-01-02")
			if resultStr != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, resultStr)
			}
		})
	}
}

func TestEstimateDuration(t *testing.T) {
	cfg := &config.Config{}
	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	tests := []struct {
		title       string
		episodeInfo string
		expected    int
	}{
		{"Stranger Things", "S01E01", 40},
		{"The Crown", "S02E05", 40},
		{"Some Movie", "", 105},
		{"Another Film", "", 105},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := scraper.estimateDuration(tt.title, tt.episodeInfo)
			if result != tt.expected {
				t.Errorf("Expected duration %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseEpisodeInfo(t *testing.T) {
	tests := []struct {
		input          string
		expectedSeason int
		expectedEp     int
		wantErr        bool
	}{
		{"S01E05", 1, 5, false},
		{"S1E5", 1, 5, false},
		{"S10E25", 10, 25, false},
		{"Season 1: Episode 5", 1, 5, false},
		{"Season 10: Episode 25", 10, 25, false},
		{"invalid", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			season, episode, err := parseEpisodeInfo(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if season != tt.expectedSeason {
				t.Errorf("Expected season %d, got %d", tt.expectedSeason, season)
			}

			if episode != tt.expectedEp {
				t.Errorf("Expected episode %d, got %d", tt.expectedEp, episode)
			}
		})
	}
}

func TestNewNetflixScraper(t *testing.T) {
	cfg := &config.Config{
		Scraper: config.ScraperConfig{
			Timeout:   300,
			Headless:  true,
			UserAgent: "TestAgent/1.0",
		},
		Services: map[string]config.ServiceConfig{
			"netflix": {
				Enabled:  true,
				Email:    "test@example.com",
				Password: "testpass",
			},
		},
	}

	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	if scraper == nil {
		t.Fatal("Expected scraper to be created")
	}

	if scraper.config != cfg {
		t.Error("Config not set correctly")
	}

	if scraper.serviceKey != "Netflix" {
		t.Errorf("Expected service key 'Netflix', got '%s'", scraper.serviceKey)
	}
}

// TestScraperDisabled tests that scraper fails gracefully when disabled
func TestScraperDisabled(t *testing.T) {
	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			"netflix": {
				Enabled: false,
			},
		},
	}

	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	// Note: We can't actually test Scrape() without a real browser context
	// This just verifies the scraper is created properly
	if scraper.Name() != "Netflix" {
		t.Error("Scraper should still be created even if disabled")
	}
}

func TestDateParsing(t *testing.T) {
	cfg := &config.Config{}
	db, _ := database.New(":memory:")
	defer db.Close()
	scraper := NewNetflixScraper(cfg, db)

	// Test that we handle current date
	now := time.Now()
	dateStr := now.Format("1/2/06")

	parsed, err := scraper.parseDate(dateStr)
	if err != nil {
		t.Fatalf("Failed to parse current date: %v", err)
	}

	if parsed.Year() != now.Year() || parsed.Month() != now.Month() || parsed.Day() != now.Day() {
		t.Errorf("Parsed date doesn't match expected date")
	}
}
