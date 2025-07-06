# Cookie Monitoring and Expiration Notifications

## Overview

ytclipper-go now includes automatic cookie monitoring to track YouTube cookie expiration and send notifications when cookies need to be refreshed. This prevents service disruption by alerting you before cookies expire.

## Features

- **Automatic Cookie Parsing**: Extracts expiration dates from Netscape format cookies
- **Configurable Monitoring**: Set custom warning and urgent notification thresholds
- **Ntfy Integration**: Send notifications to your devices via ntfy.sh or self-hosted ntfy server
- **Scheduler Integration**: Uses existing scheduler infrastructure for periodic checks
- **Docker Compatible**: Works seamlessly in Docker containers

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `YTCLIPPER_COOKIE_MONITOR_ENABLED` | `true` | Enable/disable cookie monitoring |
| `YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS` | `24` | How often to check cookie health (hours) |
| `YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS` | `30` | Days before expiration to send warning |
| `YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS` | `7` | Days before expiration to send urgent alert |
| `YTCLIPPER_COOKIE_MONITOR_NTFY_ENABLED` | `false` | Enable ntfy notifications |
| `YTCLIPPER_COOKIE_MONITOR_NTFY_SERVER_URL` | `""` | Ntfy server URL (e.g., https://ntfy.sh) |
| `YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC` | `ytclipper-cookies` | Ntfy topic for notifications |

### Example Configuration

```bash
# Enable cookie monitoring with ntfy notifications
export YTCLIPPER_COOKIE_MONITOR_ENABLED=true
export YTCLIPPER_COOKIE_MONITOR_INTERVAL_HOURS=12
export YTCLIPPER_COOKIE_MONITOR_WARNING_THRESHOLD_DAYS=14
export YTCLIPPER_COOKIE_MONITOR_URGENT_THRESHOLD_DAYS=3

# Configure ntfy notifications
export YTCLIPPER_COOKIE_MONITOR_NTFY_ENABLED=true
export YTCLIPPER_COOKIE_MONITOR_NTFY_SERVER_URL="https://ntfy.sh"
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
      
      # Ntfy notifications
      - YTCLIPPER_COOKIE_MONITOR_NTFY_ENABLED=true
      - YTCLIPPER_COOKIE_MONITOR_NTFY_SERVER_URL=https://ntfy.sh
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
   export YTCLIPPER_COOKIE_MONITOR_NTFY_SERVER_URL="https://ntfy.sh"
   export YTCLIPPER_COOKIE_MONITOR_NTFY_TOPIC="your-name-ytclipper-2024"
   ```

### Self-Hosted Ntfy
1. Install ntfy on your server: `docker run -p 80:80 binwiederhier/ntfy serve`
2. Configure environment variables:
   ```bash
   export YTCLIPPER_COOKIE_MONITOR_NTFY_SERVER_URL="https://your-ntfy-server.com"
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
   Successfully sent ntfy notification: 🧪 ytclipper-go Test
   ```

### Ongoing Monitoring
1. **Daily Health Checks**: Monitor runs every 24 hours (configurable)
2. **Warning Phase**: 30 days before expiration
3. **Urgent Phase**: 7 days before expiration  
4. **Expired Phase**: Cookie has expired, fallback strategy active

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
4. **Ensure ntfy is enabled**: `YTCLIPPER_COOKIE_MONITOR_NTFY_ENABLED=true`

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

This monitoring system ensures your ytclipper-go instance maintains optimal performance by proactively managing YouTube cookie lifecycle.