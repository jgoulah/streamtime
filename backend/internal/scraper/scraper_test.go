package scraper

import (
	"context"
	"testing"
	"time"

	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

// MockScraper implements the Scraper interface for testing
type MockScraper struct {
	name      string
	items     []database.WatchHistory
	shouldErr bool
}

func (m *MockScraper) Name() string {
	return m.name
}

func (m *MockScraper) Scrape(ctx context.Context) ([]database.WatchHistory, error) {
	if m.shouldErr {
		return nil, ErrNoDataFound
	}
	return m.items, nil
}

func setupTestManager(t *testing.T) (*Manager, *database.DB) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			"netflix": {
				Enabled: true,
			},
		},
	}

	manager := NewManager(db, cfg)
	return manager, db
}

func TestNewManager(t *testing.T) {
	db, _ := database.New(":memory:")
	defer db.Close()

	cfg := &config.Config{}
	manager := NewManager(db, cfg)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.db != db {
		t.Error("Database not set correctly")
	}

	if manager.config != cfg {
		t.Error("Config not set correctly")
	}

	if len(manager.scrapers) != 0 {
		t.Error("Scrapers map should be empty initially")
	}
}

func TestRegisterScraper(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	mockScraper := &MockScraper{
		name: "test_service",
	}

	manager.Register(mockScraper)

	if len(manager.scrapers) != 1 {
		t.Errorf("Expected 1 scraper, got %d", len(manager.scrapers))
	}

	scraper, ok := manager.GetScraper("test_service")
	if !ok {
		t.Error("Scraper not found after registration")
	}

	if scraper.Name() != "test_service" {
		t.Errorf("Expected scraper name 'test_service', got '%s'", scraper.Name())
	}
}

func TestGetScraper(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	mockScraper := &MockScraper{name: "test_service"}
	manager.Register(mockScraper)

	// Test existing scraper
	scraper, ok := manager.GetScraper("test_service")
	if !ok {
		t.Error("Should find registered scraper")
	}
	if scraper.Name() != "test_service" {
		t.Error("Wrong scraper returned")
	}

	// Test non-existent scraper
	_, ok = manager.GetScraper("nonexistent")
	if ok {
		t.Error("Should not find non-existent scraper")
	}
}

func TestRunSuccess(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Enable Netflix service
	service, _ := db.GetServiceByName("Netflix")
	db.UpdateServiceEnabled(service.ID, true)

	// Create mock scraper with test data
	now := time.Now()
	mockItems := []database.WatchHistory{
		{
			Title:           "Test Movie 1",
			DurationMinutes: 120,
			WatchedAt:       now,
		},
		{
			Title:           "Test Movie 2",
			DurationMinutes: 90,
			WatchedAt:       now.Add(-24 * time.Hour),
		},
	}

	mockScraper := &MockScraper{
		name:  "Netflix",
		items: mockItems,
	}

	manager.Register(mockScraper)

	// Run the scraper
	ctx := context.Background()
	result, err := manager.Run(ctx, "Netflix")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	if result.ItemsScraped != 2 {
		t.Errorf("Expected 2 items scraped, got %d", result.ItemsScraped)
	}

	if result.ServiceName != "Netflix" {
		t.Errorf("Expected service name 'Netflix', got '%s'", result.ServiceName)
	}

	// Verify items were stored in database
	history, err := db.GetWatchHistory(service.ID, now.Add(-48*time.Hour), now.Add(1*time.Hour), 10, 0)
	if err != nil {
		t.Fatalf("Failed to get watch history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 items in database, got %d", len(history))
	}
}

func TestRunFailure(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Enable Netflix service
	service, _ := db.GetServiceByName("Netflix")
	db.UpdateServiceEnabled(service.ID, true)

	// Create mock scraper that fails
	mockScraper := &MockScraper{
		name:      "Netflix",
		shouldErr: true,
	}

	manager.Register(mockScraper)

	// Run the scraper
	ctx := context.Background()
	result, err := manager.Run(ctx, "Netflix")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result.Success {
		t.Error("Expected unsuccessful result")
	}

	if result.ItemsScraped != 0 {
		t.Errorf("Expected 0 items scraped, got %d", result.ItemsScraped)
	}

	// Verify failed run was recorded
	runs, err := db.GetLatestScraperRuns()
	if err != nil {
		t.Fatalf("Failed to get scraper runs: %v", err)
	}

	if len(runs) == 0 {
		t.Error("Expected at least one scraper run recorded")
	}

	var foundRun bool
	for _, run := range runs {
		if run.ServiceID == service.ID && run.Status == "failed" {
			foundRun = true
			break
		}
	}

	if !foundRun {
		t.Error("Failed scraper run not found in database")
	}
}

func TestRunNonExistentScraper(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	ctx := context.Background()
	_, err := manager.Run(ctx, "nonexistent")

	if err != ErrScraperNotFound {
		t.Errorf("Expected ErrScraperNotFound, got: %v", err)
	}
}

func TestRunAll(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Enable Netflix
	service, _ := db.GetServiceByName("Netflix")
	db.UpdateServiceEnabled(service.ID, true)

	// Register multiple scrapers
	mockScraper1 := &MockScraper{
		name: "Netflix",
		items: []database.WatchHistory{
			{Title: "Movie 1", DurationMinutes: 120, WatchedAt: time.Now()},
		},
	}

	mockScraper2 := &MockScraper{
		name:      "test_service_2",
		shouldErr: true, // This one will fail
	}

	manager.Register(mockScraper1)
	manager.Register(mockScraper2)

	// Run all scrapers
	ctx := context.Background()
	results, err := manager.RunAll(ctx)

	if err != nil {
		t.Errorf("RunAll should not return error even if individual scrapers fail: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that at least one succeeded
	var hasSuccess bool
	var hasFailure bool
	for _, result := range results {
		if result.Success {
			hasSuccess = true
		} else {
			hasFailure = true
		}
	}

	if !hasSuccess {
		t.Error("Expected at least one successful result")
	}

	if !hasFailure {
		t.Error("Expected at least one failed result")
	}
}

func TestResultTiming(t *testing.T) {
	manager, db := setupTestManager(t)
	defer db.Close()

	// Enable Netflix
	service, _ := db.GetServiceByName("Netflix")
	db.UpdateServiceEnabled(service.ID, true)

	mockScraper := &MockScraper{
		name: "Netflix",
		items: []database.WatchHistory{
			{Title: "Test", DurationMinutes: 60, WatchedAt: time.Now()},
		},
	}

	manager.Register(mockScraper)

	ctx := context.Background()
	result, err := manager.Run(ctx, "Netflix")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify timing is set
	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}
