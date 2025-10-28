package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jgoulah/streamtime/internal/api"
	"github.com/jgoulah/streamtime/internal/config"
	"github.com/jgoulah/streamtime/internal/database"
	"github.com/jgoulah/streamtime/internal/scraper"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded configuration from %s", configPath)

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

	log.Println("Scraper manager initialized with Netflix and YouTube TV scrapers")

	// Create API handler
	handler := api.NewHandler(db, scraperMgr, cfg)
	router := api.NewRouter(handler)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	log.Printf("Server starting on %s", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("Server stopped")
}
