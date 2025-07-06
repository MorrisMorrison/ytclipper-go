# HTTP API Testing

This directory contains HTTP request files for testing the ytclipper-go API using REST clients like IntelliJ IDEA, VS Code with REST Client extension, or other HTTP clients that support `.http` files.

## Setup

### 1. Configure Environment

Edit `http-client.env.json` to set your environment variables:

```json
{
  "dev": {
    "baseUrl": "http://localhost:8080",
    "basicAuth": {
      "username": "your-username",
      "password": "your-password"
    }
  }
}
```

### 2. Start the Server

```bash
# Run locally
go run main.go

# Or with Docker
docker-compose up
```

### 3. Set Authentication (Optional)

If Basic Auth is enabled, set your credentials in the environment file or as environment variables:

```bash
export YTCLIPPER_AUTH_USERNAME="your-username"
export YTCLIPPER_AUTH_PASSWORD="your-password"
```

## File Structure

### Core API Testing
- **`health.http`** - Health check and homepage endpoints
- **`video-info.http`** - Video duration and format information
- **`clips.http`** - Clip creation with various parameters
- **`jobs.http`** - Job status checking and clip downloads

### Advanced Testing
- **`workflow.http`** - Complete end-to-end workflow with automated steps
- **`auth-and-limits.http`** - Authentication and rate limiting tests
- **`edge-cases.http`** - Error handling and edge case scenarios

### Configuration
- **`http-client.env.json`** - Environment variables for different deployments

## Quick Start

### 1. Test Server Health
Run the health check first to ensure the server is running:
```http
GET {{baseUrl}}/health
```

### 2. Get Video Information
Get duration and available formats:
```http
GET {{baseUrl}}/api/v1/video/duration?youtubeUrl=https://www.youtube.com/watch?v=dQw4w9WgXcQ
GET {{baseUrl}}/api/v1/video/formats?youtubeUrl=https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

### 3. Create and Download Clip
```http
# Create clip
POST {{baseUrl}}/api/v1/clip
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "from": "00:00:10",
  "to": "00:00:20",
  "format": "136"
}

# Check job status (use job ID from creation response)
GET {{baseUrl}}/api/v1/jobs/status?jobId=your-job-id

# Download completed clip
GET {{baseUrl}}/api/v1/clip?jobId=your-job-id
```

## API Endpoints Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check (no auth) |
| `GET` | `/` | Homepage HTML |
| `GET` | `/api/v1/video/duration` | Get video duration |
| `GET` | `/api/v1/video/formats` | Get available formats |
| `POST` | `/api/v1/clip` | Create clip job |
| `GET` | `/api/v1/jobs/status` | Check job status |
| `GET` | `/api/v1/clip` | Download completed clip |

## Common Parameters

### YouTube URLs
Supported formats:
- `https://www.youtube.com/watch?v=VIDEO_ID`
- `https://youtu.be/VIDEO_ID`
- URLs with additional parameters (playlists, timestamps, etc.)

### Time Formats
- **HH:MM:SS**: `"00:01:30"` (1 minute 30 seconds)
- **MM:SS**: `"01:30"` (1 minute 30 seconds)  
- **Seconds**: `"90"` (90 seconds)

### Format IDs
Common format IDs from yt-dlp:
- **136**: 720p MP4 (video only)
- **137**: 1080p MP4 (video only)
- **298**: 720p MP4 (video + audio)
- **299**: 1080p MP4 (video + audio)

Get available formats using the `/api/v1/video/formats` endpoint.

## Response Codes

| Code | Meaning | Example |
|------|---------|---------|
| 200 | Success | Video info retrieved, clip downloaded |
| 201 | Created/Processing | Job created or still processing |
| 400 | Bad Request | Invalid URL, missing parameters |
| 401 | Unauthorized | Invalid authentication |
| 404 | Not Found | Job doesn't exist |
| 429 | Rate Limited | Too many requests |
| 500 | Server Error | Processing failure |

## Error Responses

All error responses return JSON:
```json
{
  "error": "Description of the error"
}
```

## Testing Workflows

### 1. Basic Functionality Test
1. Run `health.http` - Test server availability
2. Run `video-info.http` - Test video information retrieval
3. Run `clips.http` - Test clip creation
4. Run `jobs.http` - Test job management

### 2. End-to-End Test
Use `workflow.http` for automated end-to-end testing with JavaScript post-processing.

### 3. Stress Testing
Use `auth-and-limits.http` to test rate limiting and authentication.

### 4. Error Handling
Use `edge-cases.http` to test error scenarios and edge cases.

## IDE Integration

### IntelliJ IDEA / WebStorm
- Built-in support for `.http` files
- Environment variables automatically loaded from `http-client.env.json`
- Response history and testing features included

### VS Code
Install the "REST Client" extension:
```bash
code --install-extension humao.rest-client
```

### Other Clients
- **Postman**: Import requests manually
- **curl**: Convert requests to curl commands
- **HTTPie**: Use for command-line testing

## Environment Variables

Set these in your `.env` or environment:

```bash
# Server Configuration
YTCLIPPER_PORT=8080
YTCLIPPER_DEBUG=true

# Authentication (optional)
YTCLIPPER_AUTH_USERNAME=""
YTCLIPPER_AUTH_PASSWORD=""

# Rate Limiting
YTCLIPPER_RATE_LIMITER_RATE=5
YTCLIPPER_RATE_LIMITER_BURST=20

# Video Processing
YTCLIPPER_YT_DLP_CLIP_SIZE_LIMIT_IN_MB=300
YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS=60
```

## Troubleshooting

### Connection Refused
- Ensure server is running on correct port
- Check firewall and network settings
- Verify `baseUrl` in environment configuration

### Authentication Errors
- Verify username/password in `http-client.env.json`
- Check if Basic Auth is enabled on server
- Test with `/health` endpoint (no auth required)

### Rate Limiting
- Wait between requests or increase rate limits
- Check rate limiter configuration
- Use different IP or restart client

### Video Processing Errors
- Verify YouTube URL is accessible
- Check if video is available in your region
- Ensure video is not private or deleted
- Try different format IDs

## Examples

See individual `.http` files for comprehensive examples of:
- Error handling scenarios
- Different parameter combinations
- Authentication testing
- Rate limiting verification
- Complete workflow automation

For more information about the ytclipper-go API, see the main project documentation.