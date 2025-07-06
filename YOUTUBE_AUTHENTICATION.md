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

### 1. Cookie Authentication (Recommended)

**Export cookies from your browser:**

#### Method A: Browser Extension
1. Install "Get cookies.txt LOCALLY" extension
2. Go to youtube.com and sign in
3. Click the extension icon
4. Save cookies as `youtube_cookies.txt`

#### Method B: Manual Export (Chrome)
1. Go to youtube.com in Chrome
2. Open DevTools (F12) → Application → Cookies
3. Copy all YouTube cookies to a text file in Netscape format

#### Method C: Using yt-dlp directly
```bash
# Extract cookies from Chrome
yt-dlp --cookies-from-browser chrome --write-info-json --skip-download "https://youtube.com/watch?v=dQw4w9WgXcQ"
```

**Configure ytclipper-go:**
```bash
export YTCLIPPER_YT_DLP_COOKIES_FILE="/path/to/youtube_cookies.txt"
```

### 2. Anti-Detection Configuration

The application now includes several anti-detection measures:

| Environment Variable | Default | Purpose |
|---------------------|---------|---------|
| `YTCLIPPER_YT_DLP_COOKIES_FILE` | "" | Path to browser cookies file |
| `YTCLIPPER_YT_DLP_USER_AGENT` | Chrome 120 | Browser user agent string |
| `YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES` | 3 | Number of retry attempts |
| `YTCLIPPER_YT_DLP_PROXY` | "" | Proxy server (if needed) |

### 3. Built-in Anti-Bot Measures

The application automatically adds:
- `--sleep-requests 1` - Sleep between requests
- `--sleep-interval 1` - Random sleep intervals
- `--max-sleep-interval 3` - Maximum sleep time
- Modern user agent headers
- Cookie-based session persistence

### 4. Automatic Fallback Strategies

The application implements intelligent fallback when encountering issues:

1. **First attempt**: Full anti-detection with proxy (if configured)
2. **Second attempt**: Anti-detection without proxy (if proxy fails)
3. **Third attempt**: Basic execution with minimal arguments

This ensures maximum compatibility across different network environments.

## 📝 Configuration Examples

### Local Development
```bash
# Set environment variables
export YTCLIPPER_YT_DLP_COOKIES_FILE="./cookies/youtube_cookies.txt"
export YTCLIPPER_YT_DLP_USER_AGENT="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
export YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES=5

# Run the application
go run main.go
```

### Docker Deployment
```yaml
version: '3.8'
services:
  ytclipper:
    build: .
    ports:
      - "8080:8080"
    environment:
      - YTCLIPPER_YT_DLP_COOKIES_FILE=/app/cookies/youtube_cookies.txt
      - YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES=5
    volumes:
      - ./cookies:/app/cookies:ro
      - ./videos:/app/videos
```

### GitHub Actions (CI/CD)
```yaml
- name: Setup YouTube Authentication
  run: |
    # Use a dummy cookies file or skip tests requiring authentication
    mkdir -p cookies
    echo "# Dummy cookies for CI" > cookies/youtube_cookies.txt
  env:
    YTCLIPPER_YT_DLP_COOKIES_FILE: cookies/youtube_cookies.txt
    YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES: 1
```

## 🔧 Troubleshooting

### Still Getting Bot Detection?

1. **Update cookies regularly** - YouTube cookies expire
2. **Use residential proxy** - Avoid datacenter IPs
3. **Reduce request rate** - Increase sleep intervals
4. **Update yt-dlp** - `pip install --upgrade yt-dlp`

### For GitHub Actions:

1. **Mock authentication** - Use dummy data for tests
2. **Skip real YouTube calls** - Test with local video files
3. **Use secrets** - Store real cookies in GitHub Secrets (not recommended for public repos)

### Proxy Configuration:
```bash
# HTTP proxy
export YTCLIPPER_YT_DLP_PROXY="http://proxy.example.com:8080"

# SOCKS5 proxy
export YTCLIPPER_YT_DLP_PROXY="socks5://proxy.example.com:1080"
```

## 🔒 Security Considerations

- **Never commit cookies** to version control
- **Use .gitignore** for cookies directory
- **Rotate cookies regularly** (monthly)
- **Use environment variables** for sensitive data
- **Consider cookie encryption** for production

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

### Test cookie authentication:
```bash
# Set environment variables
export YTCLIPPER_YT_DLP_COOKIES_FILE="./cookies/youtube_cookies.txt"

# Test with a known video
curl -X POST "http://localhost:8080/api/v1/clip" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://youtube.com/watch?v=dQw4w9WgXcQ","from":"0","to":"10","format":"best"}'
```

### Verify configuration:
```bash
# Check yt-dlp with your cookies
yt-dlp --cookies ./cookies/youtube_cookies.txt --get-title "https://youtube.com/watch?v=dQw4w9WgXcQ"
```

## 💡 Best Practices

1. **Use fresh cookies** from a logged-in YouTube session
2. **Monitor error logs** for authentication failures
3. **Implement fallback strategies** for bot detection
4. **Rate limit your own requests** to avoid triggering detection
5. **Keep yt-dlp updated** for latest anti-detection measures
6. **Test regularly** to ensure authentication remains valid

## 🆘 Emergency Fixes

If you're still getting blocked:

1. **Change IP address** (restart router or use VPN)
2. **Clear YouTube cookies** and re-login in browser
3. **Export fresh cookies** following the guide above
4. **Reduce concurrent requests** 
5. **Increase sleep intervals** to 5-10 seconds

This configuration should significantly reduce YouTube bot detection errors in both development and production environments.