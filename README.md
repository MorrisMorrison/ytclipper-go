# ytclipper-go

[![CI](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml)

Cut a clip out of any YouTube video and download it. Paste a link, pick a format and a time range, get a file.

ytclipper is a small self-hostable web app: a Go server with a plain HTML/JS page in front of [yt-dlp](https://github.com/yt-dlp/yt-dlp) and [FFmpeg](https://ffmpeg.org/). Only the selected section is downloaded, not the whole video. No database, no accounts; clips are deleted automatically after a while.

<img width="2513" height="1197" alt="ytclipper screenshot" src="https://github.com/user-attachments/assets/9480ccbb-a7ec-4407-9428-317c7d3702bb" />

## Features
- Clip any section of a YouTube video by start and end time (`34`, `1:28` or `1:09:24`).
- Choose from the formats YouTube offers for the video (video, audio, or both).
- Preview the video in the page before clipping.
- Clips are processed as background jobs; the download starts when the job is done.
- Light and dark theme.
- For self-hosting: per-IP rate limiting, optional basic auth, automatic cleanup of old clips, a `/health` endpoint, and an optional proxy for yt-dlp.

## Quick start

### Docker (recommended)
The image bundles yt-dlp, FFmpeg and deno.

```bash
docker run -d -p 8080:8080 ghcr.io/morrismorrison/ytclipper-go:latest
```

Then open http://localhost:8080. With Docker Compose, `docker compose up -d` uses the included `docker-compose.yml` (port 8080, clips in a named volume). The container runs as a non-root user.

### From source
Requirements: Go (see `go.mod`), [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation), FFmpeg, and [deno](https://deno.com/) (yt-dlp uses it to solve YouTube's JavaScript challenges).

```bash
git clone https://github.com/MorrisMorrison/ytclipper-go.git
cd ytclipper-go
go run .
```

The server checks for `yt-dlp` and `ffmpeg` on startup and exits if either is missing.

## Configuration
All settings are environment variables; `.env.example` lists them.

| Variable | Default | Description |
|---|---|---|
| `YTCLIPPER_PORT` | `8080` | HTTP port |
| `YTCLIPPER_DEBUG` | `false` | Echo debug mode and verbose yt-dlp logging |
| `YTCLIPPER_AUTH_USERNAME` | _(empty)_ | Basic auth user; auth is on only when user and password are both set |
| `YTCLIPPER_AUTH_PASSWORD` | _(empty)_ | Basic auth password |
| `YTCLIPPER_RATE_LIMITER_RATE` | `5` | Requests per second per IP (static files are not limited) |
| `YTCLIPPER_RATE_LIMITER_BURST` | `20` | Burst size |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES` | `1` | How long an idle IP's limiter is kept |
| `YTCLIPPER_YT_DLP_CLIP_SIZE_LIMIT_IN_MB` | `300` | Maximum clip size |
| `YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS` | `60` | Timeout for each yt-dlp call |
| `YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES` | `3` | yt-dlp `--extractor-retries` |
| `YTCLIPPER_YT_DLP_PROXY` | _(empty)_ | Proxy for yt-dlp, e.g. `socks5h://host:1080` |
| `YTCLIPPER_MAX_CONCURRENT_JOBS` | `3` | Clips processed at the same time; more wait in the queue |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED` | `true` | Delete old clips and jobs periodically |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES` | `5` | How often cleanup runs |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_RETENTION_IN_MINUTES` | `30` | How long clips and finished jobs are kept |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH` | `./videos/` | Where clips are stored |

### YouTube blocking datacenter IPs
YouTube often blocks downloads from cloud servers. If clips fail on a VPS but work at home, route yt-dlp through a proxy that exits from a residential network:

```bash
YTCLIPPER_YT_DLP_PROXY=socks5h://proxy-host:1080
```

`socks5h://` also resolves DNS through the proxy. With a residential exit, no cookies or other workarounds are needed.

## API
The page uses a small JSON API, which you can also call directly. The `http/` folder has ready-made requests for REST clients.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/video/formats?youtubeUrl=…` | Formats available for the video |
| `GET` | `/api/v1/video/duration?youtubeUrl=…` | Video length in seconds |
| `POST` | `/api/v1/clip` | Start a clip job; returns the job id |
| `GET` | `/api/v1/jobs/status?jobId=…` | Job state |
| `GET` | `/api/v1/clip?jobId=…` | Download the finished clip |
| `GET` | `/health` | Health check (never behind auth) |

Create a clip:

```bash
curl -X POST localhost:8080/api/v1/clip \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","from":"00:00:10","to":"00:00:20","format":"18"}'
```

## Development
```bash
make check       # gofmt check, go vet, unit tests (yt-dlp is mocked), build
make e2e         # browser e2e suite against a running server (needs Chrome)
```

CI never calls real YouTube. It puts a deterministic yt-dlp stub (`test/ci/yt-dlp`) on the `PATH` and runs the unit tests and the e2e suite against it. Every push to `main` publishes the Docker image to GHCR.

Working with Claude Code? See [`CLAUDE.md`](CLAUDE.md).

## Contributing
Issues and pull requests are welcome. Please open an issue first for larger changes, keep PRs focused, and make sure `go vet`, `go test` and CI pass.

## License
[MIT](LICENSE)
