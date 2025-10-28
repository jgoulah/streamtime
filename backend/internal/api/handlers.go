package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
	"github.com/jgoulah/streamtime/internal/scraper"
)

// Handler holds dependencies for API handlers
type Handler struct {
	db             *database.DB
	scraperManager *scraper.Manager
	config         *config.Config
}

// NewHandler creates a new API handler
func NewHandler(db *database.DB, scraperMgr *scraper.Manager, cfg *config.Config) *Handler {
	return &Handler{
		db:             db,
		scraperManager: scraperMgr,
		config:         cfg,
	}
}

// healthCheck returns the health status of the API
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// getServices returns all services with their statistics
func (h *Handler) getServices(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for date range
	query := r.URL.Query()

	var startDate, endDate time.Time

	// Check if specific year or month is requested
	yearStr := query.Get("year")
	monthStr := query.Get("month")

	if yearStr != "" {
		// Specific year requested
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 2000 || year > 2100 {
			respondError(w, http.StatusBadRequest, "Invalid year parameter", fmt.Errorf("year must be between 2000 and 2100"))
			return
		}

		if monthStr != "" {
			// Specific month within year
			month, err := strconv.Atoi(monthStr)
			if err != nil || month < 1 || month > 12 {
				respondError(w, http.StatusBadRequest, "Invalid month parameter", fmt.Errorf("month must be between 1 and 12"))
				return
			}
			startDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 1, 0)
		} else {
			// Entire year
			startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			endDate = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
		}
	} else {
		// All-time stats (default)
		startDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Now().AddDate(1, 0, 0) // One year from now
	}

	stats, err := h.db.GetServiceStats(startDate, endDate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch services", err)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// getServiceHistory returns watch history for a specific service
func (h *Handler) getServiceHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID", err)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Date range (default to current month)
	now := time.Now()
	startDate := parseDate(query.Get("start"), time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	endDate := parseDate(query.Get("end"), startDate.AddDate(0, 1, 0))

	// Pagination
	limit := parseIntParam(query.Get("limit"), 100)
	offset := parseIntParam(query.Get("offset"), 0)

	history, err := h.db.GetWatchHistory(serviceID, startDate, endDate, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch history", err)
		return
	}

	// Get daily stats for charting
	dailyStats, err := h.db.GetDailyStats(serviceID, startDate, endDate)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch daily stats", err)
		return
	}

	response := map[string]interface{}{
		"history":     history,
		"daily_stats": dailyStats,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    endDate.Format("2006-01-02"),
	}

	respondJSON(w, http.StatusOK, response)
}

// triggerScrape manually triggers a scraper for a specific service
func (h *Handler) triggerScrape(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["service"]

	// Capitalize service name to match database format (e.g., "netflix" -> "Netflix")
	serviceNameCapitalized := capitalizeServiceName(serviceName)

	// Run scraper in background (with timeout)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		result, err := h.scraperManager.Run(ctx, serviceNameCapitalized)
		if err != nil {
			// Error is already logged in scraper manager
			return
		}

		// Log result
		if result.Success {
			// Successfully scraped
			return
		}
	}()

	// Return immediate response
	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Scraper triggered",
		"service": serviceName,
		"status":  "running",
	})
}

// capitalizeServiceName converts service names to database format
func capitalizeServiceName(name string) string {
	switch name {
	case "netflix":
		return "Netflix"
	case "youtube_tv":
		return "YouTube TV"
	case "amazon_video":
		return "Amazon Video"
	case "hbo_max":
		return "HBO Max"
	case "apple_tv":
		return "Apple TV+"
	case "peacock":
		return "Peacock"
	default:
		return name
	}
}

// getScraperStatus returns the status of recent scraper runs
func (h *Handler) getScraperStatus(w http.ResponseWriter, r *http.Request) {
	runs, err := h.db.GetLatestScraperRuns()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch scraper status", err)
		return
	}

	respondJSON(w, http.StatusOK, runs)
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]string{
		"error":   message,
		"details": err.Error(),
	}
	respondJSON(w, status, response)
}

func parseDate(dateStr string, defaultDate time.Time) time.Time {
	if dateStr == "" {
		return defaultDate
	}
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return defaultDate
	}
	return parsed
}

func parseIntParam(param string, defaultVal int) int {
	if param == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(param)
	if err != nil {
		return defaultVal
	}
	return val
}
