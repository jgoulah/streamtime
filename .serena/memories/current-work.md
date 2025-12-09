# StreamTime - Current Work Status

**Last Updated**: December 8, 2025

## Current Branch
- `main` (working directly on main)

## Recently Completed Work

### 1. Amazon Video Scraper (COMPLETED ✅)
- **Issue**: chromedp blocking when querying DOM nodes
- **Solution**: Rewrote to use pure JavaScript evaluation instead of chromedp selectors
- **Status**: Fully functional, extracts ~7 items from watch history
- **Key Learning**: Amazon page structure has date sections and show lists as siblings, not nested
- **Performance**: ~20 seconds to complete

### 2. Hulu Scraper (COMPLETED ✅)
- **Challenge**: Hulu doesn't provide full watch history with dates
- **Solution**: 
  - Scrapes "Continue Watching" collection only
  - Assigns yesterday's date to new items
  - Smart duplicate detection against existing DB
- **Status**: Fully functional, extracted 11 items on first run
- **Selector**: `div[data-testid="standard-emphasis-tile-subtitle"]`
- **Future**: With daily scraping, will approximate watch dates

### 3. Database Schema Fix (COMPLETED ✅)
- **Issue**: `logo_url` column is nullable but Go code scanning into string
- **Fix**: Updated all queries to use `sql.NullString` for nullable fields
- **Files Modified**: 
  - `backend/internal/database/queries.go` (3 functions)
  - GetAllServices, GetServiceByID, GetServiceByName, GetServiceStats

### 4. Cookie Export Tool Updates (COMPLETED ✅)
- Added Hulu support
- Enhanced instructions for services where login URL ≠ test URL
- Hulu workflow: login → manually navigate to Continue Watching → press Enter

## Active Services Status

| Service | Scraper Status | Cookie Status | Notes |
|---------|---------------|---------------|-------|
| Netflix | ✅ Working | Valid | Extracts from viewing activity |
| YouTube TV | ✅ Working | Valid | Scrapes Google My Activity |
| Amazon Video | ✅ Working | Valid | Uses JavaScript evaluation |
| Hulu | ✅ Working | Valid | Continue Watching only |

## Known Issues

### None Currently
All scrapers are working as expected.

## Next Steps / Future Work

### High Priority
1. **Production Deployment**
   - Deploy latest changes to mediaserver
   - Enable Hulu in production config
   - Set up cron jobs for daily scraping

2. **Cron Schedule**
   ```bash
   # Run all scrapers daily at 6am
   0 6 * * * /opt/streamtime/scraper --config /usr/local/etc/streamtime/config.yaml
   ```

### Medium Priority
1. **Frontend Enhancements**
   - Display Hulu data in UI
   - Add service logos
   - Improve date range filtering

2. **Scraper Improvements**
   - Add retry logic for failed scrapes
   - Better error reporting
   - Email notifications on scraper failures

3. **Data Quality**
   - Handle duplicate episodes better
   - Extract more metadata (thumbnails, genres)
   - Improve date parsing across services

### Low Priority
1. **Additional Services**
   - Disney+
   - HBO Max
   - Paramount+
   - Apple TV+

2. **Features**
   - Export to CSV
   - Statistics/analytics dashboard
   - Search/filter capabilities

## Technical Debt

1. **Test Coverage**: No unit tests exist for scrapers
2. **Error Handling**: Some scrapers silently skip errors
3. **Configuration**: Hardcoded values in some scrapers
4. **Documentation**: API endpoints not documented

## Development Environment

- **OS**: macOS (Darwin 24.6.0)
- **Go Version**: 1.21+
- **Node Version**: Latest (Vite compatible)
- **Database**: SQLite 3.x

## Recent Learnings

1. **chromedp Pitfall**: `chromedp.Nodes()` hangs when called from node contexts
   - **Solution**: Use JavaScript evaluation for all DOM queries
   
2. **Amazon Cookie Issue**: Must wait for watch history page to fully load before capturing cookies
   - Timing is critical - page must show actual viewing history

3. **Hulu Limitation**: No full watch history API available
   - Best we can do is "Continue Watching" collection
   - Daily scraping provides approximate watch dates

4. **Database Nullability**: Go's database/sql requires special handling for NULL columns
   - Use `sql.NullString`, `sql.NullInt64`, etc. for nullable columns
