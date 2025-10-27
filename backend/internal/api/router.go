package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// NewRouter creates and configures the API router
func NewRouter(handler *Handler) http.Handler {
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/health", handler.healthCheck).Methods("GET")
	api.HandleFunc("/services", handler.getServices).Methods("GET")
	api.HandleFunc("/services/{id:[0-9]+}/history", handler.getServiceHistory).Methods("GET")
	api.HandleFunc("/scrape/{service}", handler.triggerScrape).Methods("POST")
	api.HandleFunc("/scraper/status", handler.getScraperStatus).Methods("GET")
	api.HandleFunc("/upload/netflix", handler.uploadNetflixCSV).Methods("POST")

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	return c.Handler(r)
}
