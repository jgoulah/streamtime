package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

// AmazonScraper implements the Scraper interface for Amazon Prime Video
type AmazonScraper struct {
	config     *config.Config
	db         *database.DB
	serviceKey string
}

// NewAmazonScraper creates a new Amazon scraper
func NewAmazonScraper(cfg *config.Config, db *database.DB) *AmazonScraper {
	return &AmazonScraper{
		config:     cfg,
		db:         db,
		serviceKey: "Amazon Video",
	}
}

// Name returns the service name
func (s *AmazonScraper) Name() string {
	return s.serviceKey
}

// Scrape fetches viewing history from Amazon Prime Video
func (s *AmazonScraper) Scrape(ctx context.Context) ([]database.WatchHistory, error) {
	// Get service config
	serviceCfg, ok := s.config.Services["amazon_video"]
	if !ok || !serviceCfg.Enabled {
		return nil, fmt.Errorf("amazon_video not configured or not enabled")
	}

	// Create chrome context with timeout
	timeout := time.Duration(s.config.Scraper.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Setup chromedp options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.config.Scraper.Headless),
		chromedp.UserAgent(s.config.Scraper.UserAgent),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer chromeCancel()

	// Load authentication cookies
	if err := s.loadCookies(chromeCtx, serviceCfg.Cookies); err != nil {
		return nil, fmt.Errorf("failed to load cookies: %w", err)
	}

	// Navigate to watch history
	if err := s.navigateToWatchHistory(chromeCtx); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Extract viewing history
	items, err := s.extractViewingHistory(chromeCtx)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	log.Printf("Amazon scraper extracted %d items", len(items))
	return items, nil
}

// loadCookies loads authentication cookies into the browser
func (s *AmazonScraper) loadCookies(ctx context.Context, cookies []config.Cookie) error {
	// First navigate to amazon.com to set cookies
	if err := chromedp.Run(ctx, chromedp.Navigate("https://www.amazon.com")); err != nil {
		return fmt.Errorf("failed to navigate to amazon.com: %w", err)
	}

	// Wait a moment for the page to load
	time.Sleep(2 * time.Second)

	// Convert and set cookies
	for _, cookie := range cookies {
		expr := cdp.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour))
		if err := chromedp.Run(ctx,
			network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(".amazon.com").
				WithPath("/").
				WithHTTPOnly(false).
				WithSecure(true).
				WithExpires(&expr),
		); err != nil {
			return fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err)
		}
		log.Printf("Set cookie: %s", cookie.Name)
	}

	return nil
}

// navigateToWatchHistory navigates to the Prime Video watch history page
func (s *AmazonScraper) navigateToWatchHistory(ctx context.Context) error {
	url := "https://www.amazon.com/gp/video/settings/watch-history"

	log.Printf("Navigating to Amazon watch history: %s", url)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	); err != nil {
		return fmt.Errorf("failed to navigate to watch history: %w", err)
	}

	// Wait for content to load (adjust selector after inspection)
	time.Sleep(5 * time.Second)

	log.Println("Waiting for content to load...")
	// TODO: Add proper wait condition based on actual page structure
	// Example: chromedp.WaitVisible(`div[data-testid="watch-history"]`)

	return nil
}

// extractViewingHistory extracts watch history from the current page
func (s *AmazonScraper) extractViewingHistory(ctx context.Context) ([]database.WatchHistory, error) {
	var items []database.WatchHistory
	itemCount := 0

	log.Println("Extracting viewing history from Amazon Prime Video...")

	// Find all date sections (div.RdNoU_.j98KWz)
	var dateSections []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(`div.RdNoU_.j98KWz`, &dateSections, chromedp.ByQueryAll),
	); err != nil {
		return nil, fmt.Errorf("failed to find date sections: %w", err)
	}

	log.Printf("Found %d date sections", len(dateSections))

	for _, dateSection := range dateSections {
		// Extract the date from h3 tag within this section
		var dateText string
		if err := chromedp.Run(ctx,
			chromedp.TextContent(`h3`, &dateText, chromedp.ByQuery, chromedp.FromNode(dateSection)),
		); err != nil {
			log.Printf("Failed to extract date from section: %v", err)
			continue
		}

		watchDate, err := parseAmazonDate(dateText)
		if err != nil {
			log.Printf("Failed to parse date '%s': %v", dateText, err)
			continue
		}

		log.Printf("Processing date section: %s", dateText)

		// Find all show/movie containers within this date section
		var showContainers []*cdp.Node
		if err := chromedp.Run(ctx,
			chromedp.Nodes(`div._6YbHut`, &showContainers, chromedp.ByQueryAll, chromedp.FromNode(dateSection)),
		); err != nil {
			log.Printf("Failed to find show containers for date %s: %v", dateText, err)
			continue
		}

		log.Printf("Found %d shows/movies for date %s", len(showContainers), dateText)

		for _, container := range showContainers {
			// Extract the title
			var title string
			if err := chromedp.Run(ctx,
				chromedp.TextContent(`a._1NNx6V.ZrYV9r`, &title, chromedp.ByQuery, chromedp.FromNode(container)),
			); err != nil {
				log.Printf("Failed to extract title: %v", err)
				continue
			}

			title = strings.TrimSpace(title)
			log.Printf("Processing: %s", title)

			// Check if there are episodes (p.vTfuZU)
			var episodeNodes []*cdp.Node
			if err := chromedp.Run(ctx,
				chromedp.Nodes(`p.vTfuZU`, &episodeNodes, chromedp.ByQueryAll, chromedp.FromNode(container)),
			); err != nil || len(episodeNodes) == 0 {
				// No episodes - this is a movie or single video
				item := database.WatchHistory{
					Title:           title,
					DurationMinutes: 0, // Amazon doesn't show duration in history
					WatchedAt:       watchDate,
					EpisodeInfo:     "",
					Created:         time.Now(),
				}
				items = append(items, item)
				itemCount++
				log.Printf("Added movie/video: %s", title)
			} else {
				// This is a TV show with episodes
				log.Printf("Found %d episodes for show: %s", len(episodeNodes), title)

				for _, episodeNode := range episodeNodes {
					var episodeName string
					if err := chromedp.Run(ctx,
						chromedp.TextContent(`.`, &episodeName, chromedp.ByQuery, chromedp.FromNode(episodeNode)),
					); err != nil {
						log.Printf("Failed to extract episode name: %v", err)
						continue
					}

					episodeName = strings.TrimSpace(episodeName)

					// Create entry with format "Title - Episode Name"
					item := database.WatchHistory{
						Title:           fmt.Sprintf("%s - %s", title, episodeName),
						DurationMinutes: 0, // Amazon doesn't show duration in history
						WatchedAt:       watchDate,
						EpisodeInfo:     episodeName,
						Created:         time.Now(),
					}
					items = append(items, item)
					itemCount++
					log.Printf("Added episode: %s - %s", title, episodeName)

					if s.config.Scraper.TestMode && itemCount >= s.config.Scraper.TestLimit {
						log.Printf("Test mode: stopping at %d items", s.config.Scraper.TestLimit)
						return items, nil
					}
				}
			}

			if s.config.Scraper.TestMode && itemCount >= s.config.Scraper.TestLimit {
				log.Printf("Test mode: stopping at %d items", s.config.Scraper.TestLimit)
				return items, nil
			}
		}
	}

	log.Printf("Amazon scraper extracted %d total items", len(items))
	return items, nil
}

// Helper function to parse duration string (e.g., "1h 30m" -> minutes)
func parseDuration(durationStr string) int {
	// TODO: Implement based on Amazon's duration format
	// This will depend on how Amazon displays video duration
	return 0
}

// parseAmazonDate parses Amazon's date format from watch history
// Handles formats like "October 28, 2024", "Today", "Yesterday"
func parseAmazonDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	now := time.Now()

	// Handle relative dates
	switch strings.ToLower(dateStr) {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local), nil
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local), nil
	}

	// Try common date formats Amazon might use
	formats := []string{
		"January 2, 2006",   // "October 28, 2024"
		"Jan 2, 2006",       // "Oct 28, 2024"
		"1/2/2006",          // "10/28/2024"
		"2006-01-02",        // "2024-10-28"
		"January 2",         // "October 28" (assumes current year)
		"Jan 2",             // "Oct 28" (assumes current year)
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// If format doesn't include year, assume current year
			if !strings.Contains(format, "2006") {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
