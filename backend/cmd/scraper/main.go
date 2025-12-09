package main

import (
	"context"
	"flag"
	"log"

	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
	"github.com/jgoulah/streamtime/internal/scraper"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "./config.yaml", "Path to config.yaml file")
	dbPath := flag.String("database", "", "Path to database file (overrides config)")
	service := flag.String("service", "", "Specific service to scrape (Netflix, YouTube TV, Amazon Video, Hulu). If empty, scrapes all enabled services")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded configuration from %s", *configPath)

	// Override database path if specified via flag
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at %s", cfg.Database.Path)

	// Initialize scraper manager
	scraperMgr := scraper.NewManager(db, cfg)

	// Register scrapers
	netflixScraper := scraper.NewNetflixScraper(cfg, db)
	scraperMgr.Register(netflixScraper)

	youtubeTVScraper := scraper.NewYouTubeTVScraper(cfg, db)
	scraperMgr.Register(youtubeTVScraper)

	amazonScraper := scraper.NewAmazonScraper(cfg, db)
	scraperMgr.Register(amazonScraper)

	huluScraper := scraper.NewHuluScraper(cfg, db)
	scraperMgr.Register(huluScraper)

	// Create context with timeout
	ctx := context.Background()

	// Run scraper(s)
	if *service != "" {
		// Run specific service
		log.Printf("Running scraper for service: %s", *service)
		result, err := scraperMgr.Run(ctx, *service)
		if err != nil {
			log.Fatalf("Scraper failed: %v", err)
		}
		log.Printf("Scraper completed: %s - %d items scraped in %v",
			result.ServiceName, result.ItemsScraped, result.EndTime.Sub(result.StartTime))
	} else {
		// Run all enabled scrapers
		log.Println("Running all enabled scrapers...")
		results, _ := scraperMgr.RunAll(ctx)
		log.Println("All scrapers completed")
		for _, result := range results {
			if result.Success {
				log.Printf("  ✓ %s: %d items in %v",
					result.ServiceName, result.ItemsScraped, result.EndTime.Sub(result.StartTime))
			} else {
				log.Printf("  ✗ %s: failed - %v", result.ServiceName, result.Error)
			}
		}
	}
}
