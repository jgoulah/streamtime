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
	"github.com/jgoulah/streamtime/internal/tmdb"
)

// HuluScraper implements the Scraper interface for Hulu
type HuluScraper struct {
	config     *config.Config
	db         *database.DB
	serviceKey string
	tmdbClient *tmdb.Client
	// Cache for TMDB show IDs and series runtimes
	showIDCache       map[string]int       // showTitle -> TMDB ID
	seriesRuntimeCache map[int]int          // TMDB ID -> typical episode runtime
}

// NewHuluScraper creates a new Hulu scraper
func NewHuluScraper(cfg *config.Config, db *database.DB) *HuluScraper {
	var tmdbClient *tmdb.Client
	if cfg.TMDB.APIKey != "" {
		tmdbClient = tmdb.NewClient(cfg.TMDB.APIKey)
	}

	return &HuluScraper{
		config:             cfg,
		db:                 db,
		serviceKey:         "Hulu",
		tmdbClient:         tmdbClient,
		showIDCache:        make(map[string]int),
		seriesRuntimeCache: make(map[int]int),
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
			// Get episode runtime from TMDB
			runtime := s.getEpisodeRuntime(showTitle, episodeInfo)

			// New item - use yesterday's date
			item := database.WatchHistory{
				Title:           fullTitle,
				DurationMinutes: runtime,
				WatchedAt:       watchDate,
				EpisodeInfo:     episodeInfo,
				Created:         time.Now(),
			}
			items = append(items, item)
			itemCount++
			log.Printf("Added new item: %s (watched: %s, runtime: %d min)", fullTitle, watchDate.Format("2006-01-02"), runtime)
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

// getEpisodeRuntime fetches episode runtime from TMDB
func (s *HuluScraper) getEpisodeRuntime(showTitle, episodeInfo string) int {
	// If TMDB client is not configured, return 30 as default
	if s.tmdbClient == nil {
		return 30
	}

	// Try to parse episode information (S16 E3 format)
	season, episode, ok := tmdb.ParseEpisodeInfo(episodeInfo)
	if !ok {
		log.Printf("Could not parse episode info: %s, using series runtime", episodeInfo)
		// Fall back to series-level runtime
		return s.getSeriesRuntime(showTitle)
	}

	// Check if we have the show ID cached
	showID, ok := s.showIDCache[showTitle]
	if !ok {
		// Search for the show
		results, err := s.tmdbClient.SearchTVShow(showTitle)
		if err != nil {
			log.Printf("TMDB search failed for '%s': %v, using default", showTitle, err)
			return 30
		}

		if len(results.Results) == 0 {
			log.Printf("No TMDB results for '%s', using default", showTitle)
			return 30
		}

		// Use the first result
		showID = results.Results[0].ID
		s.showIDCache[showTitle] = showID
		log.Printf("Found TMDB ID %d for '%s'", showID, showTitle)
	}

	// Try to get specific episode runtime
	ep, err := s.tmdbClient.GetEpisode(showID, season, episode)
	if err != nil {
		log.Printf("Failed to get episode S%dE%d for show ID %d: %v, trying series runtime", season, episode, showID, err)
		return s.getSeriesRuntime(showTitle)
	}

	// If episode has runtime, use it
	if ep.Runtime != nil && *ep.Runtime > 0 {
		log.Printf("TMDB: '%s' S%dE%d runtime: %d minutes", showTitle, season, episode, *ep.Runtime)
		return *ep.Runtime
	}

	// Fall back to series runtime
	log.Printf("Episode runtime not available for '%s' S%dE%d, using series runtime", showTitle, season, episode)
	return s.getSeriesRuntime(showTitle)
}

// getSeriesRuntime gets the typical episode runtime for a series
func (s *HuluScraper) getSeriesRuntime(showTitle string) int {
	// Check if we have the show ID cached
	showID, ok := s.showIDCache[showTitle]
	if !ok {
		// Search for the show
		results, err := s.tmdbClient.SearchTVShow(showTitle)
		if err != nil {
			log.Printf("TMDB search failed for '%s': %v, using default", showTitle, err)
			return 30
		}

		if len(results.Results) == 0 {
			log.Printf("No TMDB results for '%s', using default", showTitle)
			return 30
		}

		showID = results.Results[0].ID
		s.showIDCache[showTitle] = showID
	}

	// Check if we have series runtime cached
	if runtime, ok := s.seriesRuntimeCache[showID]; ok {
		return runtime
	}

	// Get series details
	details, err := s.tmdbClient.GetTVShowDetails(showID)
	if err != nil {
		log.Printf("Failed to get series details for show ID %d: %v, using default", showID, err)
		return 30
	}

	// Use the first episode runtime from the series, or 30 if not available
	runtime := 30
	if len(details.EpisodeRunTime) > 0 && details.EpisodeRunTime[0] > 0 {
		runtime = details.EpisodeRunTime[0]
		log.Printf("TMDB: '%s' series runtime: %d minutes", showTitle, runtime)
	} else {
		log.Printf("No series runtime for '%s', using default 30 minutes", showTitle)
	}

	// Cache it
	s.seriesRuntimeCache[showID] = runtime

	return runtime
}
