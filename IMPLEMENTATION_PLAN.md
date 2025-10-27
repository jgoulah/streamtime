# StreamTime Implementation Plan

## Overview
Single-user web application to track screen time across streaming services using web scraping.

**Tech Stack:**
- Backend: Go with SQLite
- Frontend: React with Tailwind CSS
- Infrastructure: Docker Compose
- Scraping: chromedp (headless Chrome)

**Priority Services:** Netflix, YouTube TV, Amazon Video

---

## Stage 1: Project Setup & Infrastructure
**Goal**: Set up project structure, database schema, and Docker configuration
**Success Criteria**:
- Backend and frontend folders created with proper initialization
- SQLite database schema defined
- Docker Compose runs both services
- Config file structure documented

**Tests**:
- Database migrations run successfully
- Docker containers start without errors
- Config file is properly parsed

**Status**: Complete

---

## Stage 2: Backend Core & REST API
**Goal**: Build REST API for retrieving watch history data
**Success Criteria**:
- Database models for services, watch history, and sessions
- RESTful endpoints for:
  - GET /api/services (list all services with monthly totals)
  - GET /api/services/:id/history (detailed history for a service)
  - POST /api/scrape/:service (manual trigger)
  - GET /api/health
- CORS configured for frontend
- Basic error handling

**Tests**:
- Unit tests for database operations
- API endpoint tests
- Config loading tests

**Status**: Complete

---

## Stage 3: Netflix Scraper Implementation
**Goal**: Implement working Netflix viewing history scraper
**Success Criteria**:
- Authenticate with Netflix (credentials + OAuth fallback)
- Navigate to viewing activity page
- Extract: title, duration, date, episode info, thumbnail
- Store data in database without duplicates
- Handle errors gracefully

**Tests**:
- Mock scraper tests
- Data parsing tests
- Database storage tests
- Handle missing data fields

**Status**: Complete

---

## Stage 4: YouTube TV & Amazon Video Scrapers
**Goal**: Implement scrapers for remaining priority services
**Success Criteria**:
- YouTube TV scraper functional
- Amazon Video scraper functional
- Daily scheduler running (cron-like)
- All scrapers use consistent interface
- Logging for scraper runs

**Tests**:
- Scraper integration tests
- Scheduler tests
- Error handling tests

**Status**: Not Started

---

## Stage 5: Frontend Implementation
**Goal**: Build React UI for viewing watch time data
**Success Criteria**:
- Dashboard showing:
  - Monthly total per service
  - Overall monthly total
  - Visual service cards with logos/colors
- Detail page showing:
  - Historical chart (daily/weekly breakdowns)
  - List of shows/movies watched
  - Episode details where available
- Manual scrape trigger button
- Responsive design with Tailwind

**Tests**:
- Component tests
- API integration tests
- Responsive layout verification

**Status**: Complete

---

## Technical Decisions

### Database Schema
```sql
services (id, name, color, logo_url)
watch_history (id, service_id, title, duration_minutes, watched_at, episode_info, thumbnail_url, genre)
scraper_runs (id, service_id, ran_at, status, error_message)
```

### Config File Format (config.yaml)
```yaml
database:
  path: ./data/streamtime.db

services:
  netflix:
    enabled: true
    email: user@example.com
    password: encrypted_password
    use_oauth: false

  youtube_tv:
    enabled: true
    email: user@example.com
    password: encrypted_password
    use_oauth: true

  amazon_video:
    enabled: true
    email: user@example.com
    password: encrypted_password
    use_oauth: false

scraper:
  schedule: "0 3 * * *"  # Daily at 3am
  headless: true
  timeout: 300  # seconds
```

### API Endpoints
- `GET /api/services` - List services with current month totals
- `GET /api/services/:id/history?start=&end=` - Get watch history
- `POST /api/scrape/:service` - Trigger manual scrape
- `GET /api/health` - Health check

---

## Risk & Mitigation

**Risk**: Streaming sites change HTML structure
**Mitigation**: Use flexible selectors, log failures, graceful degradation

**Risk**: Authentication failures (2FA, captchas)
**Mitigation**: OAuth fallback, detailed error messages, manual retry

**Risk**: Rate limiting / account flags
**Mitigation**: Respectful scraping delays, daily-only schedule, user awareness

**Risk**: Incomplete data extraction
**Mitigation**: Store partial data, log missing fields, continue processing
