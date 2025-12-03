# Export Cookies Tool

A semi-automated tool for exporting and managing authentication cookies for StreamTime scrapers.

## Features

✅ **Multi-Service Support** - Works with Netflix, YouTube TV, and Amazon Video
✅ **Auto-Update Config** - Automatically updates `config.yaml` with new cookies
✅ **Cookie Validation** - Tests cookies to ensure they work before saving
✅ **Interactive Login** - Opens a real browser for you to log in (handles 2FA, CAPTCHA)
✅ **Progress Indicators** - Clear feedback at every step
✅ **Validation Mode** - Check if existing cookies are still valid

## Usage

### Export New Cookies

```bash
# From the project root
./backend/export-cookies --service <service_name>

# Examples:
./backend/export-cookies --service netflix
./backend/export-cookies --service youtube_tv
./backend/export-cookies --service amazon_video
```

**What happens:**
1. Opens a browser window to the login page
2. You log in manually (handles 2FA, CAPTCHA, etc.)
3. Press Enter when logged in
4. Tool extracts all cookies automatically
5. Validates that cookies work
6. Updates `config.yaml` automatically
7. Done!

### Validate Existing Cookies

Check if your current cookies are still valid without exporting new ones:

```bash
./backend/export-cookies --service netflix --validate
./backend/export-cookies --service youtube_tv --validate
./backend/export-cookies --service amazon_video --validate
```

This is useful for:
- Checking if cookies have expired
- Troubleshooting scraper authentication issues
- Verifying setup before running scrapers

### Custom Config Path

If your `config.yaml` is in a different location:

```bash
./backend/export-cookies --service netflix --config /path/to/config.yaml
```

## When to Use This Tool

Run this tool when:

- **Initial Setup**: First time setting up a service
- **Cookie Expiration**: Scrapers report authentication failures
- **After Password Change**: Changed your streaming service password
- **Periodic Maintenance**: Cookies typically expire after 30-90 days

## How It Works

### Export Process

1. **Opens Browser**: Launches a real Chrome window (not headless)
2. **You Log In**: You manually complete the login process
3. **Extracts Cookies**: Grabs all authentication cookies from the browser
4. **Validates**: Tests cookies in a headless browser to ensure they work
5. **Updates Config**: Automatically writes cookies to `config.yaml`
6. **Enables Service**: Sets `enabled: true` for the service

### Validation Process

1. **Loads Config**: Reads existing cookies from `config.yaml`
2. **Tests Cookies**: Opens a headless browser with those cookies
3. **Checks Access**: Navigates to service-specific pages
4. **Reports Status**: Tells you if cookies are valid or expired

## Service-Specific Details

### Netflix

- **Required Cookies**: `NetflixId`, `SecureNetflixId`
- **Test URL**: https://www.netflix.com/viewingactivity
- **Cookie Domain**: `.netflix.com`

### YouTube TV

- **Required Cookies**: `SID`, `HSID`, `SSID`, `APISID`, `SAPISID`
- **Test URL**: https://myactivity.google.com/product/youtube
- **Cookie Domain**: `.google.com`
- **Note**: Exports many Google cookies for full compatibility

### Amazon Video

- **Required Cookies**: `session-id`, `ubid-main`
- **Test URL**: https://www.amazon.com/gp/video/settings/watch-history
- **Cookie Domain**: `.amazon.com`

## Troubleshooting

### "No cookies found for domain"

**Problem**: Tool couldn't find any cookies after login
**Solution**: Make sure you fully completed the login process before pressing Enter

### "Missing some expected cookies"

**Problem**: Some important cookies weren't found
**Solution**: The tool still saves what it found, but the scraper might not work. Try logging in again.

### "Cookie validation failed"

**Problem**: Exported cookies don't seem to work
**Solution**:
- Try logging in again
- Clear your browser cookies and start fresh
- Make sure you're logging into the correct account

### "Cookies appear invalid - redirected to login page"

**Problem**: Validation detected that cookies aren't working
**Solution**: Re-export cookies by running the tool again without `--validate`

## Building from Source

If you've modified the code:

```bash
cd backend
go build -o export-cookies ./cmd/export-cookies/
```

## Examples

### Complete Workflow

```bash
# 1. Check if current cookies are valid
./backend/export-cookies --service youtube_tv --validate

# Output: ❌ Validation failed: ...

# 2. Export fresh cookies
./backend/export-cookies --service youtube_tv

# Browser opens, you log in, press Enter...
# Output: ✅ Cookies validated successfully!
# Output: ✅ Configuration updated successfully!

# 3. Verify it worked
./backend/export-cookies --service youtube_tv --validate

# Output: ✅ Cookies are valid!

# 4. Run the scraper
curl -X POST http://localhost:8080/api/scrape/youtube_tv
```

### Quick Check All Services

```bash
./backend/export-cookies --service netflix --validate
./backend/export-cookies --service youtube_tv --validate
./backend/export-cookies --service amazon_video --validate
```

## Security Notes

- Cookies are stored in plaintext in `config.yaml` - keep this file secure
- Don't commit `config.yaml` to git (it's in `.gitignore`)
- Cookies are authentication tokens - treat them like passwords
- Cookies typically expire after 30-90 days for security
- Use this tool on a secure, personal machine only

## Tips

- **Keep cookies fresh**: Run validation monthly
- **Batch export**: Export all services in one session
- **Before automated runs**: Validate cookies before enabling schedulers
- **After changes**: Re-export if you change passwords or log out
- **Share with care**: Never share your `config.yaml` or cookies
