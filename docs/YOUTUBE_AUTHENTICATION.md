# YouTube Authentication & Bot Detection Bypass

This guide explains how to configure ytclipper-go to bypass YouTube's bot detection and authentication requirements.

## 🚨 The Problem

YouTube frequently shows this error:
```
ERROR: [youtube] dQw4w9WgXcQ: Sign in to confirm you're not a bot. 
Use --cookies-from-browser or --cookies for the authentication.
```

This happens because:
- YouTube detects automated requests
- Rate limiting triggers bot detection
- Missing browser-like headers and cookies
- Datacenter IPs are flagged

## 🛠️ Solutions Implemented

### 1. Cookie Authentication (Optional)

**Export cookies from your browser if needed:**

#### Method A: Browser Extension
1. Install "Get cookies.txt LOCALLY" extension
2. Go to youtube.com and sign in
3. Click the extension icon
4. Save cookies as `youtube_cookies.txt`

#### Method B: Manual Export (Chrome)
1. Go to youtube.com in Chrome
2. Open DevTools (F12) → Application → Cookies
3. Copy all YouTube cookies to a text file in Netscape format

**Configure ytclipper-go (only if needed):**
```bash
export YTCLIPPER_YT_DLP_COOKIES_FILE="/path/to/youtube_cookies.txt"
```

**Note:** The application now uses a simplified cookie-based authentication approach. Cookie authentication is the primary strategy, with anti-detection headers as a fallback when cookies are not available.

### 2. Built-in Anti-Detection Strategy (Fallback)

The application uses a simplified 2-tier fallback strategy:

#### Tier 1: Cookie-Based Authentication (Primary)
- Uses cookie authentication when configured via environment variables
- Supports both cookie content and cookie files
- Includes basic anti-detection headers
- Highest success rate when cookies are available

#### Tier 2: Anti-Detection Headers (Fallback)
- User agent rotation (6 modern browser user agents)
- Enhanced HTTP headers for authenticity
- No authentication required
- Automatic retry on failure

### 3. Optional Environment Configuration

| Environment Variable | Default | Purpose |
|---------------------|---------|---------|
| `YTCLIPPER_YT_DLP_COOKIES_CONTENT` | "" | Cookie content as string (tier 1 primary) |
| `YTCLIPPER_YT_DLP_COOKIES_FILE` | "" | Path to browser cookies file (tier 1 alternative) |
| `YTCLIPPER_YT_DLP_PROXY` | "" | Proxy server (tier 1 optional) |

### 4. Built-in Anti-Bot Measures

The application automatically implements:

#### Primary Strategy (Cookie-Based Authentication)
- Uses cookie authentication when configured via environment variables
- Supports both cookie content and cookie files
- Includes basic anti-detection headers
- Highest success rate when cookies are available
- Optional proxy support

#### Fallback Strategy (Enhanced Anti-Detection)
- User agent rotation (6 modern browser user agents)
- Enhanced HTTP headers for authenticity
- No authentication required
- Automatic retry on failure

### 5. No Configuration Required

The primary approach works automatically without any configuration:
- **Cookie-free by default**
- **No proxy required**
- **No user intervention needed**
- **Comprehensive fallback system**

## 📝 Configuration Examples

### Local Development (No Configuration Needed)
```bash
# Run the application - uses built-in 2-tier fallback strategy
go run main.go
```

### Local Development (With Optional Configuration)
```bash
# Optional: Set environment variables only if needed
export YTCLIPPER_YT_DLP_COOKIES_FILE="./cookies/youtube_cookies.txt"
export YTCLIPPER_YT_DLP_PROXY="http://proxy.example.com:8080"

# Run the application
go run main.go
```

### Docker Deployment (No Configuration Needed)
```yaml
version: '3.8'
services:
  ytclipper:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./videos:/app/videos
```

### Docker Deployment (With Optional Configuration)
```yaml
version: '3.8'
services:
  ytclipper:
    build: .
    ports:
      - "8080:8080"
    environment:
      - YTCLIPPER_YT_DLP_COOKIES_FILE=/app/cookies/youtube_cookies.txt
      - YTCLIPPER_YT_DLP_PROXY=http://proxy.example.com:8080
    volumes:
      - ./cookies:/app/cookies:ro
      - ./videos:/app/videos
```

### GitHub Actions (CI/CD)
```yaml
- name: Run ytclipper tests
  run: |
    # No configuration needed - uses built-in anti-detection strategies
    go test ./...
```

## 🔧 Troubleshooting

### Still Getting Bot Detection?

The application automatically tries 2 progressive strategies, but if all fail:

1. **Check yt-dlp version** - `pip install --upgrade yt-dlp`
2. **Use proxy configuration** - Set `YTCLIPPER_YT_DLP_PROXY` environment variable
3. **Export fresh cookies** - Set `YTCLIPPER_YT_DLP_COOKIES_FILE` environment variable
4. **Monitor logs** - Check which tier is failing

### For GitHub Actions:

1. **No configuration needed** - Built-in strategies work in CI/CD
2. **Test with mock data** - Use local video files for tests
3. **Add proxy if needed** - Use GitHub Secrets for proxy configuration

### Proxy Configuration:
```bash
# HTTP proxy
export YTCLIPPER_YT_DLP_PROXY="http://proxy.example.com:8080"

# SOCKS5 proxy
export YTCLIPPER_YT_DLP_PROXY="socks5://proxy.example.com:1080"
```

## 🔒 Security Considerations

- **No cookies by default** - Primary approach is cookie-free
- **Never commit cookies** to version control (if using optional cookie configuration)
- **Use .gitignore** for cookies directory (if using optional cookie configuration)
- **Rotate cookies regularly** (if using optional cookie configuration)
- **Use environment variables** for sensitive data
- **Proxy security** - Use trusted proxy services (if using optional proxy configuration)

## 📁 Recommended Directory Structure

```
ytclipper-go/
├── cookies/
│   ├── .gitignore
│   └── youtube_cookies.txt
├── videos/
├── docker-compose.yml
└── ...
```

**.gitignore entry:**
```
cookies/*.txt
cookies/*.json
!cookies/.gitignore
```

## 🧪 Testing

### Test built-in anti-detection:
```bash
# Test with a known video - no configuration needed
curl -X POST "http://localhost:8080/api/v1/clip" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://youtube.com/watch?v=dQw4w9WgXcQ","from":"0","to":"10","format":"best"}'
```

### Verify fallback configuration (if needed):
```bash
# Check yt-dlp with optional cookies
yt-dlp --cookies ./cookies/youtube_cookies.txt --get-title "https://youtube.com/watch?v=dQw4w9WgXcQ"
```

## 💡 Best Practices

1. **Use default configuration** - Built-in strategies work for most cases
2. **Monitor error logs** for failures across all tiers
3. **Keep yt-dlp updated** for latest anti-detection measures
4. **Use proxy configuration** only if needed for specific use cases
5. **Use fresh cookies** only if configuring the optional cookie fallback
6. **Test regularly** to ensure continued effectiveness

## 🆘 Emergency Fixes

If you're still getting blocked after both tiers fail:

1. **Update yt-dlp** - `pip install --upgrade yt-dlp`
2. **Configure proxy** - Set `YTCLIPPER_YT_DLP_PROXY` environment variable
3. **Export fresh cookies** - Set `YTCLIPPER_YT_DLP_COOKIES_FILE` environment variable
4. **Change IP address** (restart router or use VPN)
5. **Check logs** to see which tier is consistently failing

This 2-tier fallback system should significantly reduce YouTube bot detection errors in both development and production environments without requiring any configuration for most use cases.