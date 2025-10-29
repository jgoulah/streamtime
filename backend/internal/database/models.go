package database

import (
	"time"
)

// Service represents a streaming service (Netflix, YouTube TV, etc.)
type Service struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`    // Hex color for UI
	LogoURL  string `json:"logo_url"` // URL or path to logo
	Enabled  bool   `json:"enabled"`
	Created  time.Time `json:"created"`
}

// WatchHistory represents a single viewing session
type WatchHistory struct {
	ID              int64     `json:"id"`
	ServiceID       int64     `json:"service_id"`
	ServiceName     string    `json:"service_name"`
	Title           string    `json:"title"`
	DurationMinutes int       `json:"duration_minutes"`
	WatchedAt       time.Time `json:"watched_at"`
	EpisodeInfo     string    `json:"episode_info"`  // e.g., "S01E05"
	ThumbnailURL    string    `json:"thumbnail_url"`
	Genre           string    `json:"genre"`
	Created         time.Time `json:"created"`
}

// ScraperRun tracks scraper execution history
type ScraperRun struct {
	ID           int64     `json:"id"`
	ServiceID    int64     `json:"service_id"`
	RanAt        time.Time `json:"ran_at"`
	Status       string    `json:"status"` // "success", "failed", "partial"
	ErrorMessage string    `json:"error_message,omitempty"`
	ItemsScraped int       `json:"items_scraped"`
}

// ServiceStats represents aggregated statistics for a service
type ServiceStats struct {
	ServiceID       int64  `json:"service_id"`
	ServiceName     string `json:"service_name"`
	Color           string `json:"color"`
	LogoURL         string `json:"logo_url"`
	TotalMinutes    int    `json:"total_minutes"`
	TotalShows      int    `json:"total_shows"`
	LastWatched     *time.Time `json:"last_watched,omitempty"`
}
