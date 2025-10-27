package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

// NetflixScraper implements the Scraper interface for Netflix
type NetflixScraper struct {
	config     *config.Config
	db         *database.DB
	serviceKey string
}

// NewNetflixScraper creates a new Netflix scraper
func NewNetflixScraper(cfg *config.Config, db *database.DB) *NetflixScraper {
	return &NetflixScraper{
		config:     cfg,
		db:         db,
		serviceKey: "Netflix",
	}
}

// Name returns the service name
func (s *NetflixScraper) Name() string {
	return s.serviceKey
}

// Scrape fetches viewing history from Netflix
func (s *NetflixScraper) Scrape(ctx context.Context) ([]database.WatchHistory, error) {
	// Get service config
	serviceCfg, ok := s.config.Services["netflix"]
	if !ok || !serviceCfg.Enabled {
		return nil, fmt.Errorf("netflix not configured or not enabled")
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

	// Navigate to viewing activity
	if err := s.navigateToViewingActivity(chromeCtx); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Extract viewing history
	items, err := s.extractViewingHistory(chromeCtx)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return items, nil
}

// loadCookies loads authentication cookies into the browser session
func (s *NetflixScraper) loadCookies(ctx context.Context, cookies []config.Cookie) error {
	log.Println("Loading Netflix authentication cookies...")

	if len(cookies) == 0 {
		return fmt.Errorf("no cookies provided - please configure Netflix cookies in config.yaml")
	}

	// Navigate to Netflix first to set the domain
	if err := chromedp.Run(ctx, chromedp.Navigate("https://www.netflix.com")); err != nil {
		return fmt.Errorf("failed to navigate to Netflix: %w", err)
	}

	// Set each cookie
	for _, cookie := range cookies {
		log.Printf("Setting cookie: %s", cookie.Name)

		expr := network.SetCookie(cookie.Name, cookie.Value).
			WithDomain(".netflix.com").
			WithPath("/").
			WithSecure(true).
			WithHTTPOnly(true)

		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			return expr.Do(ctx)
		})); err != nil {
			return fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err)
		}
	}

	log.Printf("Successfully loaded %d cookies", len(cookies))
	return nil
}

// navigateToViewingActivity navigates to the viewing activity page
func (s *NetflixScraper) navigateToViewingActivity(ctx context.Context) error {
	log.Println("Navigating to viewing activity page...")

	viewingActivityURL := "https://www.netflix.com/viewingactivity"

	err := chromedp.Run(ctx,
		chromedp.Navigate(viewingActivityURL),
		chromedp.WaitVisible(`.retableRow`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Allow page to fully load
	)

	if err != nil {
		return ErrNavigationFailed
	}

	log.Println("Successfully navigated to viewing activity")
	return nil
}

// extractViewingHistory extracts viewing history from the page
func (s *NetflixScraper) extractViewingHistory(ctx context.Context) ([]database.WatchHistory, error) {
	log.Println("Extracting viewing history...")

	// Scroll to load more items (Netflix loads lazily)
	err := s.scrollToLoadItems(ctx)
	if err != nil {
		return nil, err
	}

	// Extract the viewing activity items
	var htmlContent string
	err = chromedp.Run(ctx,
		chromedp.InnerHTML(`.retableRow`, &htmlContent, chromedp.ByQueryAll),
	)

	if err != nil {
		return nil, ErrNoDataFound
	}

	// Get all row elements
	var nodes []*cdp.Node
	err = chromedp.Run(ctx,
		chromedp.Nodes(`.retableRow`, &nodes, chromedp.ByQueryAll),
	)

	if err != nil || len(nodes) == 0 {
		return nil, ErrNoDataFound
	}

	log.Printf("Found %d viewing activity items", len(nodes))

	var items []database.WatchHistory

	// Extract data from each row
	for _, node := range nodes {
		item, err := s.parseViewingActivityRow(ctx, node)
		if err != nil {
			log.Printf("Error parsing row: %v", err)
			continue
		}
		items = append(items, item)
	}

	log.Printf("Successfully extracted %d items", len(items))
	return items, nil
}

// scrollToLoadItems clicks "Show More" button to load more items until we reach existing data or 2024
func (s *NetflixScraper) scrollToLoadItems(ctx context.Context) error {
	log.Println("Loading viewing history (will stop at existing data or year 2024)...")

	previousCount := 0
	stableCountIterations := 0
	targetYear := 2025
	serviceID := int64(1) // Netflix service ID
	clickCount := 0

	for {
		clickCount++
		// Try to click the "Show More" button
		var showMoreExists bool
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelector('button.btn-blue.btn-small') !== null`, &showMoreExists),
		)
		if err != nil {
			log.Printf("Error checking for Show More button: %v", err)
		}

		// If Show More button exists, click it
		if showMoreExists {
			err = chromedp.Run(ctx,
				chromedp.Click(`button.btn-blue.btn-small`, chromedp.ByQuery),
				chromedp.Sleep(2*time.Second), // Wait for items to load
			)
			if err != nil {
				log.Printf("Error clicking Show More button: %v", err)
			}
		}

		// Count current number of items
		var currentCount int
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelectorAll('.retableRow').length`, &currentCount),
		)
		if err != nil {
			log.Printf("Error counting items: %v", err)
			continue
		}

		// Track if count is stable
		if currentCount == previousCount {
			stableCountIterations++
		} else {
			stableCountIterations = 0
		}

		// Get the last visible item's details
		var lastTitleText string
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				const rows = document.querySelectorAll('.retableRow');
				const lastRow = rows[rows.length - 1];
				if (lastRow) {
					const title = lastRow.querySelector('.title') ? lastRow.querySelector('.title').textContent : '';
					const date = lastRow.querySelector('.date') ? lastRow.querySelector('.date').textContent : '';
					JSON.stringify({title: title, date: date});
				} else {
					'{}';
				}
			`, &lastTitleText),
		)

		if err == nil && lastTitleText != "" {
			// Parse the JSON response
			var lastItem struct {
				Title string `json:"title"`
				Date  string `json:"date"`
			}
			if err := json.Unmarshal([]byte(lastTitleText), &lastItem); err == nil && lastItem.Date != "" {
				// Parse the date
				lastDate, dateErr := s.parseDate(strings.TrimSpace(lastItem.Date))

				// Check if we've reached 2024 or earlier
				if dateErr == nil && lastDate.Year() < targetYear {
					log.Printf("Reached year %d at click %d. Stopping. Total items: %d", lastDate.Year(), clickCount, currentCount)
					break
				}

				// Check if this item already exists in the database
				if dateErr == nil {
					title := strings.TrimSpace(lastItem.Title)
					episodeInfo := ""

					// Extract episode info if present
					if strings.Contains(title, ":") {
						parts := strings.SplitN(title, ":", 2)
						if len(parts) == 2 {
							title = strings.TrimSpace(parts[0])
							episodeInfo = strings.TrimSpace(parts[1])
						}
					}

					// Check if entry exists
					exists, checkErr := s.db.WatchHistoryExists(serviceID, title, episodeInfo, lastDate)
					if checkErr == nil && exists {
						log.Printf("Found existing entry '%s' from %s at click %d. Stopping pagination. Total items: %d",
							title, lastDate.Format("2006-01-02"), clickCount, currentCount)
						break
					}
				}
			}
		}

		// Log progress every 10 clicks or when count changes
		if clickCount%10 == 0 || currentCount != previousCount {
			log.Printf("Click %d: Found %d items (prev: %d), Show More button: %v", clickCount, currentCount, previousCount, showMoreExists)
		}

		// If count has been stable for 3 iterations, we're done
		if stableCountIterations >= 3 {
			log.Printf("Count stable for %d iterations. Total items: %d", stableCountIterations, currentCount)
			break
		}

		// If no Show More button and count hasn't changed, we're done
		if !showMoreExists && currentCount == previousCount {
			log.Printf("No Show More button found and count stable. Total items: %d", currentCount)
			break
		}

		previousCount = currentCount
	}

	log.Printf("Finished loading. Total items available: %d", previousCount)
	return nil
}

// parseViewingActivityRow parses a single viewing activity row
func (s *NetflixScraper) parseViewingActivityRow(ctx context.Context, node *cdp.Node) (database.WatchHistory, error) {
	var item database.WatchHistory

	// Extract title
	var title string
	chromedp.Run(ctx,
		chromedp.Text(`.title`, &title, chromedp.ByQuery, chromedp.FromNode(node)),
	)
	item.Title = strings.TrimSpace(title)

	// Extract date
	var dateStr string
	chromedp.Run(ctx,
		chromedp.Text(`.date`, &dateStr, chromedp.ByQuery, chromedp.FromNode(node)),
	)

	// Parse date
	watchedAt, err := s.parseDate(dateStr)
	if err != nil {
		return item, fmt.Errorf("failed to parse date: %w", err)
	}
	item.WatchedAt = watchedAt

	// Try to extract episode info
	if strings.Contains(title, ":") {
		parts := strings.SplitN(title, ":", 2)
		if len(parts) == 2 {
			item.Title = strings.TrimSpace(parts[0])
			item.EpisodeInfo = strings.TrimSpace(parts[1])
		}
	}

	// Extract season/episode pattern (e.g., "S1:E5" or "Season 1: Episode 5")
	episodePattern := regexp.MustCompile(`[Ss](\d+):?\s*[Ee](\d+)`)
	if matches := episodePattern.FindStringSubmatch(item.EpisodeInfo); len(matches) > 0 {
		item.EpisodeInfo = fmt.Sprintf("S%02sE%02s", matches[1], matches[2])
	}

	// Set default duration (Netflix doesn't always show duration on viewing activity)
	// We'll estimate based on title type
	item.DurationMinutes = s.estimateDuration(item.Title, item.EpisodeInfo)

	return item, nil
}

// parseDate parses Netflix date format
func (s *NetflixScraper) parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Netflix typically shows dates like "1/15/25" (MM/DD/YY)
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

// estimateDuration estimates content duration based on title and episode info
func (s *NetflixScraper) estimateDuration(title, episodeInfo string) int {
	// If it has episode info, likely a TV show episode (average 30-45 min)
	if episodeInfo != "" {
		return 40
	}

	// Otherwise, assume it's a movie (average 90-120 min)
	return 105
}

// Helper to convert season/episode string to structured format
func parseEpisodeInfo(episodeStr string) (season int, episode int, err error) {
	// Match patterns like "S01E05", "S1E5", "Season 1: Episode 5"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[Ss](\d+):?[Ee](\d+)`),
		regexp.MustCompile(`Season\s+(\d+).*Episode\s+(\d+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(episodeStr)
		if len(matches) == 3 {
			season, _ = strconv.Atoi(matches[1])
			episode, _ = strconv.Atoi(matches[2])
			return season, episode, nil
		}
	}

	return 0, 0, fmt.Errorf("unable to parse episode info: %s", episodeStr)
}
