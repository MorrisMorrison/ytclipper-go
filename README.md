# ytclipper-go
[![main](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/build_and_deploy_prod.yml)

A simple web application to create clips from YouTube videos and download them. This is the Go version of [ytclipper](https://github.com/MorrisMorrison/ytclipper).

https://github.com/MorrisMorrison/ytclipper/assets/22982151/bc950608-114f-4d10-b9cd-e46c5cf37333

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
- [python3](https://www.python.org/downloads/)
- [pip](https://pypi.org/project/pip/)
- [certifi](https://pypi.org/project/certifi/)
- [ffmpeg](https://ffmpeg.org/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [requests](https://pypi.org/project/requests/)
- [urrlib3](https://pypi.org/project/urllib3/)

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

### Build Prod
To run all tests and build the application for production:
`make build-prod`

### Tests
To run tests
`make test`
`make e2e`

### Configuration

The application can be configured using the following environment variables:

| Environment Variable                         | Description                                            | Default Value  |
|----------------------------------------------|--------------------------------------------------------|----------------|
| `YTCLIPPER_PORT`                             | The port on which the application runs.                | `8080`         |
| `YTCLIPPER_DEBUG`                            | Enable debug mode (true/false).                        | `true`         |
| `YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB`       | Maximum clip size (in MB) for yt-dlp.                  | `300`          |
| `YTCLIPPER_RATE_LIMITER_RATE`                | Rate limiter requests per second.                      | `5`            |
| `YTCLIPPER_RATE_LIMITER_BURST`               | Maximum number of requests allowed in a burst.         | `20`           |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES`  | Rate limiter token expiration time (in minutes).       | `1`            |
| `YTCLIPPER_YT_DLP_PROXY`                     | Proxy used by yt-dlp.                     | ``             |
| `YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS`| yt-dlp command timeout (in seconds).                   | `30`           |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES`                     | Execution interval used by scheduler (in minutes).                     | `5`             |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH`| Directory to search for old files to delete.                   | `./videos`           |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED`| Flag to enable/disable the scheduler.                   | `true`           |
---

## TODO
- [x] automatically delete downloaded videos
- [x] track created/finished time of jobs
- [x] kill suspended jobs after a specified timeout
- [x] save clips in correct file format
- [ ] add workerpool

