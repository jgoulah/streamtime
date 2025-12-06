# StreamTime

Personal streaming service watch time tracker. Monitors and displays screen time across Netflix, YouTube TV, Amazon Video, and other streaming platforms.

## Features

- 📊 **Dashboard View**: Monthly watch time totals per service
- 📈 **Detailed Analytics**: Historical trends, shows watched, episode breakdowns
- 🔄 **Automated Scraping**: Daily collection of viewing history
- 🎯 **Single User**: Simple setup, no authentication required

## Tech Stack

- **Backend**: Go with SQLite database
- **Frontend**: React with Tailwind CSS
- **Infrastructure**: Docker Compose
- **Scraping**: chromedp (headless Chrome)

## Setup

### Prerequisites

- Docker and Docker Compose
- Or: Go 1.21+, Node.js 18+, Chrome/Chromium

### Configuration

1. Copy the example config:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Export authentication cookies for your streaming services:
   ```bash
   # Using Make (recommended)
   make refresh-cookies SERVICE=netflix
   make refresh-cookies SERVICE=youtube_tv
   make refresh-cookies SERVICE=amazon_video

   # Or directly
   ./backend/export-cookies --service netflix
   ./backend/export-cookies --service youtube_tv
   ./backend/export-cookies --service amazon_video
   ```

   The tool will:
   - Open a browser window for you to log in
   - Automatically extract and validate cookies
   - Update `config.yaml` with the correct configuration

   See [Cookie Export Tool Documentation](backend/cmd/export-cookies/README.md) for details.

3. (Optional) Configure scraping schedule (default: daily at 3 AM)

## Building and Running

### Using Make (Recommended)

The project includes a Makefile for common tasks:

```bash
# Show all available commands
make help

# Build the backend server
make build

# Build all binaries (server + export-cookies)
make build-all

# Run the server locally
make run

# Cookie management
make refresh-cookies SERVICE=netflix      # Refresh Netflix cookies
make refresh-cookies SERVICE=youtube_tv   # Refresh YouTube TV cookies
make refresh-cookies SERVICE=amazon_video # Refresh Amazon Video cookies

# Docker commands
make docker-build    # Build Docker images
make docker-up       # Start Docker containers
make docker-down     # Stop Docker containers
make docker-logs     # Show backend logs

# Clean build artifacts
make clean
```

### Running with Docker

```bash
make docker-up
# or
docker-compose up -d
```

Access the app at `http://localhost:3000`

### Running Locally

**Backend:**
```bash
make run
# or manually:
cd backend
go mod download
go run cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm start
```

### Remote Installation

To install the backend on a remote server for cron-based scraping:

```bash
# Using defaults (mediaserver host, /opt/streamtime for binaries, /usr/local/etc/streamtime for config/db)
make install-remote

# Or override paths
export REMOTE_HOST=user@your-server.com
export REMOTE_BIN_DIR=/opt/streamtime
export REMOTE_ETC_DIR=/usr/local/etc/streamtime
make install-remote
```

This will:
- Build both `server` and `export-cookies` binaries
- Create remote directories if needed
- Copy binaries to `/opt/streamtime`
- Copy `config.yaml` to `/usr/local/etc/streamtime`

On the remote host, run:
```bash
# Using the installed paths
/opt/streamtime/server --config /usr/local/etc/streamtime/config.yaml --database /usr/local/etc/streamtime/streamtime.db

# Or with custom paths
/opt/streamtime/server --config /path/to/config.yaml --database /path/to/streamtime.db
```

**Command-line flags:**

**Server** (`/opt/streamtime/server`):
- `--config <path>` - Path to config.yaml file (default: `./config.yaml`)
- `--database <path>` - Path to database file, overrides config setting (optional)

**Scraper** (`/opt/streamtime/scraper`):
- `--config <path>` - Path to config.yaml file (default: `./config.yaml`)
- `--database <path>` - Path to database file, overrides config setting (optional)
- `--service <name>` - Specific service to scrape (netflix, youtube_tv, amazon_video). If omitted, runs all enabled services.

### Cron Setup

To run the scraper daily at 6am, add to crontab (`crontab -e`):

```cron
# Run StreamTime scraper daily at 6am
0 6 * * * /opt/streamtime/scraper --config /usr/local/etc/streamtime/config.yaml --database /usr/local/etc/streamtime/streamtime.db >> /var/log/streamtime.log 2>&1
```

Or for a specific service:
```cron
# Run only YouTube TV scraper daily at 6am
0 6 * * * /opt/streamtime/scraper --service youtube_tv --config /usr/local/etc/streamtime/config.yaml --database /usr/local/etc/streamtime/streamtime.db >> /var/log/streamtime.log 2>&1
```

## Cookie Maintenance

Cookies typically expire after 30-90 days. To maintain your scrapers:

**Refresh expired cookies:**
```bash
# Using Make (recommended)
make refresh-cookies SERVICE=netflix
make refresh-cookies SERVICE=youtube_tv
make refresh-cookies SERVICE=amazon_video

# Or directly
./backend/export-cookies --service netflix
./backend/export-cookies --service youtube_tv
./backend/export-cookies --service amazon_video
```

**Check if cookies are valid:**
```bash
./backend/export-cookies --service netflix --validate
./backend/export-cookies --service youtube_tv --validate
./backend/export-cookies --service amazon_video --validate
```

The tool will open a browser, you log in, and it automatically updates your config.

## API Endpoints

- `GET /api/services` - List all services with current month totals
- `GET /api/services/:id/history` - Get detailed watch history
- `POST /api/scrape/:service` - Manually trigger scraping
- `GET /api/health` - Health check

## Important Notes

⚠️ **For Personal Use Only**: This application uses web scraping which may violate streaming service Terms of Service. Use at your own risk.

⚠️ **Security**: Store credentials securely. The config file contains sensitive information.

⚠️ **2FA**: Services with two-factor authentication may require OAuth or manual session management.

## Development

See `IMPLEMENTATION_PLAN.md` for detailed development stages and technical decisions.

