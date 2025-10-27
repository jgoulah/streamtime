package database

import (
	"testing"
	"time"
)

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) *DB {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Verify database is accessible
	err := db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestMigrateCreatesTablesAndSeeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check that services table exists and has seed data
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM services").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query services: %v", err)
	}

	if count != 6 {
		t.Errorf("Expected 6 seeded services, got %d", count)
	}

	// Verify watch_history table exists
	_, err = db.Exec("SELECT * FROM watch_history LIMIT 1")
	if err != nil {
		t.Errorf("watch_history table doesn't exist: %v", err)
	}

	// Verify scraper_runs table exists
	_, err = db.Exec("SELECT * FROM scraper_runs LIMIT 1")
	if err != nil {
		t.Errorf("scraper_runs table doesn't exist: %v", err)
	}
}

func TestGetAllServices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	services, err := db.GetAllServices()
	if err != nil {
		t.Fatalf("Failed to get all services: %v", err)
	}

	if len(services) != 6 {
		t.Errorf("Expected 6 services, got %d", len(services))
	}

	// Verify first service has expected fields
	if services[0].Name == "" {
		t.Error("Service name is empty")
	}
	if services[0].Color == "" {
		t.Error("Service color is empty")
	}
}

func TestGetServiceByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Get a valid service first
	services, _ := db.GetAllServices()
	if len(services) == 0 {
		t.Fatal("No services available for testing")
	}

	service, err := db.GetServiceByID(services[0].ID)
	if err != nil {
		t.Fatalf("Failed to get service by ID: %v", err)
	}

	if service == nil {
		t.Fatal("Expected service, got nil")
	}

	if service.ID != services[0].ID {
		t.Errorf("Expected ID %d, got %d", services[0].ID, service.ID)
	}

	// Test with invalid ID
	service, err = db.GetServiceByID(99999)
	if err != nil {
		t.Fatalf("Expected no error for non-existent service, got: %v", err)
	}
	if service != nil {
		t.Error("Expected nil for non-existent service")
	}
}

func TestGetServiceByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, err := db.GetServiceByName("Netflix")
	if err != nil {
		t.Fatalf("Failed to get service by name: %v", err)
	}

	if service == nil {
		t.Fatal("Expected Netflix service, got nil")
	}

	if service.Name != "Netflix" {
		t.Errorf("Expected name 'Netflix', got '%s'", service.Name)
	}

	// Test with invalid name
	service, err = db.GetServiceByName("NonExistentService")
	if err != nil {
		t.Fatalf("Expected no error for non-existent service, got: %v", err)
	}
	if service != nil {
		t.Error("Expected nil for non-existent service")
	}
}

func TestInsertAndGetWatchHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Get a service to use
	service, _ := db.GetServiceByName("Netflix")
	if service == nil {
		t.Fatal("Netflix service not found")
	}

	// Enable the service first
	err := db.UpdateServiceEnabled(service.ID, true)
	if err != nil {
		t.Fatalf("Failed to enable service: %v", err)
	}

	now := time.Now()
	wh := &WatchHistory{
		ServiceID:       service.ID,
		Title:           "Test Movie",
		DurationMinutes: 120,
		WatchedAt:       now,
		EpisodeInfo:     "N/A",
		ThumbnailURL:    "http://example.com/thumb.jpg",
		Genre:           "Action",
	}

	err = db.InsertWatchHistory(wh)
	if err != nil {
		t.Fatalf("Failed to insert watch history: %v", err)
	}

	if wh.ID == 0 {
		t.Error("Expected ID to be set after insert")
	}

	// Retrieve the watch history
	history, err := db.GetWatchHistory(service.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10, 0)
	if err != nil {
		t.Fatalf("Failed to get watch history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 watch history entry, got %d", len(history))
	}

	if history[0].Title != "Test Movie" {
		t.Errorf("Expected title 'Test Movie', got '%s'", history[0].Title)
	}
}

func TestInsertWatchHistoryDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, _ := db.GetServiceByName("Netflix")
	now := time.Now()

	wh := &WatchHistory{
		ServiceID:       service.ID,
		Title:           "Test Movie",
		DurationMinutes: 120,
		WatchedAt:       now,
	}

	// Insert first time
	err := db.InsertWatchHistory(wh)
	if err != nil {
		t.Fatalf("Failed to insert watch history: %v", err)
	}

	// Insert duplicate with updated duration
	wh2 := &WatchHistory{
		ServiceID:       service.ID,
		Title:           "Test Movie",
		DurationMinutes: 150,
		WatchedAt:       now,
	}

	err = db.InsertWatchHistory(wh2)
	if err != nil {
		t.Fatalf("Failed to insert duplicate watch history: %v", err)
	}

	// Retrieve and verify it was updated
	history, err := db.GetWatchHistory(service.ID, now.Add(-1*time.Hour), now.Add(1*time.Hour), 10, 0)
	if err != nil {
		t.Fatalf("Failed to get watch history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 watch history entry (not 2), got %d", len(history))
	}

	if history[0].DurationMinutes != 150 {
		t.Errorf("Expected duration 150 (updated), got %d", history[0].DurationMinutes)
	}
}

func TestGetServiceStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Get Netflix service
	service, _ := db.GetServiceByName("Netflix")
	err := db.UpdateServiceEnabled(service.ID, true)
	if err != nil {
		t.Fatalf("Failed to enable service: %v", err)
	}

	// Insert some watch history
	now := time.Now()
	for i := 0; i < 3; i++ {
		wh := &WatchHistory{
			ServiceID:       service.ID,
			Title:           "Test Movie",
			DurationMinutes: 60,
			WatchedAt:       now.Add(time.Duration(-i) * time.Hour),
		}
		db.InsertWatchHistory(wh)
	}

	// Get stats for current month
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	stats, err := db.GetServiceStats(startOfMonth, endOfMonth)
	if err != nil {
		t.Fatalf("Failed to get service stats: %v", err)
	}

	// Should have stats for all enabled services (even if 0)
	if len(stats) == 0 {
		t.Error("Expected at least one service stat")
	}

	// Find Netflix in stats
	var netflixStats *ServiceStats
	for i := range stats {
		if stats[i].ServiceName == "Netflix" {
			netflixStats = &stats[i]
			break
		}
	}

	if netflixStats == nil {
		t.Fatal("Netflix stats not found")
	}

	if netflixStats.TotalMinutes != 180 {
		t.Errorf("Expected total minutes 180, got %d", netflixStats.TotalMinutes)
	}

	if netflixStats.TotalShows != 3 {
		t.Errorf("Expected total shows 3, got %d", netflixStats.TotalShows)
	}
}

func TestGetDailyStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, _ := db.GetServiceByName("Netflix")

	// Insert watch history for different days
	baseDate := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		wh := &WatchHistory{
			ServiceID:       service.ID,
			Title:           "Test Movie",
			DurationMinutes: 60,
			WatchedAt:       baseDate.AddDate(0, 0, i),
		}
		db.InsertWatchHistory(wh)
	}

	// Get daily stats
	stats, err := db.GetDailyStats(service.ID, baseDate, baseDate.AddDate(0, 0, 5))
	if err != nil {
		t.Fatalf("Failed to get daily stats: %v", err)
	}

	if len(stats) != 3 {
		t.Errorf("Expected 3 days of stats, got %d", len(stats))
	}

	// Verify each day has 60 minutes
	for day, minutes := range stats {
		if minutes != 60 {
			t.Errorf("Expected 60 minutes for %s, got %d", day, minutes)
		}
	}
}

func TestInsertAndGetScraperRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, _ := db.GetServiceByName("Netflix")

	run := &ScraperRun{
		ServiceID:    service.ID,
		RanAt:        time.Now(),
		Status:       "success",
		ErrorMessage: "",
		ItemsScraped: 10,
	}

	err := db.InsertScraperRun(run)
	if err != nil {
		t.Fatalf("Failed to insert scraper run: %v", err)
	}

	if run.ID == 0 {
		t.Error("Expected ID to be set after insert")
	}

	// Get latest scraper runs
	runs, err := db.GetLatestScraperRuns()
	if err != nil {
		t.Fatalf("Failed to get latest scraper runs: %v", err)
	}

	if len(runs) == 0 {
		t.Error("Expected at least one scraper run")
	}

	// Find our run
	var found bool
	for _, r := range runs {
		if r.ServiceID == service.ID {
			found = true
			if r.Status != "success" {
				t.Errorf("Expected status 'success', got '%s'", r.Status)
			}
			if r.ItemsScraped != 10 {
				t.Errorf("Expected 10 items scraped, got %d", r.ItemsScraped)
			}
			break
		}
	}

	if !found {
		t.Error("Scraper run not found in latest runs")
	}
}

func TestUpdateServiceEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service, _ := db.GetServiceByName("Netflix")

	// Initially disabled (from seed)
	if service.Enabled {
		t.Error("Expected service to be disabled initially")
	}

	// Enable it
	err := db.UpdateServiceEnabled(service.ID, true)
	if err != nil {
		t.Fatalf("Failed to enable service: %v", err)
	}

	// Verify it's enabled
	service, _ = db.GetServiceByID(service.ID)
	if !service.Enabled {
		t.Error("Expected service to be enabled")
	}

	// Disable it
	err = db.UpdateServiceEnabled(service.ID, false)
	if err != nil {
		t.Fatalf("Failed to disable service: %v", err)
	}

	// Verify it's disabled
	service, _ = db.GetServiceByID(service.ID)
	if service.Enabled {
		t.Error("Expected service to be disabled")
	}
}
