# ytclipper-go
[![main](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml)

A high-performance web application for creating and downloading video clips from YouTube videos. This is the Go version of [ytclipper](https://github.com/MorrisMorrison/ytclipper), built with modern web technologies and designed for production use.


https://github.com/user-attachments/assets/8ab2d567-0ca7-44c6-9c76-07203b2fd986



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
- **Advanced Bot Detection Bypass**: 3-tier fallback system with optional cookie/proxy support
- **Intelligent Fallback**: Multi-tier fallback strategies for maximum compatibility
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
| `YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS` | yt-dlp timeout (seconds) | `60` |
| `YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES` | Number of retry attempts | `3` |
| `YTCLIPPER_YT_DLP_SLEEP_INTERVAL` | Base sleep interval (seconds) | `2` |
| `YTCLIPPER_YT_DLP_ENABLE_USER_AGENT_ROTATION` | Enable rotating user agents | `true` |

### Anti-Bot Detection
| Variable | Description | Default |
|----------|-------------|---------|
| `YTCLIPPER_YT_DLP_USER_AGENT` | Custom user agent (overrides rotation) | `` |
| `YTCLIPPER_YT_DLP_COOKIES_FILE` | Path to cookies file (optional) | `` |
| `YTCLIPPER_YT_DLP_PROXY` | Proxy server (optional) | `` |

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
- **Minimal Configuration**: Optional cookie/proxy support with no credentials stored by default

## YouTube Bot Detection Bypass

The application uses a **3-tier fallback strategy** to maximize compatibility and success rate when downloading YouTube videos:

### **Tier 1: Legacy Configuration (Cookies/Proxy)** (Primary)
**Environment Variable Configuration**: Use `YTCLIPPER_YT_DLP_COOKIES_FILE` for cookie file path and `YTCLIPPER_YT_DLP_PROXY` for proxy server  
**Highest Success Rate**: Provides authentication for restricted content when configured  
**Geographic Restrictions**: Bypass location-based restrictions with proxy support  
**Network Diversity**: Alternative network path for enhanced reliability  

### **Tier 2: Enhanced Cookie-Free Anti-Detection** (Secondary)
**No Authentication Required**: Advanced spoofing without cookies or user interaction  
**User Agent Rotation**: 6 modern browser user agents automatically rotated  
**Enhanced Headers**: Browser-like HTTP headers for authenticity  
**Progressive Retries**: Automatic retry with different strategies on failure  

### **Tier 3: Alternative Extraction** (Final Fallback)
**Alternative Extraction Methods**: Different header patterns and extraction methods  
**Backup Strategies**: Final fallback for difficult videos  
**Comprehensive Coverage**: Handles edge cases and challenging content  

### Quick Setup (Default Configuration)
```bash
# Default configuration (no setup required)
export YTCLIPPER_YT_DLP_ENABLE_USER_AGENT_ROTATION=true
export YTCLIPPER_YT_DLP_SLEEP_INTERVAL=2
export YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES=3

# Optional: Add cookie file for enhanced compatibility
export YTCLIPPER_YT_DLP_COOKIES_FILE=/path/to/cookies.txt

# Optional: Add proxy for geographic restrictions
export YTCLIPPER_YT_DLP_PROXY=http://proxy-server:port
```

### How It Works
1. **Automatic Fallback**: System tries each tier progressively until successful
2. **No User Interaction**: Runs completely automatically with environment variables
3. **Maximum Compatibility**: Covers edge cases and restricted content
4. **Optimal Configuration**: Works best with cookies/proxy configured, but provides fallback options

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

