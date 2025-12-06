package api

import (
	"net/http"
	"os"
	"path/filepath"

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

	// Serve static frontend files if they exist
	frontendDir := "./frontend/dist"
	if _, err := os.Stat(frontendDir); err == nil {
		// Serve static files
		fs := http.FileServer(http.Dir(frontendDir))

		// Serve index.html for all non-API routes (SPA routing)
		r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(frontendDir, r.URL.Path)

			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				// File doesn't exist, serve index.html for SPA routing
				http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
				return
			}

			// File exists, serve it
			fs.ServeHTTP(w, r)
		})
	}

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	return c.Handler(r)
}
