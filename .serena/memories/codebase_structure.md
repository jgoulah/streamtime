# Codebase Structure

## Repository Layout

```
streamtime/
├── backend/              # Go backend service
│   ├── cmd/
│   │   ├── server/      # Main server entrypoint
│   │   └── export-cookies/ # Cookie export utility
│   ├── internal/
│   │   ├── api/         # REST API handlers and router
│   │   ├── config/      # Configuration management
│   │   ├── database/    # DB models, queries, operations
│   │   ├── scraper/     # Scraper implementations
│   │   └── scheduler/   # Job scheduling logic
│   ├── go.mod           # Go module definition
│   └── Dockerfile       # Backend container definition
├── frontend/            # React frontend application
│   ├── src/
│   │   ├── pages/       # Page components
│   │   ├── components/  # Reusable UI components
│   │   ├── services/    # API service calls
│   │   ├── utils/       # Utility functions
│   │   ├── assets/      # Static assets
│   │   ├── main.jsx     # Application entry point
│   │   └── App.jsx      # Root component
│   ├── public/          # Public static files
│   ├── package.json     # Node dependencies
│   ├── vite.config.js   # Vite configuration
│   ├── tailwind.config.js # Tailwind configuration
│   └── Dockerfile       # Frontend container definition
├── data/                # SQLite database storage
├── config.yaml          # Application configuration (gitignored)
├── config.example.yaml  # Example configuration template
├── docker-compose.yml   # Multi-container orchestration
├── IMPLEMENTATION_PLAN.md # Development stages and decisions
└── CLAUDE.md           # Development guidelines
```

## Backend Modules

### internal/api
- `handlers.go` - HTTP request handlers
- `router.go` - Route definitions and middleware
- `handlers_test.go` - API endpoint tests

### internal/config
- `config.go` - Configuration loading and validation
- `config_test.go` - Config parsing tests

### internal/database
- `db.go` - Database connection and initialization
- `models.go` - Data structure definitions
- `queries.go` - SQL queries and operations
- `db_test.go` - Database operation tests

### internal/scraper
- `scraper.go` - Base scraper interface/implementation
- `netflix.go` - Netflix scraper
- `youtube_tv.go` - YouTube TV scraper
- `amazon.go` - Amazon Video scraper
- `errors.go` - Custom error types
- Test files for each scraper

### internal/scheduler
- Job scheduling for automated daily scraping

## Frontend Structure

### src/pages
Page-level components representing different routes

### src/components
Reusable UI components (buttons, cards, charts, etc.)

### src/services
API client code for backend communication

### src/utils
Helper functions and utilities

## Database Schema

```sql
services (id, name, color, logo_url)
watch_history (id, service_id, title, duration_minutes, watched_at, episode_info, thumbnail_url, genre)
scraper_runs (id, service_id, ran_at, status, error_message)
```

## API Endpoints

- `GET /api/services` - List all services with current month totals
- `GET /api/services/:id/history?start=&end=` - Get watch history for a service
- `POST /api/scrape/:service` - Manually trigger scraping for a service
- `GET /api/health` - Health check endpoint
