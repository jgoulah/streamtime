package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const (
	baseURL = "https://api.themoviedb.org/3"
)

// Client handles interactions with The Movie Database API
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new TMDB API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SearchResult represents a TV show search result
type SearchResult struct {
	Page         int        `json:"page"`
	Results      []TVShow   `json:"results"`
	TotalPages   int        `json:"total_pages"`
	TotalResults int        `json:"total_results"`
}

// TVShow represents a TV show
type TVShow struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	OriginalName    string   `json:"original_name"`
	FirstAirDate    string   `json:"first_air_date"`
	EpisodeRunTime  []int    `json:"episode_run_time"`
}

// Episode represents a TV episode with runtime
type Episode struct {
	AirDate        string  `json:"air_date"`
	EpisodeNumber  int     `json:"episode_number"`
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Overview       string  `json:"overview"`
	Runtime        *int    `json:"runtime"` // Pointer to handle null values
	SeasonNumber   int     `json:"season_number"`
	StillPath      string  `json:"still_path"`
	VoteAverage    float64 `json:"vote_average"`
	VoteCount      int     `json:"vote_count"`
}

// TVShowDetails represents detailed TV show information
type TVShowDetails struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	EpisodeRunTime []int  `json:"episode_run_time"`
}

// SearchTVShow searches for a TV show by name
func (c *Client) SearchTVShow(name string) (*SearchResult, error) {
	params := url.Values{}
	params.Add("query", name)
	params.Add("language", "en-US")
	params.Add("page", "1")

	u := fmt.Sprintf("%s/search/tv?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetEpisode retrieves episode details including runtime
func (c *Client) GetEpisode(seriesID, seasonNumber, episodeNumber int) (*Episode, error) {
	u := fmt.Sprintf("%s/tv/%d/season/%d/episode/%d?language=en-US",
		baseURL, seriesID, seasonNumber, episodeNumber)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var episode Episode
	if err := json.NewDecoder(resp.Body).Decode(&episode); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &episode, nil
}

// GetTVShowDetails retrieves TV show details including series-level runtime
func (c *Client) GetTVShowDetails(seriesID int) (*TVShowDetails, error) {
	u := fmt.Sprintf("%s/tv/%d?language=en-US", baseURL, seriesID)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var details TVShowDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &details, nil
}

// ParseEpisodeInfo extracts season and episode numbers from episode info strings
// Examples: "S16 E3", "S1 E10", "Season 2, Episode 5"
func ParseEpisodeInfo(episodeInfo string) (season, episode int, ok bool) {
	// Pattern 1: S16 E3 format
	re1 := regexp.MustCompile(`S(\d+)\s*E(\d+)`)
	if matches := re1.FindStringSubmatch(episodeInfo); len(matches) == 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
		return season, episode, true
	}

	// Pattern 2: Season 2, Episode 5 format
	re2 := regexp.MustCompile(`Season\s+(\d+).*Episode\s+(\d+)`)
	if matches := re2.FindStringSubmatch(episodeInfo); len(matches) == 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
		return season, episode, true
	}

	return 0, 0, false
}
