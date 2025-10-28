package scraper

import (
	"context"
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

// YouTubeTVScraper implements the Scraper interface for YouTube TV
type YouTubeTVScraper struct {
	config         *config.Config
	db             *database.DB
	serviceKey     string
	serviceIDCache map[string]int64 // Cache service IDs to avoid repeated DB queries
}

// NewYouTubeTVScraper creates a new YouTube TV scraper
func NewYouTubeTVScraper(cfg *config.Config, db *database.DB) *YouTubeTVScraper {
	return &YouTubeTVScraper{
		config:         cfg,
		db:             db,
		serviceKey:     "YouTube TV",
		serviceIDCache: make(map[string]int64),
	}
}

// Name returns the service name
func (s *YouTubeTVScraper) Name() string {
	return s.serviceKey
}

// Scrape fetches viewing history from YouTube TV
func (s *YouTubeTVScraper) Scrape(ctx context.Context) ([]database.WatchHistory, error) {
	// Get service config
	serviceCfg, ok := s.config.Services["youtube_tv"]
	if !ok || !serviceCfg.Enabled {
		return nil, fmt.Errorf("youtube_tv not configured or not enabled")
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
	if err := s.navigateToHistory(chromeCtx); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Extract viewing history
	items, err := s.extractViewingHistory(chromeCtx)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	log.Printf("YouTube TV scraper completed: extracted %d items", len(items))

	return items, nil
}

// loadCookies loads authentication cookies into the browser
func (s *YouTubeTVScraper) loadCookies(ctx context.Context, cookies []config.Cookie) error {
	// First navigate to myactivity.google.com so cookies can be set
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://myactivity.google.com"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}

	// Set cookies on multiple Google domains since myactivity uses cookies from different domains
	domains := []string{".google.com", ".accounts.google.com"}

	cookiesSet := 0
	for _, domain := range domains {
		for _, cookie := range cookies {
			// APISID cookies should not be HTTPOnly
			httpOnly := !strings.Contains(cookie.Name, "APISID")

			expr := network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(domain).
				WithPath("/").
				WithSecure(true).
				WithHTTPOnly(httpOnly)

			if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
				return expr.Do(ctx)
			})); err != nil {
				// Some domains might reject certain cookies, that's okay
				log.Printf("Note: Could not set cookie %s on domain %s", cookie.Name, domain)
			} else {
				cookiesSet++
			}
		}
	}

	log.Printf("Successfully loaded %d cookies across Google domains", cookiesSet)
	return nil
}

// navigateToHistory navigates to the Google My Activity YouTube page
func (s *YouTubeTVScraper) navigateToHistory(ctx context.Context) error {
	log.Println("Navigating to Google My Activity YouTube page...")

	var pageTitle string
	var url string
	var bodyText string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://myactivity.google.com/product/youtube"),
		chromedp.Sleep(5*time.Second), // Wait for page to load
		chromedp.Title(&pageTitle),
		chromedp.Location(&url),
	)

	if err != nil {
		return err
	}

	log.Printf("Page loaded with title: %s, URL: %s", pageTitle, url)

	// Check if we're logged in by looking for sign-in text
	chromedp.Run(ctx,
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	)

	// Log a snippet to see what's on the page
	if len(bodyText) > 200 {
		log.Printf("Page content preview: %s...", bodyText[:200])
	} else {
		log.Printf("Page content preview: %s", bodyText)
	}

	// Check for common sign-in indicators
	if strings.Contains(bodyText, "Sign in") || strings.Contains(bodyText, "sign in") {
		log.Println("WARNING: Page appears to require sign-in. Cookies may need to be updated from google.com domain.")
	} else {
		log.Println("Page appears accessible - proceeding with scraping")
	}

	// Wait for dynamic content to load and scroll to trigger lazy-loading
	log.Println("Waiting for dynamic content to load...")
	chromedp.Run(ctx,
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, 500)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, 1000)`, nil),
		chromedp.Sleep(3*time.Second),
	)

	// Debug: Try to find activity items with various selectors
	var outerCells, dataItems, contentItems, divCount int
	var sampleHTML string
	chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.outer-cell').length`, &outerCells),
		chromedp.Evaluate(`document.querySelectorAll('[data-item-id]').length`, &dataItems),
		chromedp.Evaluate(`document.querySelectorAll('.content-cell, .inner-cell, [role="article"]').length`, &contentItems),
		chromedp.Evaluate(`document.querySelectorAll('div').length`, &divCount),
		chromedp.Evaluate(`(() => {
			// Try to find actual content containers
			const selectors = ['.outer-cell', '.content-cell', '.inner-cell', '[role="article"]', '[data-item-id]'];
			for (let selector of selectors) {
				const items = document.querySelectorAll(selector);
				if (items.length > 0) {
					return selector + " (found " + items.length + "):\n" + items[0].outerHTML.substring(0, 2000);
				}
			}
			// Check if there's any meaningful text content (not just scripts)
			const textNodes = Array.from(document.querySelectorAll('div')).filter(d =>
				d.textContent.trim().length > 20 &&
				!d.textContent.startsWith('window.') &&
				!d.textContent.includes('function')
			);
			if (textNodes.length > 0) {
				return "Found " + textNodes.length + " content divs. Sample text: " + textNodes[0].textContent.substring(0, 500);
			}
			return "No content items found. Page may not have loaded yet or may require user interaction.";
		})()`, &sampleHTML),
	)
	log.Printf("Selector check - outer-cell: %d, data-item-id: %d, content items: %d, total divs: %d", outerCells, dataItems, contentItems, divCount)
	log.Printf("Sample content:\n%s", sampleHTML)

	return chromedp.Run(ctx, s.scrollToLoadItems(ctx))
}

// scrollToLoadItems scrolls through the history to load more items
func (s *YouTubeTVScraper) scrollToLoadItems(ctx context.Context) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		log.Println("Starting to load items with pagination...")

		service, err := s.db.GetServiceByName(s.serviceKey)
		if err != nil {
			return fmt.Errorf("failed to get service: %w", err)
		}

		stableCountIterations := 0
		previousCount := 0
		clickCount := 0
		maxClicks := 200 // Safety limit for full history

		// Check if test mode is enabled
		testMode := s.config.Scraper.TestMode
		testLimit := s.config.Scraper.TestLimit
		if testMode {
			log.Printf("Test mode enabled - will stop after %d items", testLimit)
		}

		for clickCount < maxClicks {
			clickCount++

			// Wait for content to load
			time.Sleep(2 * time.Second)

			// Get current count of items - Google My Activity uses div[jsname="MFYZYe"]
			var currentCount int
			chromedp.Evaluate(`document.querySelectorAll('div[jsname="MFYZYe"]').length`, &currentCount).Do(ctx)

			log.Printf("Iteration %d: Found %d items", clickCount, currentCount)
			// If test mode and we've reached the limit, stop
			if testMode && currentCount >= testLimit {
				log.Printf("Test mode: Reached limit of %d items, stopping pagination", testLimit)
				break
			}

			// Check if count is stable
			if currentCount == previousCount {
				stableCountIterations++
			} else {
				stableCountIterations = 0
			}
			previousCount = currentCount

			// Get the last item's date from Google My Activity
			var lastDateText string
			chromedp.Evaluate(`
				(() => {
					const items = document.querySelectorAll('div[jsname="MFYZYe"]');
					if (items.length > 0) {
						const lastItem = items[items.length - 1];
						// Try to find date text in the item
						return lastItem.textContent.trim().substring(0, 200);
					}
					return '';
				})()
			`, &lastDateText).Do(ctx)

			// Check if this item already exists in database
			if lastDateText != "" {
				// Get title of last item for duplicate check
				var lastTitle string
				chromedp.Evaluate(`
					(() => {
						const videos = document.querySelectorAll('ytd-video-renderer');
						if (videos.length > 0) {
							const lastVideo = videos[videos.length - 1];
							const titleEl = lastVideo.querySelector('#video-title');
							return titleEl ? titleEl.textContent.trim() : '';
						}
						return '';
					})()
				`, &lastTitle).Do(ctx)

				if lastTitle != "" {
					lastDate, err := s.parseDate(lastDateText)
					if err == nil {
						exists, _ := s.db.WatchHistoryExists(service.ID, lastTitle, "", lastDate)
						if exists {
							log.Println("Found existing entry in database, stopping pagination")
							break
						}
					}
				}
			}

			// Check if we should stop (stable for 3 iterations)
			if stableCountIterations >= 3 {
				log.Println("Item count stable for 3 iterations, stopping pagination")
				break
			}

			// Scroll to bottom to trigger more loading
			chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil).Do(ctx)
			time.Sleep(2 * time.Second)
		}

		log.Printf("Pagination complete after %d iterations. Total items: %d", clickCount, previousCount)
		return nil
	}
}

// extractViewingHistory extracts all viewing history items from the page
func (s *YouTubeTVScraper) extractViewingHistory(ctx context.Context) ([]database.WatchHistory, error) {
	var items []database.WatchHistory

	// Get all activity items from Google My Activity
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(`div[jsname="MFYZYe"]`, &nodes, chromedp.ByQueryAll),
	); err != nil {
		return nil, err
	}

	log.Printf("Found %d activity items to extract", len(nodes))

	// Extract data from each video
	for i, node := range nodes {
		item, err := s.extractHistoryItem(ctx, node, i)
		if err != nil {
			log.Printf("Failed to extract item %d: %v", i, err)
			continue
		}

		if item != nil {
			items = append(items, *item)
		}
	}

	return items, nil
}

// extractHistoryItem extracts data from a single Google My Activity item
func (s *YouTubeTVScraper) extractHistoryItem(ctx context.Context, node *cdp.Node, itemIndex int) (*database.WatchHistory, error) {
	var title, timeText, dateHeader, platformLabel string

	// Extract the show title from the link (a.l8sGWb)
	chromedp.Run(ctx,
		chromedp.Text("a.l8sGWb", &title, chromedp.ByQuery, chromedp.FromNode(node)),
	)

	// Extract the platform label to distinguish YouTube vs YouTube TV
	chromedp.Run(ctx,
		chromedp.Text("span.hJ7x8b", &platformLabel, chromedp.ByQuery, chromedp.FromNode(node)),
	)

	// Extract the time from div.wlgrwd (e.g., "6:00 PM • Details")
	chromedp.Run(ctx,
		chromedp.Text("div.wlgrwd", &timeText, chromedp.ByQuery, chromedp.FromNode(node)),
	)

	// Find the date header (.rp10kf) that precedes this item
	// This contains "Yesterday", "Oct 27", etc.
	// Strategy: Compare DOM position of item vs all date headers to find the closest preceding one
	chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`
			(() => {
				const items = document.querySelectorAll('div[jsname="MFYZYe"]');
				if (items.length <= %d) return '';

				const targetItem = items[%d];
				const dateHeaders = document.querySelectorAll('.rp10kf');

				// Find the last date header that comes before this item in document order
				let lastDate = '';
				for (const header of dateHeaders) {
					// Check if header comes before item in document order
					const position = header.compareDocumentPosition(targetItem);
					// DOCUMENT_POSITION_FOLLOWING (4) means targetItem comes after header
					if (position & Node.DOCUMENT_POSITION_FOLLOWING) {
						lastDate = header.textContent.trim();
					} else {
						// Once we find a header that comes after the item, stop
						break;
					}
				}
				return lastDate;
			})()
		`, itemIndex, itemIndex), &dateHeader),
	)

	// Skip items that don't have a title (these are likely category headers or UI elements)
	if title == "" {
		return nil, fmt.Errorf("missing title")
	}

	// Determine which service this belongs to based on platform label
	platformLabel = strings.TrimSpace(platformLabel)
	var serviceName string
	if platformLabel == "YouTube TV" {
		serviceName = "YouTube TV"
	} else if platformLabel == "YouTube" {
		serviceName = "YouTube"
	} else {
		// Unknown platform, skip
		return nil, fmt.Errorf("unknown platform: %s", platformLabel)
	}

	// Look up service ID (with caching)
	serviceID, ok := s.serviceIDCache[serviceName]
	if !ok {
		// Not in cache, query database
		service, err := s.db.GetServiceByName(serviceName)
		if err != nil || service == nil {
			return nil, fmt.Errorf("service not found: %s", serviceName)
		}
		serviceID = service.ID
		s.serviceIDCache[serviceName] = serviceID
	}

	// Combine date header and time to get full timestamp
	// dateHeader is like "Yesterday", "Oct 27", etc.
	// timeText is like "6:00 PM • Details"
	watchedAt := time.Now() // Default to now if we can't parse

	// Extract just the time part (before the bullet)
	timePart := timeText
	if idx := strings.Index(timeText, "•"); idx > 0 {
		timePart = strings.TrimSpace(timeText[:idx])
	}

	if dateHeader != "" && timePart != "" {
		parsed, err := s.parseDateAndTime(dateHeader, timePart)
		if err == nil {
			watchedAt = parsed
		} else {
			log.Printf("Failed to parse date '%s' + time '%s': %v", dateHeader, timePart, err)
		}
	}

	// Build the history item
	item := &database.WatchHistory{
		ServiceID: serviceID, // Set the correct service ID based on platform
		Title:     strings.TrimSpace(title),
		WatchedAt: watchedAt,
	}

	// Store the platform label as episode info for reference
	if platformLabel != "" {
		item.EpisodeInfo = strings.TrimSpace(platformLabel)
	}

	// YouTube doesn't provide duration in history, default to estimate
	item.DurationMinutes = s.estimateDuration(title, "")

	return item, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseDateAndTime combines date header and time to create timestamp
func (s *YouTubeTVScraper) parseDateAndTime(dateHeader, timeStr string) (time.Time, error) {
	now := time.Now()

	// Normalize Unicode spaces (Google uses narrow no-break space U+202F)
	// Replace it with regular space for easier parsing
	timeStr = strings.ReplaceAll(timeStr, "\u202F", " ")

	// Parse the time first (e.g., "6:00 PM")
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(AM|PM)`)
	timeMatches := timeRe.FindStringSubmatch(timeStr)
	if len(timeMatches) != 4 {
		return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
	}

	hour, _ := strconv.Atoi(timeMatches[1])
	minute, _ := strconv.Atoi(timeMatches[2])
	if timeMatches[3] == "PM" && hour != 12 {
		hour += 12
	}
	if timeMatches[3] == "AM" && hour == 12 {
		hour = 0
	}

	// Parse the date header
	dateHeader = strings.TrimSpace(dateHeader)

	// Handle "Yesterday"
	if dateHeader == "Yesterday" {
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), hour, minute, 0, 0, time.Local), nil
	}

	// Handle "Today" (though it's usually not shown)
	if dateHeader == "Today" {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local), nil
	}

	// Handle date format like "Oct 27" or "October 27"
	dateRe := regexp.MustCompile(`([A-Za-z]+)\s+(\d{1,2})`)
	dateMatches := dateRe.FindStringSubmatch(dateHeader)

	if len(dateMatches) >= 3 {
		monthStr := dateMatches[1]
		day, _ := strconv.Atoi(dateMatches[2])

		// Parse month (support both abbreviated and full month names)
		monthMap := map[string]time.Month{
			"Jan": time.January, "January": time.January,
			"Feb": time.February, "February": time.February,
			"Mar": time.March, "March": time.March,
			"Apr": time.April, "April": time.April,
			"May": time.May,
			"Jun": time.June, "June": time.June,
			"Jul": time.July, "July": time.July,
			"Aug": time.August, "August": time.August,
			"Sep": time.September, "September": time.September,
			"Oct": time.October, "October": time.October,
			"Nov": time.November, "November": time.November,
			"Dec": time.December, "December": time.December,
		}

		month, ok := monthMap[monthStr]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown month: %s", monthStr)
		}

		// Determine year (if date is in future, it's from last year)
		year := now.Year()
		testDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		if testDate.After(now) {
			year--
		}

		return time.Date(year, month, day, hour, minute, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date header: %s", dateHeader)
}

// parseDate parses the date string from YouTube TV history (legacy, kept for compatibility)
func (s *YouTubeTVScraper) parseDate(dateStr string) (time.Time, error) {
	// YouTube TV shows dates like "Dec 25 at 8:00 PM" or "Yesterday at 8:00 PM"
	// We'll parse different formats

	now := time.Now()

	// Handle "Yesterday"
	if strings.Contains(dateStr, "Yesterday") {
		yesterday := now.AddDate(0, 0, -1)
		// Try to extract time
		timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(AM|PM)`)
		if matches := timeRe.FindStringSubmatch(dateStr); len(matches) == 4 {
			hour, _ := strconv.Atoi(matches[1])
			minute, _ := strconv.Atoi(matches[2])
			if matches[3] == "PM" && hour != 12 {
				hour += 12
			}
			if matches[3] == "AM" && hour == 12 {
				hour = 0
			}
			return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), hour, minute, 0, 0, time.Local), nil
		}
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 20, 0, 0, 0, time.Local), nil
	}

	// Handle "Today"
	if strings.Contains(dateStr, "Today") {
		timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(AM|PM)`)
		if matches := timeRe.FindStringSubmatch(dateStr); len(matches) == 4 {
			hour, _ := strconv.Atoi(matches[1])
			minute, _ := strconv.Atoi(matches[2])
			if matches[3] == "PM" && hour != 12 {
				hour += 12
			}
			if matches[3] == "AM" && hour == 12 {
				hour = 0
			}
			return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local), nil
		}
		return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local), nil
	}

	// Handle date format like "Dec 25 at 8:00 PM"
	dateRe := regexp.MustCompile(`([A-Za-z]{3})\s+(\d{1,2})`)
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(AM|PM)`)

	dateMatches := dateRe.FindStringSubmatch(dateStr)
	timeMatches := timeRe.FindStringSubmatch(dateStr)

	if len(dateMatches) >= 3 {
		monthStr := dateMatches[1]
		day, _ := strconv.Atoi(dateMatches[2])

		// Parse month
		monthMap := map[string]time.Month{
			"Jan": time.January, "Feb": time.February, "Mar": time.March,
			"Apr": time.April, "May": time.May, "Jun": time.June,
			"Jul": time.July, "Aug": time.August, "Sep": time.September,
			"Oct": time.October, "Nov": time.November, "Dec": time.December,
		}

		month, ok := monthMap[monthStr]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown month: %s", monthStr)
		}

		// Determine year (if date is in future, it's from last year)
		year := now.Year()
		testDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		if testDate.After(now) {
			year--
		}

		hour, minute := 20, 0 // Default time
		if len(timeMatches) == 4 {
			hour, _ = strconv.Atoi(timeMatches[1])
			minute, _ = strconv.Atoi(timeMatches[2])
			if timeMatches[3] == "PM" && hour != 12 {
				hour += 12
			}
			if timeMatches[3] == "AM" && hour == 12 {
				hour = 0
			}
		}

		return time.Date(year, month, day, hour, minute, 0, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// estimateDuration estimates the duration based on title and episode info
func (s *YouTubeTVScraper) estimateDuration(title, episodeInfo string) int {
	// Default estimates
	if episodeInfo != "" {
		// TV show episode: typically 30-60 minutes
		return 45
	}

	// Movies are typically 90-120 minutes
	// For now, default to 60 minutes
	return 60
}
