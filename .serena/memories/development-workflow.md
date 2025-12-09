# StreamTime Development Workflow

## Code Sync & Deployment

### Remote Host
- **Production Host**: `mediaserver` (SSH alias)
- **Remote Path**: `/opt/streamtime/` (binaries)
- **Config Path**: `/usr/local/etc/streamtime/` (config.yaml, database)

### Build Commands
```bash
# Local builds (on macOS)
make build-scraper          # Build scraper binary
make build-server           # Build API server binary
make build-export-cookies   # Build cookie export tool
make build-frontend         # Build React frontend (Vite)
make build-all             # Build all components

# Remote builds (cross-compile for Linux)
make build-remote          # Build scraper + server for Linux
make build-scraper-remote  # Build just scraper for Linux
```

### Deployment
```bash
# Deploy to remote server (builds, copies, restarts service)
make install-remote

# This does:
# 1. Cross-compiles binaries for Linux
# 2. Builds frontend (npm run build)
# 3. Copies to mediaserver via SCP
# 4. Restarts systemd service
```

### Systemd Service
- **Service Name**: `streamtime`
- **Service File**: `deployment/streamtime.service`
- **Commands** (passwordless sudo configured):
  ```bash
  sudo systemctl start streamtime
  sudo systemctl stop streamtime
  sudo systemctl restart streamtime
  sudo systemctl status streamtime
  ```

## Testing Approach

### Local Testing
1. **Scraper Testing**:
   ```bash
   ./backend/scraper --config config.yaml --service "Service Name"
   # Services: Netflix, YouTube TV, Amazon Video, Hulu
   ```

2. **Server Testing**:
   ```bash
   ./backend/server --config config.yaml
   # Runs on http://localhost:8080
   ```

3. **Frontend Testing**:
   ```bash
   cd frontend
   npm run dev
   # Runs on http://localhost:5173 (development mode)
   ```

### Cookie Management
```bash
# Export cookies interactively (opens browser)
make refresh-cookies SERVICE=netflix
make refresh-cookies SERVICE=youtube_tv
make refresh-cookies SERVICE=amazon_video
make refresh-cookies SERVICE=hulu

# Validate existing cookies
./backend/export-cookies --service netflix --validate
```

### Database
- **Local**: `./data/streamtime.db`
- **Remote**: `/usr/local/etc/streamtime/streamtime.db`
- **Query**: `sqlite3 ./data/streamtime.db "SELECT ..."`

## Build Tools & Dependencies

### Backend (Go)
- Go 1.21+
- chromedp for browser automation
- SQLite database
- Mux router, CORS middleware

### Frontend (React + TypeScript)
- Vite build tool
- React 18
- TypeScript
- Recharts for visualization

## Configuration

### config.yaml Structure
```yaml
database:
  path: ./data/streamtime.db
scraper:
  headless: true
  timeout: 300
  test_mode: false
  test_limit: 100
server:
  host: 0.0.0.0
  port: 8080
services:
  netflix: { cookies: [...], enabled: true }
  youtube_tv: { cookies: [...], enabled: true }
  amazon_video: { cookies: [...], enabled: true }
  hulu: { cookies: [...], enabled: true }
```

## Common Development Tasks

### Adding a New Scraper
1. Create `backend/internal/scraper/servicename.go`
2. Implement `Scraper` interface (Name(), Scrape())
3. Register in `backend/cmd/scraper/main.go`
4. Add to `backend/cmd/export-cookies/main.go`
5. Add service to database: `INSERT INTO services (name, color, enabled) VALUES (...)`
6. Add config section to `config.yaml`

### Debugging Scrapers
- Set `headless: false` in config.yaml to see browser
- Use JavaScript evaluation instead of chromedp selectors (avoids blocking)
- Check cookie expiration - refresh if needed
- Wait longer for AJAX/React content to load (15+ seconds)

### Git Workflow
- Main branch: `main`
- No specific branch strategy documented
- Commits include: file changes, build outputs, cookie refreshes
