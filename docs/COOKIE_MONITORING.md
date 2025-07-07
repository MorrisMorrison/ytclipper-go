# Cookie Monitoring and Expiration Notifications

## Overview

ytclipper-go now includes automatic cookie monitoring to track YouTube cookie expiration and send notifications when cookies need to be refreshed. This prevents service disruption by alerting you before cookies expire.

## Features

- **Dual Validation System**: Combines time-based expiration checking with functional API validation
- **API Validation**: Tests cookie functionality using actual YouTube API calls
- **Automatic Cookie Parsing**: Extracts expiration dates from Netscape format cookies
- **Smart Priority Notifications**: API validation failures suppress time-based warnings
- **Configurable Monitoring**: Set custom warning and urgent notification thresholds
- **Ntfy Integration**: Send notifications to your devices via ntfy.sh or self-hosted ntfy server
- **Scheduler Integration**: Uses existing scheduler infrastructure for periodic checks
- **Docker Compatible**: Works seamlessly in Docker containers
- **Early Detection**: Catch invalid cookies before their expiration date

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `YTCLIPPER_COOKIE_MONITOR_ENABLED` | `true` | Enable/disable cookie monitoring |
| `YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS` | `24` | How often to check cookie health (hours) |
| `YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS` | `30` | Days before expiration to send warning |
| `YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS` | `7` | Days before expiration to send urgent alert |
| `YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC` | `ytclipper-cookies` | Ntfy topic for notifications |
| `YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_ENABLED` | `true` | Enable API validation using actual YouTube calls |
| `YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_INTERVAL_HOURS` | `6` | How often to run API validation (hours) |
| `YTCLIPPER_COOKIE_MONITOR_TEST_VIDEO_URL` | `https://www.youtube.com/watch?v=dQw4w9WgXcQ` | YouTube video URL for API validation testing |
| `YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_TIMEOUT_SECS` | `30` | Timeout for API validation calls (seconds) |
| `YTCLIPPER_NTFY_ENABLED` | `false` | Enable ntfy notifications (global) |
| `YTCLIPPER_NTFY_SERVER_URL` | `""` | Ntfy server URL (e.g., https://ntfy.sh) |

### Example Configuration

```bash
# Enable cookie monitoring with ntfy notifications
export YTCLIPPER_COOKIE_MONITOR_ENABLED=true
export YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS=12
export YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS=14
export YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS=3

# Configure API validation (new feature)
export YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_ENABLED=true
export YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_INTERVAL_HOURS=6
export YTCLIPPER_COOKIE_MONITOR_TEST_VIDEO_URL="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
export YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_TIMEOUT_SECS=30

# Configure ntfy notifications
export YTCLIPPER_NTFY_ENABLED=true
export YTCLIPPER_NTFY_SERVER_URL="https://ntfy.sh"
export YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC="my-ytclipper-alerts"

# Set your YouTube cookies
export YTCLIPPER_YT_DLP_COOKIES_CONTENT=".youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123"
```

## Notification Types

### 🟡 Warning Notification
- **Trigger**: Cookie expires within warning threshold (default: 30 days)
- **Priority**: Normal
- **Action**: Plan cookie refresh

### 🔴 Urgent Notification
- **Trigger**: Cookie expires within urgent threshold (default: 7 days)
- **Priority**: High
- **Action**: Refresh cookies immediately

### ❌ Expired Notification
- **Trigger**: Cookie has expired
- **Priority**: Maximum
- **Action**: Service using fallback strategy, update cookies

### 🔴 API Validation Failed Notification
- **Trigger**: Cookie fails functional validation with YouTube API
- **Priority**: Critical
- **Action**: Update cookies immediately - they may be invalid or rate-limited

### ✅ Test Notification
- **Trigger**: Application startup (if ntfy enabled)
- **Priority**: Low
- **Action**: Verify notification configuration

## Cookie Parsing

### Supported Format
The monitor parses Netscape format cookies:
```
.youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123
```

### Target Cookie
Specifically monitors `VISITOR_INFO1_LIVE` cookie which:
- Expires every 6 months
- Is critical for YouTube authentication
- Indicates overall cookie health

### Fallback Behavior
If `VISITOR_INFO1_LIVE` not found, estimates 6 months from current time.

## API Validation System

### How It Works
- **Functional Testing**: Uses `GetVideoDuration` to test cookie functionality with actual YouTube API calls
- **Test Video**: Configurable YouTube video URL for validation (default: Rick Astley - Never Gonna Give You Up)
- **Timeout Protection**: Configurable timeout to prevent hanging (default: 30 seconds)
- **Error Recovery**: Handles network failures, rate limiting, and other API errors gracefully

### Validation Logic
1. **Time-based Check**: Verifies cookie expiration date
2. **API Validation**: Tests cookie functionality with YouTube API
3. **Priority Handling**: API validation failures suppress time-based notifications
4. **Smart Notifications**: Different alerts for time-based vs functional failures

### Benefits
- **Early Detection**: Catch invalid cookies before expiration date
- **Rate Limiting Detection**: Identify when cookies are being throttled
- **Real-world Validation**: Test actual functionality, not just dates
- **Proactive Monitoring**: Prevent service disruption from invalid cookies

## Docker Configuration

### docker-compose.yml
```yaml
version: '3.8'
services:
  ytclipper:
    build: .
    environment:
      # Cookie monitoring
      - YTCLIPPER_COOKIE_MONITOR_ENABLED=true
      - YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS=24
      - YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS=30
      - YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS=7
      
      # API validation (new feature)
      - YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_ENABLED=true
      - YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_INTERVAL_HOURS=6
      - YTCLIPPER_COOKIE_MONITOR_TEST_VIDEO_URL=https://www.youtube.com/watch?v=dQw4w9WgXcQ
      - YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_TIMEOUT_SECS=30
      
      # Ntfy notifications
      - YTCLIPPER_NTFY_ENABLED=true
      - YTCLIPPER_NTFY_SERVER_URL=https://ntfy.sh
      - YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC=ytclipper-cookies
      
      # YouTube cookies
      - YTCLIPPER_YT_DLP_COOKIES_CONTENT=${YOUTUBE_COOKIES}
    volumes:
      - ./videos:/app/videos
    ports:
      - "8080:8080"
```

### .env file
```bash
YOUTUBE_COOKIES=".youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123"
```

## Ntfy Setup

### Using ntfy.sh (Public)
1. Go to [ntfy.sh](https://ntfy.sh)
2. Choose a unique topic name (e.g., `your-name-ytclipper-2024`)
3. Subscribe to the topic on your devices
4. Configure environment variables:
   ```bash
   export YTCLIPPER_NTFY_ENABLED=true
   export YTCLIPPER_NTFY_SERVER_URL="https://ntfy.sh"
   export YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC="your-name-ytclipper-2024"
   ```

### Self-Hosted Ntfy
1. Install ntfy on your server: `docker run -p 80:80 binwiederhier/ntfy serve`
2. Configure environment variables:
   ```bash
   export YTCLIPPER_NTFY_ENABLED=true
   export YTCLIPPER_NTFY_SERVER_URL="https://your-ntfy-server.com"
   export YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC="ytclipper-cookies"
   ```

## Monitoring Workflow

### Initial Setup
1. Configure environment variables
2. Start ytclipper-go
3. Monitor logs for:
   ```
   Starting cookie monitor: Interval 24.000000 hours
   Sending test notification...
   Successfully sent ntfy notification to topic 'ytclipper-cookies': 🧪 Cookie Monitoring Test
   ```

### Ongoing Monitoring
1. **Daily Health Checks**: Monitor runs every 24 hours (configurable)
2. **API Validation**: Functional tests run every 6 hours (configurable)
3. **Warning Phase**: 30 days before expiration (time-based)
4. **Urgent Phase**: 7 days before expiration (time-based)
5. **API Failure Phase**: Immediate notification if cookies fail functional tests
6. **Expired Phase**: Cookie has expired, fallback strategy active

### Cookie Refresh Process
1. **Receive notification** that cookies will expire soon
2. **Export fresh cookies** from your browser:
   - Use browser extension (Get cookies.txt LOCALLY)
   - Or manually via DevTools
3. **Update environment variable**:
   ```bash
   export YTCLIPPER_YT_DLP_COOKIES_CONTENT="new_cookie_content_here"
   ```
4. **Restart container** to load new cookies
5. **Verify** in logs that new expiration date is detected

## Troubleshooting

### No Notifications Received
1. **Check ntfy configuration**:
   ```bash
   curl -d "Test message" https://ntfy.sh/your-topic
   ```
2. **Verify environment variables** are set correctly
3. **Check logs** for error messages
4. **Ensure ntfy is enabled**: `YTCLIPPER_NTFY_ENABLED=true`

### Cookie Parsing Issues
1. **Check cookie format** - must be Netscape format
2. **Verify VISITOR_INFO1_LIVE** cookie is present
3. **Check logs** for parsing errors
4. **Test with minimal cookie content**

### Monitoring Not Running
1. **Verify monitoring is enabled**: `YTCLIPPER_COOKIE_MONITOR_ENABLED=true`
2. **Check scheduler startup** in logs
3. **Ensure no configuration errors** on startup

## Logs Examples

### Successful Monitoring
```
Starting cookie monitor: Interval 24.000000 hours
Sending test notification...
Successfully sent ntfy notification: 🧪 ytclipper-go Test
Cookie VISITOR_INFO1_LIVE expires at 2024-12-01T00:00:00Z (in 8760h0m0s)
Performing API validation for cookie health...
Validating cookie with API using test video: https://www.youtube.com/watch?v=dQw4w9WgXcQ
Cookie validation successful - retrieved duration: 212
API validation successful - cookies are working properly
Cookie is healthy (expires in 8760h)
```

### Warning Notification
```
Running periodic cookie health check...
Cookie VISITOR_INFO1_LIVE expires at 2024-12-01T00:00:00Z (in 720h0m0s)
Successfully sent ntfy notification: 🟡 YouTube Cookie Warning
```

### Urgent Notification
```
Cookie VISITOR_INFO1_LIVE expires at 2024-12-01T00:00:00Z (in 120h0m0s)
Successfully sent ntfy notification: 🔴 URGENT: YouTube Cookie Expiring
```

### API Validation Failure
```
Cookie VISITOR_INFO1_LIVE expires at 2024-12-01T00:00:00Z (in 8760h0m0s)
Performing API validation for cookie health...
Validating cookie with API using test video: https://www.youtube.com/watch?v=dQw4w9WgXcQ
API validation failed: failed to get video duration: HTTP 403: Forbidden
Successfully sent ntfy notification: 🔴 Cookie API Validation Failed
Skipping time-based notifications due to API validation failure
```

### API Validation Disabled
```
Cookie VISITOR_INFO1_LIVE expires at 2024-12-01T00:00:00Z (in 720h0m0s)
API validation is disabled - skipping functional tests
Successfully sent ntfy notification: 🟡 YouTube Cookie Warning
```

## Migration from Previous Versions

### Automatic Migration
- **Existing configurations**: All existing environment variables continue to work
- **Default behavior**: API validation is enabled by default for enhanced monitoring
- **Backward compatibility**: Time-based monitoring continues unchanged

### New vs Old Behavior
- **Old**: Only time-based expiration checking
- **New**: Dual validation (time-based + API functional testing)
- **Smart notifications**: API failures take priority over time-based warnings
- **Enhanced detection**: Catch rate-limited or invalid cookies early

### Opting Out of API Validation
If you prefer the old behavior:
```bash
export YTCLIPPER_COOKIE_MONITOR_API_VALIDATION_ENABLED=false
```

This monitoring system ensures your ytclipper-go instance maintains optimal performance by proactively managing YouTube cookie lifecycle with both expiration tracking and functional validation.