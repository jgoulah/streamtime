package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/jgoulah/streamtime/internal/database"
)

// NetflixImporter handles importing Netflix viewing activity from CSV
type NetflixImporter struct {
	db         *database.DB
	tmdbClient *TMDBClient
	serviceID  int64
}

// NewNetflixImporter creates a new Netflix CSV importer
func NewNetflixImporter(db *database.DB, tmdbAPIKey string) *NetflixImporter {
	return &NetflixImporter{
		db:         db,
		tmdbClient: NewTMDBClient(tmdbAPIKey),
		serviceID:  1, // Netflix service ID (should match database)
	}
}

// CSVRow represents a row in the Netflix CSV
type CSVRow struct {
	Title string
	Date  string
}

// ImportResult contains statistics about the import
type ImportResult struct {
	TotalRows     int
	Imported      int
	Skipped       int
	Errors        int
	ErrorMessages []string
}

// ImportCSV imports Netflix viewing activity from CSV
func (ni *NetflixImporter) ImportCSV(reader io.Reader) (*ImportResult, error) {
	result := &ImportResult{
		ErrorMessages: make([]string, 0),
	}

	csvReader := csv.NewReader(reader)

	// Read header row
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	if len(header) < 2 {
		return nil, fmt.Errorf("invalid CSV format: expected at least 2 columns, got %d", len(header))
	}

	log.Printf("CSV Header: %v", header)

	// Read and process each row
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors++
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("CSV read error: %v", err))
			continue
		}

		result.TotalRows++

		// Parse CSV row
		if len(record) < 2 {
			result.Skipped++
			log.Printf("Skipping row with insufficient columns: %v", record)
			continue
		}

		row := CSVRow{
			Title: strings.TrimSpace(record[0]),
			Date:  strings.TrimSpace(record[1]),
		}

		// Process the row
		if err := ni.processRow(row); err != nil {
			result.Errors++
			result.ErrorMessages = append(result.ErrorMessages,
				fmt.Sprintf("Error processing '%s': %v", row.Title, err))
			log.Printf("Error processing row: %v", err)
		} else {
			result.Imported++
		}
	}

	log.Printf("Import complete: Total=%d, Imported=%d, Skipped=%d, Errors=%d",
		result.TotalRows, result.Imported, result.Skipped, result.Errors)

	return result, nil
}

// processRow processes a single CSV row
func (ni *NetflixImporter) processRow(row CSVRow) error {
	// Parse date
	watchedAt, err := ni.parseDate(row.Date)
	if err != nil {
		return fmt.Errorf("failed to parse date '%s': %w", row.Date, err)
	}

	// Split title and episode info if present
	// Netflix format: "Show Title: Season 1: Episode Name"
	title, episodeInfo := ni.splitTitleAndEpisode(row.Title)

	// Lookup duration from TMDB
	contentInfo, err := ni.tmdbClient.SearchTitle(title)
	if err != nil {
		log.Printf("TMDB lookup failed for '%s': %v, using default duration", title, err)
		// Use default durations if TMDB fails
		duration := ni.estimateDuration(title, episodeInfo)
		contentInfo = &ContentInfo{
			Title:         title,
			DurationMinutes: duration,
			MediaType:     "unknown",
		}
	}

	log.Printf("Title: '%s', Duration: %d min, Type: %s",
		title, contentInfo.DurationMinutes, contentInfo.MediaType)

	// Create watch history entry
	watchHistory := database.WatchHistory{
		ServiceID:       ni.serviceID,
		Title:           title,
		EpisodeInfo:     episodeInfo,
		DurationMinutes: contentInfo.DurationMinutes,
		WatchedAt:       watchedAt,
		ThumbnailURL:    "", // Not available in CSV
		Genre:           "", // Not available in CSV
	}

	// Check if this entry already exists (avoid duplicates)
	exists, err := ni.db.WatchHistoryExists(ni.serviceID, title, episodeInfo, watchedAt)
	if err != nil {
		return fmt.Errorf("failed to check for existing entry: %w", err)
	}

	if exists {
		log.Printf("Entry already exists, skipping: %s at %s", title, watchedAt.Format("2006-01-02"))
		return nil // Not an error, just skip
	}

	// Insert into database
	if err := ni.db.InsertWatchHistory(&watchHistory); err != nil {
		return fmt.Errorf("failed to insert watch history: %w", err)
	}

	return nil
}

// parseDate parses Netflix date format
func (ni *NetflixImporter) parseDate(dateStr string) (time.Time, error) {
	// Netflix typically uses MM/DD/YYYY or M/D/YY format
	layouts := []string{
		"1/2/06",       // M/D/YY
		"01/02/2006",   // MM/DD/YYYY
		"1/2/2006",     // M/D/YYYY
		"2006-01-02",   // YYYY-MM-DD
		"Jan 2, 2006",  // Jan 2, 2006
		"January 2, 2006", // January 2, 2006
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// splitTitleAndEpisode splits title and episode info
func (ni *NetflixImporter) splitTitleAndEpisode(fullTitle string) (title string, episodeInfo string) {
	// Netflix format examples:
	// "The Office (U.S.): Season 3: The Convict"
	// "Breaking Bad: Season 5: Ozymandias"
	// "The Matrix" (no episode info)

	parts := strings.SplitN(fullTitle, ":", 2)
	if len(parts) == 1 {
		// No episode info, it's a movie
		return strings.TrimSpace(parts[0]), ""
	}

	title = strings.TrimSpace(parts[0])
	episodeInfo = strings.TrimSpace(parts[1])
	return title, episodeInfo
}

// estimateDuration provides fallback duration estimates
func (ni *NetflixImporter) estimateDuration(title, episodeInfo string) int {
	// If it has episode info, likely a TV show episode
	if episodeInfo != "" {
		return 40 // Average TV episode
	}
	// Otherwise, assume it's a movie
	return 105 // Average movie length
}
