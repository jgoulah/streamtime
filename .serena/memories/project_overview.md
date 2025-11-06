# StreamTime Project Overview

## Purpose
Personal streaming service watch time tracker. Monitors and displays screen time across Netflix, YouTube TV, Amazon Video, and other streaming platforms using web scraping.

**Key Features:**
- Dashboard view with monthly watch time totals per service
- Detailed analytics with historical trends, shows watched, episode breakdowns
- Automated daily scraping of viewing history
- Single user application, no authentication required

## Tech Stack

### Backend
- **Language**: Go 1.24
- **Framework**: gorilla/mux for HTTP routing
- **Database**: SQLite (mattn/go-sqlite3)
- **Scraping**: chromedp (headless Chrome automation)
- **Config**: gopkg.in/yaml.v3
- **CORS**: rs/cors

### Frontend
- **Framework**: React 19
- **Build Tool**: Vite 7
- **Styling**: Tailwind CSS 4
- **Routing**: react-router-dom 7
- **Charts**: recharts 3
- **Linting**: ESLint 9

### Infrastructure
- **Containerization**: Docker Compose
- **Web Server**: nginx (for production frontend)

## Project Status
Based on IMPLEMENTATION_PLAN.md:
- ✅ Stage 1: Project Setup & Infrastructure - Complete
- ✅ Stage 2: Backend Core & REST API - Complete
- ✅ Stage 3: Netflix Scraper Implementation - Complete
- ⏳ Stage 4: YouTube TV & Amazon Video Scrapers - Not Started (partial implementation exists)
- ✅ Stage 5: Frontend Implementation - Complete

## Priority Services
1. Netflix (implemented)
2. YouTube TV (partial implementation)
3. Amazon Video (partial implementation)
