package scraper

import (
	"context"
	"time"

	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
)

// Scraper defines the interface for all service scrapers
type Scraper interface {
	// Name returns the service name (e.g., "netflix", "youtube_tv")
	Name() string

	// Scrape fetches viewing history and returns the items scraped
	Scrape(ctx context.Context) ([]database.WatchHistory, error)
}

// Result contains the outcome of a scraper run
type Result struct {
	ServiceName  string
	ItemsScraped int
	Success      bool
	Error        error
	StartTime    time.Time
	EndTime      time.Time
}

// Manager coordinates multiple scrapers
type Manager struct {
	scrapers map[string]Scraper
	db       *database.DB
	config   *config.Config
}

// NewManager creates a new scraper manager
func NewManager(db *database.DB, cfg *config.Config) *Manager {
	return &Manager{
		scrapers: make(map[string]Scraper),
		db:       db,
		config:   cfg,
	}
}

// Register adds a scraper to the manager
func (m *Manager) Register(scraper Scraper) {
	m.scrapers[scraper.Name()] = scraper
}

// Run executes a specific scraper by name
func (m *Manager) Run(ctx context.Context, serviceName string) (*Result, error) {
	scraper, ok := m.scrapers[serviceName]
	if !ok {
		return nil, ErrScraperNotFound
	}

	result := &Result{
		ServiceName: serviceName,
		StartTime:   time.Now(),
	}

	// Get service from database
	service, err := m.db.GetServiceByName(serviceName)
	if err != nil {
		result.Error = err
		result.EndTime = time.Now()
		return result, err
	}

	if service == nil {
		result.Error = ErrServiceNotFound
		result.EndTime = time.Now()
		return result, ErrServiceNotFound
	}

	// Run the scraper
	items, err := scraper.Scrape(ctx)
	result.EndTime = time.Now()

	if err != nil {
		result.Error = err
		result.Success = false

		// Record failed scraper run
		m.db.InsertScraperRun(&database.ScraperRun{
			ServiceID:    service.ID,
			RanAt:        result.StartTime,
			Status:       "failed",
			ErrorMessage: err.Error(),
			ItemsScraped: 0,
		})

		return result, err
	}

	// Store items in database
	for i := range items {
		// Only set ServiceID if not already set by the scraper
		// (Some scrapers like YouTube set it themselves to split items across services)
		if items[i].ServiceID == 0 {
			items[i].ServiceID = service.ID
		}
		if err := m.db.InsertWatchHistory(&items[i]); err != nil {
			// Log error but continue processing other items
			continue
		}
	}

	result.ItemsScraped = len(items)
	result.Success = true

	// Record successful scraper run
	m.db.InsertScraperRun(&database.ScraperRun{
		ServiceID:    service.ID,
		RanAt:        result.StartTime,
		Status:       "success",
		ErrorMessage: "",
		ItemsScraped: len(items),
	})

	return result, nil
}

// RunAll executes all registered scrapers
func (m *Manager) RunAll(ctx context.Context) ([]*Result, error) {
	var results []*Result

	for name := range m.scrapers {
		result, err := m.Run(ctx, name)
		if err != nil {
			// Continue with other scrapers even if one fails
			results = append(results, result)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// GetScraper returns a scraper by name
func (m *Manager) GetScraper(name string) (Scraper, bool) {
	scraper, ok := m.scrapers[name]
	return scraper, ok
}
