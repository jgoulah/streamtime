package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection with time parsing parameters
	// This allows SQLite to parse datetime strings into time.Time
	connStr := dbPath
	if dbPath == ":memory:" {
		connStr = "file::memory:?cache=shared&_loc=auto"
	} else {
		connStr = dbPath + "?_loc=auto"
	}
	sqlDB, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{sqlDB}

	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// migrate runs database migrations
func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS services (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			color TEXT NOT NULL,
			logo_url TEXT,
			enabled BOOLEAN DEFAULT 1,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS watch_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			watched_at TIMESTAMP NOT NULL,
			episode_info TEXT,
			thumbnail_url TEXT,
			genre TEXT,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (service_id) REFERENCES services(id),
			UNIQUE(service_id, title, watched_at)
		)`,
		`CREATE TABLE IF NOT EXISTS scraper_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id INTEGER NOT NULL,
			ran_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT NOT NULL,
			error_message TEXT,
			items_scraped INTEGER DEFAULT 0,
			FOREIGN KEY (service_id) REFERENCES services(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_history_service_id ON watch_history(service_id)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_history_watched_at ON watch_history(watched_at)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_runs_service_id ON scraper_runs(service_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Seed default services
	if err := db.seedServices(); err != nil {
		return fmt.Errorf("failed to seed services: %w", err)
	}

	return nil
}

// seedServices inserts default streaming services if they don't exist
func (db *DB) seedServices() error {
	services := []struct {
		name    string
		color   string
		logoURL string
	}{
		{"Netflix", "#E50914", "/logos/netflix.svg"},
		{"YouTube TV", "#FF0000", "/logos/youtube-tv.svg"},
		{"Amazon Video", "#00A8E1", "/logos/amazon-video.svg"},
		{"HBO Max", "#7B3FF2", "/logos/hbo-max.svg"},
		{"Apple TV+", "#000000", "/logos/apple-tv.svg"},
		{"Peacock", "#000000", "/logos/peacock.svg"},
	}

	for _, svc := range services {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO services (name, color, logo_url, enabled)
			VALUES (?, ?, ?, 0)
		`, svc.name, svc.color, svc.logoURL)
		if err != nil {
			return err
		}
	}

	return nil
}
