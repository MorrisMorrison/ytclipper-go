# ytclipper-go
[![main](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml)

A high-performance web application for creating and downloading video clips from YouTube videos. This is the Go version of [ytclipper](https://github.com/MorrisMorrison/ytclipper), built with modern web technologies and designed for production use.

https://github.com/MorrisMorrison/ytclipper/assets/22982151/bc950608-114f-4d10-b9cd-e46c5cf37333

## Features

### Core Functionality
- **YouTube Video Clipping**: Extract specific segments from YouTube videos with precise start/end timestamps
- **Format Selection**: Support for multiple video formats and quality options available from YouTube
- **Video Preview**: Built-in video player for previewing YouTube videos before clipping
- **Asynchronous Processing**: Job-based processing system with real-time status tracking
- **Progress Tracking**: Real-time progress updates for clip creation jobs

### Production Features
- **Rate Limiting**: Built-in rate limiting to prevent abuse and ensure fair usage
- **Automatic Cleanup**: Scheduled cleanup of old clips and jobs to manage storage efficiently
- **Health Monitoring**: Health check endpoint for monitoring and load balancing
- **Proxy Support**: Optional proxy configuration for yt-dlp operations
- **Responsive UI**: Clean, dark-themed web interface optimized for all devices

### Usage
1. Enter a YouTube URL
2. Select your desired video format and quality
3. Specify start and end times in flexible format (`34`, `1:28`, `1:09:24`)
4. Track progress and download your clip when ready

## Built With
- [Go](https://golang.org/)
- [Echo](https://echo.labstack.com/) - High-performance web framework for Go
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - A YouTube downloader with additional features
- [FFmpeg](https://ffmpeg.org/) - Tool for handling multimedia data
- [Video.js](https://videojs.com/) - HTML5 video player
- [Toastr](https://github.com/CodeSeven/toastr) - Notification library

---

## Quick Start

### Prerequisites
- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Python 3.8+**: [Download Python](https://www.python.org/downloads/)
- **FFmpeg**: [Download FFmpeg](https://ffmpeg.org/download.html)

### Installation
1. **Clone the repository**:
   ```bash
   git clone https://github.com/MorrisMorrison/ytclipper-go.git
   cd ytclipper-go
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Install Python dependencies**:
   ```bash
   pip install yt-dlp certifi requests urllib3
   ```

4. **Run the application**:
   ```bash
   go run main.go
   ```

5. **Access the application**:
   Open your browser and navigate to `http://localhost:8080`

### Development Commands
- **Run application**: `go run main.go` or `make run`
- **Run tests**: `make test`
- **Run end-to-end tests**: `make e2e`
- **Build for production**: `make build-prod`
- **Download static assets**: `make download-static`

## Docker Deployment

### Using Makefile (Recommended)

#### Single Container
```bash
# Build and run (rebuilds only on changes)
make docker-run

# Stop and remove container
make docker-stop

# Restart container
make docker-restart

# Build image only
make docker-build
```

#### Docker Compose
```bash
# Start services (recommended for production)
make compose-up

# Stop services
make compose-down

# Restart services
make compose-restart

# View logs
make compose-logs
```

### Manual Docker Commands
```bash
# Build the Docker image
docker build -t ytclipper .

# Run the container
docker run -d -e PORT=8080 -p 8080:8080 ytclipper
```

### Docker Compose Configuration
The included `docker-compose.yml` provides:
- **Port mapping**: 8080:8080
- **Volume mounting**: `./videos:/app/videos` for persistent storage
- **Environment variables**: Production-ready configuration
- **Restart policy**: `unless-stopped` for reliability

## API Endpoints

The application provides a REST API for programmatic access:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/clip` | Create a new clip job |
| `GET` | `/api/v1/clip` | Download completed clip |
| `GET` | `/api/v1/jobs/status` | Check job status |
| `GET` | `/api/v1/video/duration` | Get video duration |
| `GET` | `/api/v1/video/formats` | Get available video formats |
| `GET` | `/health` | Health check endpoint |

## Configuration

The application can be configured using environment variables:

### Server Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `YTCLIPPER_PORT` | Server port | `8080` |
| `YTCLIPPER_DEBUG` | Enable debug mode | `true` |

### Rate Limiting
| Variable | Description | Default |
|----------|-------------|---------|
| `YTCLIPPER_RATE_LIMITER_RATE` | Requests per second | `5` |
| `YTCLIPPER_RATE_LIMITER_BURST` | Maximum burst requests | `20` |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES` | Token expiration (minutes) | `1` |

### Video Processing
| Variable | Description | Default |
|----------|-------------|---------|
| `YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB` | Maximum clip size (MB) | `300` |
| `YTCLIPPER_YT_DLP_PROXY` | Proxy for yt-dlp | `` |
| `YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS` | yt-dlp timeout (seconds) | `30` |
| `YTCLIPPER_YT_DLP_COOKIES_FILE` | Path to YouTube cookies file | `` |
| `YTCLIPPER_YT_DLP_USER_AGENT` | Browser user agent string | Chrome 120 |
| `YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES` | Number of retry attempts | `3` |

### Cleanup Scheduler
| Variable | Description | Default |
|----------|-------------|---------|
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED` | Enable automatic cleanup | `true` |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES` | Cleanup interval (minutes) | `5` |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH` | Directory to clean | `./videos` |

## Architecture

### System Components
- **Web Server**: Echo-based HTTP server with middleware for rate limiting and logging
- **Job Queue**: In-memory job management system with concurrent-safe operations
- **Video Processor**: yt-dlp and FFmpeg integration for video downloading and clipping
- **Scheduler**: Background cleanup service for automatic file management
- **Static Assets**: Responsive web UI with real-time progress tracking

### Data Flow
1. User submits clip request via web interface
2. System validates input and creates job
3. Job queue processes request asynchronously
4. yt-dlp downloads video segment
5. FFmpeg processes and optimizes clip
6. User downloads completed clip
7. Scheduler automatically cleans up old files

## Security

- **Rate Limiting**: Prevents abuse with configurable request limits
- **Input Validation**: Sanitizes all user inputs and URL parameters
- **File Management**: Automatic cleanup prevents disk space exhaustion
- **Error Handling**: Graceful error handling without exposing internal details
- **YouTube Authentication**: Cookie-based authentication to bypass bot detection

## YouTube Bot Detection

If you encounter "Sign in to confirm you're not a bot" errors, see our comprehensive guide:

📖 **[YouTube Authentication Guide](YOUTUBE_AUTHENTICATION.md)**

**Quick Fix:**
```bash
# Export YouTube cookies and configure
export YTCLIPPER_YT_DLP_COOKIES_FILE="./cookies/youtube_cookies.txt"
export YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES=5
```

## Monitoring

- **Health Checks**: `/health` endpoint for load balancer integration
- **Logging**: Structured logging with configurable levels
- **Metrics**: Job processing statistics and performance metrics
- **Error Tracking**: Comprehensive error logging and tracking

---

## Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow the existing code style and patterns
4. **Run tests**: `make test && make e2e`
5. **Test with Docker**: `make docker-run` to verify containerized deployment
6. **Commit your changes**: `git commit -m 'Add amazing feature'`
7. **Push to the branch**: `git push origin feature/amazing-feature`
8. **Open a Pull Request**

### Development Guidelines
- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation for API changes
- Test both local and Docker deployments
- Ensure all CI checks pass

### Available Make Commands
| Command | Description |
|---------|-------------|
| `make run` | Run application locally |
| `make test` | Run unit tests |
| `make e2e` | Run end-to-end tests |
| `make build` | Build binary |
| `make build-prod` | Production build (tests + assets + build) |
| `make download-static` | Download static assets |
| `make docker-build` | Build Docker image |
| `make docker-run` | Build and run container |
| `make docker-stop` | Stop and remove container |
| `make docker-restart` | Restart container |
| `make compose-up` | Start with Docker Compose |
| `make compose-down` | Stop Docker Compose services |
| `make compose-restart` | Restart Docker Compose services |
| `make compose-logs` | View Docker Compose logs |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) for powerful YouTube downloading capabilities
- [FFmpeg](https://ffmpeg.org/) for video processing
- [Echo](https://echo.labstack.com/) for the excellent web framework
- The original [ytclipper](https://github.com/MorrisMorrison/ytclipper) project

## Roadmap

- [x] Automatically delete downloaded videos
- [x] Track created/finished time of jobs
- [x] Kill suspended jobs after specified timeout
- [x] Save clips in correct file format
- [ ] Add worker pool for better concurrency
- [ ] Add WebSocket support for real-time updates
- [ ] Implement user authentication and quotas
- [ ] Add support for batch processing
- [ ] Implement video thumbnail generation

