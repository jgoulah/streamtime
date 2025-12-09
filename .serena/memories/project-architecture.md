# StreamTime Project Architecture

## Overview
StreamTime is a streaming watch history aggregator that scrapes viewing history from Netflix, YouTube TV, Amazon Video, and Hulu. It consists of a Go backend (API + scrapers), React frontend, and SQLite database.

## Directory Structure

```
streamtime/
├── backend/
│   ├── cmd/
│   │   ├── scraper/main.go           # Scraper CLI entry point
│   │   ├── server/main.go            # API server entry point
│   │   └── export-cookies/main.go    # Cookie export tool entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler.go            # HTTP request handlers
│   │   │   └── router.go             # API routes + static file serving
│   │   ├── config/
│   │   │   └── config.go             # YAML config parsing
│   │   ├── database/
│   │   │   ├── db.go                 # Database initialization
│   │   │   ├── models.go             # Data models (Service, WatchHistory, etc.)
│   │   │   └── queries.go            # Database queries (GetServiceByName, etc.)
│   │   └── scraper/
│   │       ├── scraper.go            # Scraper interface + Manager
│   │       ├── netflix.go            # Netflix scraper implementation
│   │       ├── youtube_tv.go         # YouTube TV scraper
│   │       ├── amazon.go             # Amazon Video scraper
│   │       └── hulu.go               # Hulu scraper
│   ├── scraper                       # Compiled scraper binary
│   ├── server                        # Compiled server binary
│   └── export-cookies                # Compiled cookie tool binary
├── frontend/
│   ├── src/
│   │   ├── components/               # React components
│   │   ├── services/                 # API client
│   │   └── App.tsx                   # Main app component
│   ├── dist/                         # Built frontend (served by backend)
│   └── package.json                  # Frontend dependencies
├── deployment/
│   ├── streamtime.service            # Systemd service file
│   └── streamtime-sudoers            # Sudoers configuration
├── data/
│   └── streamtime.db                 # SQLite database (local)
├── config.yaml                       # Configuration file (gitignored)
├── Makefile                          # Build and deployment automation
└── README.md                         # Project documentation
```

## Key Components

### Backend Services

#### 1. Scraper (`cmd/scraper`)
- **Purpose**: Extract viewing history from streaming services
- **Entry Point**: `backend/cmd/scraper/main.go`
- **How It Works**:
  - Uses chromedp to automate browser interactions
  - Loads cookies from config.yaml for authentication
  - Extracts data via JavaScript evaluation (avoids chromedp blocking)
  - Stores results in SQLite database
- **Run**: `./backend/scraper --config config.yaml --service "Service Name"`

#### 2. API Server (`cmd/server`)
- **Purpose**: REST API for frontend + serves static frontend
- **Entry Point**: `backend/cmd/server/main.go`
- **Endpoints**:
  - `GET /api/health` - Health check
  - `GET /api/services` - List all services
  - `GET /api/services/{id}/history` - Get watch history for service
  - `POST /api/scrape/{service}` - Trigger scraper for service
  - `GET /api/scraper/status` - Get scraper status
  - `GET /*` - Serve frontend static files (SPA routing)
- **Run**: `./backend/server --config config.yaml`

#### 3. Cookie Export Tool (`cmd/export-cookies`)
- **Purpose**: Interactive tool to capture browser cookies
- **How It Works**:
  - Opens browser to service login page
  - User logs in manually
  - User navigates to target page (e.g., watch history)
  - Tool captures cookies and updates config.yaml
- **Run**: `make refresh-cookies SERVICE=netflix`

### Database Schema

**services**
```sql
CREATE TABLE services (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    color TEXT NOT NULL,          -- Hex color for UI
    logo_url TEXT,                 -- Nullable
    enabled BOOLEAN DEFAULT 1,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**watch_history**
```sql
CREATE TABLE watch_history (
    id INTEGER PRIMARY KEY,
    service_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    duration_minutes INTEGER,
    watched_at TIMESTAMP NOT NULL,
    episode_info TEXT,             -- e.g., "S01E05"
    thumbnail_url TEXT,
    genre TEXT,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES services(id)
);
```

**scraper_runs**
```sql
CREATE TABLE scraper_runs (
    id INTEGER PRIMARY KEY,
    service_id INTEGER NOT NULL,
    ran_at TIMESTAMP NOT NULL,
    status TEXT NOT NULL,          -- "success", "failed"
    error_message TEXT,
    items_scraped INTEGER,
    FOREIGN KEY (service_id) REFERENCES services(id)
);
```

## Scraper Implementation Patterns

### Common Pattern (Netflix, Amazon, Hulu)
1. Load cookies from config
2. Navigate to service page (e.g., viewing activity)
3. Wait for AJAX/React content to load (10-15 seconds)
4. Extract data using JavaScript evaluation:
   ```javascript
   chromedp.Evaluate(`
       Array.from(document.querySelectorAll('.selector')).map(el => ({
           title: el.textContent.trim(),
           // ... more fields
       }))
   `, &extractedData)
   ```
5. Parse and save to database

### YouTube TV Pattern (Special Case)
- Scrapes Google My Activity (myactivity.google.com)
- Splits items into YouTube and YouTube TV based on URL patterns
- Creates entries in both services' watch histories

### Hulu Pattern (No Watch Dates)
- Scrapes "Continue Watching" collection (no full history available)
- Assigns yesterday's date to new items
- Compares with existing DB to avoid duplicates
- With daily scraping, approximates when shows were watched

## Important Technical Details

### chromedp Best Practices
- **Avoid**: `chromedp.Nodes()` and `chromedp.TextContent()` from node contexts (causes blocking)
- **Use**: JavaScript evaluation with `chromedp.Evaluate()` for all DOM queries
- **Pattern**: Extract all data in one JavaScript call, process in Go

### Cookie Authentication
- Cookies are stored in config.yaml per service
- Must be refreshed periodically (streaming services expire sessions)
- Amazon requires waiting for watch history page to load before capturing cookies
- Hulu requires navigating to Continue Watching page after login

### Database Nullable Fields
- `logo_url` in services table is nullable
- Must use `sql.NullString` when scanning these fields:
  ```go
  var logoURL sql.NullString
  rows.Scan(&svc.ID, &logoURL, ...)
  if logoURL.Valid {
      svc.LogoURL = logoURL.String
  }
  ```

## Frontend Architecture

### Technology Stack
- React 18 with TypeScript
- Vite for build/dev server
- Recharts for data visualization
- Fetch API for backend communication

### Build Output
- Production build: `frontend/dist/`
- Served by backend at root path (`/`)
- SPA routing: all non-API routes serve `index.html`

## Configuration Management

### config.yaml
- **Location**: Project root (gitignored)
- **Contains**: Database path, scraper settings, service cookies
- **Critical**: Never commit to git (contains authentication cookies)

### Service Registration
- Services must exist in database before scraping
- Use: `sqlite3 ./data/streamtime.db "INSERT INTO services ..."`
- Current services: Netflix, YouTube TV, Amazon Video, Hulu, YouTube, HBO Max, Apple TV+, Peacock
