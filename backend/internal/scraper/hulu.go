package scraper

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

// HuluScraper implements the Scraper interface for Hulu
type HuluScraper struct {
	config     *config.Config
	db         *database.DB
	serviceKey string
}

// NewHuluScraper creates a new Hulu scraper
func NewHuluScraper(cfg *config.Config, db *database.DB) *HuluScraper {
	return &HuluScraper{
		config:     cfg,
		db:         db,
		serviceKey: "Hulu",
	}
}

// Name returns the service name
func (s *HuluScraper) Name() string {
	return s.serviceKey
}

// Scrape fetches "Continue Watching" from Hulu
func (s *HuluScraper) Scrape(ctx context.Context) ([]database.WatchHistory, error) {
	// Get service config
	serviceCfg, ok := s.config.Services["hulu"]
	if !ok || !serviceCfg.Enabled {
		return nil, fmt.Errorf("hulu not configured or not enabled")
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

	// Navigate to Continue Watching
	if err := s.navigateToContinueWatching(chromeCtx); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Extract continue watching items
	items, err := s.extractContinueWatching(chromeCtx)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	log.Printf("Hulu scraper extracted %d items", len(items))
	return items, nil
}

// loadCookies loads authentication cookies into the browser
func (s *HuluScraper) loadCookies(ctx context.Context, cookies []config.Cookie) error {
	// First navigate to hulu.com to set cookies
	if err := chromedp.Run(ctx, chromedp.Navigate("https://www.hulu.com")); err != nil {
		return fmt.Errorf("failed to navigate to hulu.com: %w", err)
	}

	// Wait a moment for the page to load
	time.Sleep(2 * time.Second)

	// Convert and set cookies
	for _, cookie := range cookies {
		expr := cdp.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour))
		if err := chromedp.Run(ctx,
			network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(".hulu.com").
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

// navigateToContinueWatching navigates to the Continue Watching collection
func (s *HuluScraper) navigateToContinueWatching(ctx context.Context) error {
	url := "https://www.hulu.com/hub/home/collection/282"

	log.Printf("Navigating to Hulu Continue Watching: %s", url)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	); err != nil {
		return fmt.Errorf("failed to navigate to continue watching: %w", err)
	}

	// Wait for content to load
	log.Println("Waiting for continue watching content to load...")
	time.Sleep(10 * time.Second)

	return nil
}

// extractContinueWatching extracts continue watching items from the current page
func (s *HuluScraper) extractContinueWatching(ctx context.Context) ([]database.WatchHistory, error) {
	var items []database.WatchHistory
	itemCount := 0

	log.Println("Extracting continue watching from Hulu...")

	// Extract all data using JavaScript to avoid chromedp blocking issues
	var extractedData []map[string]interface{}
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			// Find all subtitle divs (which contain episode info)
			const subtitleDivs = Array.from(document.querySelectorAll('div[data-testid="standard-emphasis-tile-subtitle"]'));

			subtitleDivs.map(subtitleDiv => {
				const episodeInfo = subtitleDiv.textContent.trim();

				// Find the associated title link by searching up the DOM tree
				// The browse-action link is in an ancestor container, not necessarily the immediate parent
				const parentTile = subtitleDiv.closest('div[class*="_6bueqf"]');

				// Try to find the title link in the parent tile first
				let titleLink = parentTile ? parentTile.querySelector('a[data-testid="browse-action"]') : null;

				// If not found in parent tile, search in parent's parent (grandparent)
				if (!titleLink && parentTile) {
					const grandParent = parentTile.parentElement;
					titleLink = grandParent ? grandParent.querySelector('a[data-testid="browse-action"]') : null;
				}

				// If still not found, search up to 5 ancestor levels
				if (!titleLink) {
					let ancestor = subtitleDiv.parentElement;
					for (let i = 0; i < 5 && ancestor && !titleLink; i++) {
						titleLink = ancestor.querySelector('a[data-testid="browse-action"]');
						ancestor = ancestor.parentElement;
					}
				}

				const showTitle = titleLink ? titleLink.textContent.trim() : '';

				return {
					showTitle: showTitle,
					episodeInfo: episodeInfo
				};
			}).filter(item => item.showTitle !== '' && item.episodeInfo !== '');
		`, &extractedData),
	); err != nil {
		return nil, fmt.Errorf("failed to extract data with JavaScript: %w", err)
	}

	log.Printf("JavaScript extracted %d items", len(extractedData))

	// For new items on first import, use yesterday's date
	// On subsequent scrapes, we'll compare with existing DB to find new items
	watchDate := time.Now().AddDate(0, 0, -1) // Yesterday by default

	// Get existing items from database to determine what's new
	existingItems, err := s.getExistingItems()
	if err != nil {
		log.Printf("Warning: failed to get existing items: %v", err)
		existingItems = make(map[string]bool)
	}

	for _, data := range extractedData {
		showTitle, ok := data["showTitle"].(string)
		if !ok || showTitle == "" {
			continue
		}

		episodeInfo, _ := data["episodeInfo"].(string)

		// Combine show title and episode info for the full title
		var fullTitle string
		if episodeInfo != "" {
			fullTitle = fmt.Sprintf("%s - %s", showTitle, episodeInfo)
		} else {
			fullTitle = showTitle
		}

		log.Printf("Processing: %s", fullTitle)

		// Check if this is a new item (not in database)
		isNew := !existingItems[fullTitle]

		if isNew {
			// New item - use yesterday's date
			item := database.WatchHistory{
				Title:           fullTitle,
				DurationMinutes: 0, // Hulu doesn't show duration in continue watching
				WatchedAt:       watchDate,
				EpisodeInfo:     episodeInfo,
				Created:         time.Now(),
			}
			items = append(items, item)
			itemCount++
			log.Printf("Added new item: %s (watched: %s)", fullTitle, watchDate.Format("2006-01-02"))
		} else {
			log.Printf("Skipping existing item: %s", fullTitle)
		}

		if s.config.Scraper.TestMode && itemCount >= s.config.Scraper.TestLimit {
			log.Printf("Test mode: stopping at %d items", s.config.Scraper.TestLimit)
			return items, nil
		}
	}

	log.Printf("Hulu scraper extracted %d new items (skipped %d existing)", len(items), len(extractedData)-len(items))
	return items, nil
}

// getExistingItems returns a set of existing titles in the database for Hulu
func (s *HuluScraper) getExistingItems() (map[string]bool, error) {
	// Get the service ID for Hulu
	service, err := s.db.GetServiceByName(s.serviceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Get all watch history for this service (last 2 years)
	startDate := time.Now().AddDate(-2, 0, 0)
	endDate := time.Now().AddDate(0, 0, 1)
	history, err := s.db.GetWatchHistory(service.ID, startDate, endDate, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch history: %w", err)
	}

	// Build a set of existing titles
	existing := make(map[string]bool)
	for _, item := range history {
		existing[item.Title] = true
	}

	return existing, nil
}
