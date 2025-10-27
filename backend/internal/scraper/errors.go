package scraper

import "errors"

var (
	// ErrScraperNotFound is returned when a scraper is not registered
	ErrScraperNotFound = errors.New("scraper not found")

	// ErrServiceNotFound is returned when a service doesn't exist in the database
	ErrServiceNotFound = errors.New("service not found in database")

	// ErrAuthenticationFailed is returned when login fails
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrNavigationFailed is returned when navigation to a page fails
	ErrNavigationFailed = errors.New("navigation failed")

	// ErrNoDataFound is returned when no viewing history is found
	ErrNoDataFound = errors.New("no viewing history data found")

	// ErrTimeout is returned when a scraper operation times out
	ErrTimeout = errors.New("scraper operation timed out")
)
