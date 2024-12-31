
markdown
Copy code
# ytclipper-go

A simple web application to create clips from YouTube videos and download them. This is the Go version of [ytclipper](https://github.com/MorrisMorrison/ytclipper).

![ytclipper demo](https://github.com/MorrisMorrison/ytclipper/assets/22982151/bc950608-114f-4d10-b9cd-e46c5cf37333)

## Features
1. Enter YouTube URL.
2. Select the desired format.
3. Specify start and end times in `HH:MM:SS` format (e.g., `34`, `1:28`, `1:09:24`).
4. Download your clip.

## Built With
- [Go](https://golang.org/)
- [Echo](https://echo.labstack.com/) - High-performance web framework for Go
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - A YouTube downloader with additional features
- [FFmpeg](https://ffmpeg.org/) - Tool for handling multimedia data
- [Video.js](https://videojs.com/) - HTML5 video player
- [Toastr](https://github.com/CodeSeven/toastr) - Notification library

---

## Running Locally

### Requirements
Make sure you have the following dependencies installed:
- [Go](https://golang.org/)
- Python 3
- Python `pip`
- [Certifi](https://pypi.org/project/certifi/) (`python3-certifi`)
- [FFmpeg](https://ffmpeg.org/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)

### Setup
1. Install Go dependencies:
`go mod tidy`
2. Run the application:
`go run main.go`

## Running with docker
Build and run the application in a Docker container:

1. Build the Docker image:
`docker build -t ytclipper .`
2. Run the container
`docker run -d -e PORT=8080 -p 8080:8080 ytclipper`

### Build
To build the application:
`make build`

### Configuration

The application can be configured using the following environment variables:

| Environment Variable                         | Description                                            | Default Value  |
|---------------------------------------------|--------------------------------------------------------|----------------|
| `YTCLIPPER_PORT`                             | The port on which the application runs.                | `8080`         |
| `YTCLIPPER_DEBUG`                            | Enable debug mode (true/false).                        | `true`        |
| `YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB`       | Maximum clip size (in MB) for yt-dlp.                  | `300`          |
| `YTCLIPPER_RATE_LIMITER_RATE`                | Rate limiter requests per second.                      | `5`            |
| `YTCLIPPER_RATE_LIMITER_BURST`               | Maximum number of requests allowed in a burst.         | `20`           |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES`  | Rate limiter token expiration time (in minutes).       | `1`            |

---

## TODO
- [ ] fix themes
- [ ] fix video player
- [ ] fix progress bar
- [ ] automatically delete downloaded videos
- [ ] track created/finished time of jobs
- [ ] kill suspended jobs after a specified timeout
- [ ] save clips in correct file format
- [x] rewrite everything in go

