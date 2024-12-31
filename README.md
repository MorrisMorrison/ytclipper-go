# ytclipper-go
Simple web application to create clips from youtube videos and download them.
Go version of https://github.com/MorrisMorrison/ytclipper-go.

https://github.com/MorrisMorrison/ytclipper/assets/22982151/bc950608-114f-4d10-b9cd-e46c5cf37333

1. Enter YouTube URL
2. Select format
2. Enter start time (HH:MM:SS, e.g. 34, 1:28, 1:09:24)
2. Enter start time (HH:MM:SS, e.g. 34, 1:28, 1:09:24)
4. Download Clip

## Built With
- go
- echo 
- yt-dlp
- ffmpeg
- video.js
- toastr

## Run locally
### Requirements
- go
- python3
- python3-pip
- python3-certifi
- ffmpeg
- yt-dlp

### Setup
Run locally
1. Install the required packages
`go mod tidy`
2. Run the app
`go run main.go`

Or run as docker container 
`docker build -t ytclipper .`
`docker run -d -e PORT=8080 -p 8080:8080 ytclipper`

### Build
`make build`


### Configuration

The application can be configured using the following environment variables:

| Environment Variable                         | Description                                            | Default Value  |
|---------------------------------------------|--------------------------------------------------------|----------------|
| `YTCLIPPER_PORT`                             | The port on which the application runs.                | `4001`         |
| `YTCLIPPER_DEBUG`                            | Enable debug mode (true/false).                        | `false`        |
| `YTCLIPPER_PORT_CLIP_SIZE_LIMIT_IN_MB`       | Maximum clip size (in MB) for yt-dlp.                  | `500`          |
| `YTCLIPPER_RATE_LIMITER_RATE`                | Rate limiter requests per second.                      | `5`            |
| `YTCLIPPER_RATE_LIMITER_BURST`               | Maximum number of requests allowed in a burst.         | `20`           |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES`  | Rate limiter token expiration time (in minutes).       | `1`            |


## TODO
- [ ] fix themes
- [ ] fix video player
- [ ] fix progress bar
- [ ] automatically delete downloaded videos
- [ ] track created/finished time of jobs
- [ ] kill suspended jobs after a specified timeout
- [ ] save clips in correct file format
- [x] rewrite everything in go