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

2. Edit `config.yaml` with your streaming service credentials:
   ```yaml
   services:
     netflix:
       enabled: true
       email: your-email@example.com
       password: your-password
   ```

3. Configure scraping schedule (default: daily at 3 AM)

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

## License

MIT License - Personal use only
