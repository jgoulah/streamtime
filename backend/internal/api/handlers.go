package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jgoulah/streamtime/internal/database"
)

// Handler holds dependencies for API handlers
type Handler struct {
	db *database.DB
}

// NewHandler creates a new API handler
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// healthCheck returns the health status of the API
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// getServices returns all services with their current month statistics
func (h *Handler) getServices(w http.ResponseWriter, r *http.Request) {
	// Get current month range
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	stats, err := h.db.GetServiceStats(startOfMonth, endOfMonth)
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

	// TODO: Implement scraper triggering
	// For now, return a placeholder response
	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "Scraper triggered",
		"service": serviceName,
		"status":  "pending",
	})
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
