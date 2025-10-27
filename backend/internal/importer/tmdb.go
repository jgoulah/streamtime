package importer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// TMDBClient is a client for The Movie Database API
type TMDBClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewTMDBClient creates a new TMDB API client
func NewTMDBClient(apiKey string) *TMDBClient {
	return &TMDBClient{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SearchResult represents a search result from TMDB
type SearchResult struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Name         string `json:"name"` // For TV shows
	MediaType    string `json:"media_type"`
	ReleaseDate  string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
}

// SearchResponse represents the search API response
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// MovieDetails represents movie details from TMDB
type MovieDetails struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Runtime int    `json:"runtime"` // in minutes
}

// TVShowDetails represents TV show details from TMDB
type TVShowDetails struct {
	ID              int   `json:"id"`
	Name            string `json:"name"`
	EpisodeRunTime []int `json:"episode_run_time"` // array of runtimes in minutes
}

// ContentInfo represents the duration information for a title
type ContentInfo struct {
	Title         string
	DurationMinutes int
	MediaType     string // "movie" or "tv"
}

// SearchTitle searches for a title and returns duration information
func (c *TMDBClient) SearchTitle(title string) (*ContentInfo, error) {
	// First, search for the title
	searchURL := fmt.Sprintf("%s/search/multi?api_key=%s&query=%s",
		c.baseURL, c.apiKey, url.QueryEscape(title))

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search TMDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: status %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	if len(searchResp.Results) == 0 {
		return nil, fmt.Errorf("no results found for: %s", title)
	}

	// Get the first result
	result := searchResp.Results[0]

	// Fetch detailed information based on media type
	if result.MediaType == "movie" || (result.Title != "" && result.MediaType == "") {
		return c.getMovieDetails(result.ID, title)
	} else if result.MediaType == "tv" || result.Name != "" {
		return c.getTVShowDetails(result.ID, title)
	}

	return nil, fmt.Errorf("unknown media type for: %s", title)
}

// getMovieDetails fetches movie details including runtime
func (c *TMDBClient) getMovieDetails(movieID int, title string) (*ContentInfo, error) {
	detailsURL := fmt.Sprintf("%s/movie/%d?api_key=%s",
		c.baseURL, movieID, c.apiKey)

	resp, err := c.httpClient.Get(detailsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: status %d", resp.StatusCode)
	}

	var details MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode movie details: %w", err)
	}

	return &ContentInfo{
		Title:         title,
		DurationMinutes: details.Runtime,
		MediaType:     "movie",
	}, nil
}

// getTVShowDetails fetches TV show details including episode runtime
func (c *TMDBClient) getTVShowDetails(tvID int, title string) (*ContentInfo, error) {
	detailsURL := fmt.Sprintf("%s/tv/%d?api_key=%s",
		c.baseURL, tvID, c.apiKey)

	resp, err := c.httpClient.Get(detailsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TV show details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: status %d", resp.StatusCode)
	}

	var details TVShowDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode TV show details: %w", err)
	}

	// Use average episode runtime if multiple values exist
	avgRuntime := 40 // default fallback
	if len(details.EpisodeRunTime) > 0 {
		sum := 0
		for _, runtime := range details.EpisodeRunTime {
			sum += runtime
		}
		avgRuntime = sum / len(details.EpisodeRunTime)
	}

	return &ContentInfo{
		Title:         title,
		DurationMinutes: avgRuntime,
		MediaType:     "tv",
	}, nil
}
