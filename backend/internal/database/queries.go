package database

import (
	"database/sql"
	"time"
)

// GetAllServices returns all services
func (db *DB) GetAllServices() ([]Service, error) {
	rows, err := db.Query(`
		SELECT id, name, color, logo_url, enabled, created
		FROM services
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var svc Service
		err := rows.Scan(&svc.ID, &svc.Name, &svc.Color, &svc.LogoURL, &svc.Enabled, &svc.Created)
		if err != nil {
			return nil, err
		}
		services = append(services, svc)
	}

	return services, rows.Err()
}

// GetServiceByID returns a service by ID
func (db *DB) GetServiceByID(id int64) (*Service, error) {
	var svc Service
	err := db.QueryRow(`
		SELECT id, name, color, logo_url, enabled, created
		FROM services
		WHERE id = ?
	`, id).Scan(&svc.ID, &svc.Name, &svc.Color, &svc.LogoURL, &svc.Enabled, &svc.Created)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

// GetServiceByName returns a service by name
func (db *DB) GetServiceByName(name string) (*Service, error) {
	var svc Service
	err := db.QueryRow(`
		SELECT id, name, color, logo_url, enabled, created
		FROM services
		WHERE name = ?
	`, name).Scan(&svc.ID, &svc.Name, &svc.Color, &svc.LogoURL, &svc.Enabled, &svc.Created)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

// GetServiceStats returns aggregated statistics for all services for a given time period
func (db *DB) GetServiceStats(startDate, endDate time.Time) ([]ServiceStats, error) {
	rows, err := db.Query(`
		SELECT
			s.id,
			s.name,
			s.color,
			s.logo_url,
			COALESCE(SUM(wh.duration_minutes), 0) as total_minutes,
			COUNT(wh.id) as total_shows,
			DATETIME(MAX(wh.watched_at)) as last_watched
		FROM services s
		LEFT JOIN watch_history wh ON s.id = wh.service_id
			AND wh.watched_at >= ?
			AND wh.watched_at < ?
		WHERE s.enabled = 1
		GROUP BY s.id, s.name, s.color, s.logo_url
		ORDER BY total_minutes DESC
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ServiceStats
	for rows.Next() {
		var stat ServiceStats
		var lastWatchedStr sql.NullString
		err := rows.Scan(
			&stat.ServiceID,
			&stat.ServiceName,
			&stat.Color,
			&stat.LogoURL,
			&stat.TotalMinutes,
			&stat.TotalShows,
			&lastWatchedStr,
		)
		if err != nil {
			return nil, err
		}
		if lastWatchedStr.Valid && lastWatchedStr.String != "" {
			// Parse the datetime string from SQLite
			// SQLite DATETIME() format: "YYYY-MM-DD HH:MM:SS"
			t, err := time.Parse("2006-01-02 15:04:05", lastWatchedStr.String)
			if err == nil {
				stat.LastWatched = &t
			}
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// GetWatchHistory returns watch history for a service within a date range
func (db *DB) GetWatchHistory(serviceID int64, startDate, endDate time.Time, limit, offset int) ([]WatchHistory, error) {
	rows, err := db.Query(`
		SELECT id, service_id, title, duration_minutes, watched_at,
		       episode_info, thumbnail_url, genre, created
		FROM watch_history
		WHERE service_id = ?
		  AND watched_at >= ?
		  AND watched_at < ?
		ORDER BY watched_at DESC
		LIMIT ? OFFSET ?
	`, serviceID, startDate, endDate, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []WatchHistory
	for rows.Next() {
		var wh WatchHistory
		err := rows.Scan(
			&wh.ID, &wh.ServiceID, &wh.Title, &wh.DurationMinutes,
			&wh.WatchedAt, &wh.EpisodeInfo, &wh.ThumbnailURL,
			&wh.Genre, &wh.Created,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, wh)
	}

	return history, rows.Err()
}

// InsertWatchHistory inserts or updates a watch history entry
func (db *DB) InsertWatchHistory(wh *WatchHistory) error {
	result, err := db.Exec(`
		INSERT INTO watch_history
		(service_id, title, duration_minutes, watched_at, episode_info, thumbnail_url, genre)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(service_id, title, watched_at) DO UPDATE SET
			duration_minutes = excluded.duration_minutes,
			episode_info = excluded.episode_info,
			thumbnail_url = excluded.thumbnail_url,
			genre = excluded.genre
	`, wh.ServiceID, wh.Title, wh.DurationMinutes, wh.WatchedAt,
		wh.EpisodeInfo, wh.ThumbnailURL, wh.Genre)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		wh.ID = id
	}

	return nil
}

// InsertScraperRun records a scraper execution
func (db *DB) InsertScraperRun(run *ScraperRun) error {
	result, err := db.Exec(`
		INSERT INTO scraper_runs (service_id, ran_at, status, error_message, items_scraped)
		VALUES (?, ?, ?, ?, ?)
	`, run.ServiceID, run.RanAt, run.Status, run.ErrorMessage, run.ItemsScraped)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		run.ID = id
	}

	return nil
}

// GetLatestScraperRuns returns the most recent scraper run for each service
func (db *DB) GetLatestScraperRuns() ([]ScraperRun, error) {
	rows, err := db.Query(`
		SELECT sr.id, sr.service_id, sr.ran_at, sr.status, sr.error_message, sr.items_scraped
		FROM scraper_runs sr
		INNER JOIN (
			SELECT service_id, MAX(ran_at) as max_ran_at
			FROM scraper_runs
			GROUP BY service_id
		) latest ON sr.service_id = latest.service_id AND sr.ran_at = latest.max_ran_at
		ORDER BY sr.ran_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []ScraperRun
	for rows.Next() {
		var run ScraperRun
		err := rows.Scan(
			&run.ID, &run.ServiceID, &run.RanAt,
			&run.Status, &run.ErrorMessage, &run.ItemsScraped,
		)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetDailyStats returns daily aggregated watch time for a service
func (db *DB) GetDailyStats(serviceID int64, startDate, endDate time.Time) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT DATE(watched_at) as day, SUM(duration_minutes) as total_minutes
		FROM watch_history
		WHERE service_id = ?
		  AND watched_at >= ?
		  AND watched_at < ?
		GROUP BY DATE(watched_at)
		ORDER BY day
	`, serviceID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var day string
		var totalMinutes int
		if err := rows.Scan(&day, &totalMinutes); err != nil {
			return nil, err
		}
		stats[day] = totalMinutes
	}

	return stats, rows.Err()
}

// UpdateServiceEnabled updates the enabled status of a service
func (db *DB) UpdateServiceEnabled(serviceID int64, enabled bool) error {
	_, err := db.Exec(`
		UPDATE services SET enabled = ? WHERE id = ?
	`, enabled, serviceID)
	return err
}
