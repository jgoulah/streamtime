package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jgoulah/streamtime/internal/database"
)

// setupTestAPI creates a test database and API handler
func setupTestAPI(t *testing.T) (*Handler, *database.DB) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	handler := NewHandler(db)
	return handler, db
}

func TestHealthCheck(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.healthCheck(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}

	if response["time"] == "" {
		t.Error("Expected time to be set")
	}
}

func TestGetServices(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	// Enable a service and add some watch history
	service, _ := db.GetServiceByName("Netflix")
	db.UpdateServiceEnabled(service.ID, true)

	now := time.Now()
	wh := &database.WatchHistory{
		ServiceID:       service.ID,
		Title:           "Test Movie",
		DurationMinutes: 120,
		WatchedAt:       now,
	}
	db.InsertWatchHistory(wh)

	req, err := http.NewRequest("GET", "/api/services", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.getServices(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var stats []database.ServiceStats
	err = json.NewDecoder(rr.Body).Decode(&stats)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(stats) == 0 {
		t.Error("Expected at least one service stat")
	}

	// Find Netflix in the stats
	var netflixStats *database.ServiceStats
	for i := range stats {
		if stats[i].ServiceName == "Netflix" {
			netflixStats = &stats[i]
			break
		}
	}

	if netflixStats == nil {
		t.Fatal("Netflix not found in service stats")
	}

	if netflixStats.TotalMinutes != 120 {
		t.Errorf("Expected total minutes 120, got %d", netflixStats.TotalMinutes)
	}
}

func TestGetServiceHistory(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	// Setup test data
	service, _ := db.GetServiceByName("Netflix")
	now := time.Now()

	for i := 0; i < 5; i++ {
		wh := &database.WatchHistory{
			ServiceID:       service.ID,
			Title:           "Test Movie",
			DurationMinutes: 60,
			WatchedAt:       now.Add(time.Duration(-i) * time.Hour),
		}
		db.InsertWatchHistory(wh)
	}

	req, err := http.NewRequest("GET", "/api/services/1/history", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Setup mux vars for service ID
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	rr := httptest.NewRecorder()
	handler.getServiceHistory(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that response has expected fields
	if response["history"] == nil {
		t.Error("Expected 'history' field in response")
	}
	if response["daily_stats"] == nil {
		t.Error("Expected 'daily_stats' field in response")
	}
	if response["start_date"] == nil {
		t.Error("Expected 'start_date' field in response")
	}
	if response["end_date"] == nil {
		t.Error("Expected 'end_date' field in response")
	}

	// Verify history is an array
	history, ok := response["history"].([]interface{})
	if !ok {
		t.Fatal("History is not an array")
	}

	if len(history) != 5 {
		t.Errorf("Expected 5 history items, got %d", len(history))
	}
}

func TestGetServiceHistoryWithInvalidID(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	req, err := http.NewRequest("GET", "/api/services/invalid/history", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})

	rr := httptest.NewRecorder()
	handler.getServiceHistory(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, status)
	}
}

func TestGetServiceHistoryWithQueryParams(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	// Setup test data
	service, _ := db.GetServiceByName("Netflix")
	baseDate := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		wh := &database.WatchHistory{
			ServiceID:       service.ID,
			Title:           "Test Movie",
			DurationMinutes: 60,
			WatchedAt:       baseDate.AddDate(0, 0, i),
		}
		db.InsertWatchHistory(wh)
	}

	req, err := http.NewRequest("GET", "/api/services/1/history?start=2025-01-01&end=2025-01-05&limit=3&offset=0", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	rr := httptest.NewRecorder()
	handler.getServiceHistory(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	history, ok := response["history"].([]interface{})
	if !ok {
		t.Fatal("History is not an array")
	}

	// Should be limited to 3 items (from the date range of 5 days, limited to 3)
	if len(history) > 3 {
		t.Errorf("Expected at most 3 history items due to limit, got %d", len(history))
	}
}

func TestTriggerScrape(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	req, err := http.NewRequest("POST", "/api/scrape/netflix", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = mux.SetURLVars(req, map[string]string{"service": "netflix"})

	rr := httptest.NewRecorder()
	handler.triggerScrape(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("Expected status code %d, got %d", http.StatusAccepted, status)
	}

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["service"] != "netflix" {
		t.Errorf("Expected service 'netflix', got '%v'", response["service"])
	}

	if response["status"] != "pending" {
		t.Errorf("Expected status 'pending', got '%v'", response["status"])
	}
}

func TestGetScraperStatus(t *testing.T) {
	handler, db := setupTestAPI(t)
	defer db.Close()

	// Insert a scraper run
	service, _ := db.GetServiceByName("Netflix")
	run := &database.ScraperRun{
		ServiceID:    service.ID,
		RanAt:        time.Now(),
		Status:       "success",
		ItemsScraped: 10,
	}
	db.InsertScraperRun(run)

	req, err := http.NewRequest("GET", "/api/scraper/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.getScraperStatus(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var runs []database.ScraperRun
	err = json.NewDecoder(rr.Body).Decode(&runs)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
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
		t.Error("Scraper run not found in response")
	}
}

func TestParseDateHelper(t *testing.T) {
	defaultDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test valid date
	result := parseDate("2025-06-15", defaultDate)
	if result.Year() != 2025 || result.Month() != 6 || result.Day() != 15 {
		t.Errorf("Expected 2025-06-15, got %v", result)
	}

	// Test empty string (should return default)
	result = parseDate("", defaultDate)
	if !result.Equal(defaultDate) {
		t.Errorf("Expected default date, got %v", result)
	}

	// Test invalid date (should return default)
	result = parseDate("invalid-date", defaultDate)
	if !result.Equal(defaultDate) {
		t.Errorf("Expected default date for invalid input, got %v", result)
	}
}

func TestParseIntParamHelper(t *testing.T) {
	// Test valid int
	result := parseIntParam("42", 10)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test empty string (should return default)
	result = parseIntParam("", 10)
	if result != 10 {
		t.Errorf("Expected default 10, got %d", result)
	}

	// Test invalid int (should return default)
	result = parseIntParam("invalid", 10)
	if result != 10 {
		t.Errorf("Expected default 10 for invalid input, got %d", result)
	}
}
