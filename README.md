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
   # Netflix
   ./backend/export-cookies --service netflix

   # YouTube TV
   ./backend/export-cookies --service youtube_tv

   # Amazon Video
   ./backend/export-cookies --service amazon_video
   ```

   The tool will:
   - Open a browser window for you to log in
   - Automatically extract and validate cookies
   - Update `config.yaml` with the correct configuration

   See [Cookie Export Tool Documentation](backend/cmd/export-cookies/README.md) for details.

3. (Optional) Configure scraping schedule (default: daily at 3 AM)

### Running with Docker

```bash
docker-compose up -d
```

Access the app at `http://localhost:3000`

### Running Locally

**Backend:**
```bash
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

## Cookie Maintenance

Cookies typically expire after 30-90 days. To maintain your scrapers:

**Check if cookies are valid:**
```bash
./backend/export-cookies --service netflix --validate
./backend/export-cookies --service youtube_tv --validate
./backend/export-cookies --service amazon_video --validate
```

**Refresh expired cookies:**
```bash
./backend/export-cookies --service youtube_tv
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

