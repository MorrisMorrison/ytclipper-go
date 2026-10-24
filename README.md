# ytclipper-go

[![CI](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml)

Cut a clip out of any YouTube video and download it. Paste a link, pick a format and a time range, get a file.

ytclipper is a small self-hostable web app: a Go server with a plain HTML/JS page in front of [yt-dlp](https://github.com/yt-dlp/yt-dlp) and [FFmpeg](https://ffmpeg.org/). Only the selected section is downloaded, not the whole video. No database, no accounts; clips are deleted automatically after a while.

<img width="2513" height="1197" alt="ytclipper screenshot" src="https://github.com/user-attachments/assets/9480ccbb-a7ec-4407-9428-317c7d3702bb" />

## Features
- Clip any section of a YouTube video by start and end time (`34`, `1:28` or `1:09:24`).
- One click for a video (H.264/AAC MP4) or audio (MP3) clip, or pick any format YouTube offers.
- Downloads get a readable name: `<title> [from - to].<ext>`.
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
The page uses a small JSON API, which you can also call directly. The `http/` folder has ready-made requests for REST clients. Errors are JSON: `{"error": "..."}`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/video/info?youtubeUrl=…` | Title, uploader, duration (s), thumbnail and the format list |
| `POST` | `/api/v1/clip` | Start a clip job; returns the job id as plain text (201) |
| `GET` | `/api/v1/jobs/{id}` | Job state as JSON; `fileName` and `downloadUrl` once completed |
| `GET` | `/api/v1/clip?jobId=…` | Download the finished clip as `<title> [h-mm-ss - h-mm-ss].<ext>` |
| `GET` | `/api/v1/video/formats?youtubeUrl=…` | Legacy: formats in the old page's shape |
| `GET` | `/api/v1/video/duration?youtubeUrl=…` | Legacy: video length in seconds |
| `GET` | `/api/v1/jobs/status?jobId=…` | Legacy: job state as status code (201 running, 200 done, 500 failed) |
| `GET` | `/health` | Health check (never behind auth) |

Video info comes from one `yt-dlp -J` call and is cached in memory per video for 10 minutes, so the formats, duration and clip endpoints don't ask YouTube again. Each format has `id`, `ext`, `kind` (`video` = video only, `audio` = audio only, `both`), `width`, `height`, `fps`, `vcodec`, `acodec`, `tbr`, `abr` (kbit/s), `filesize` (bytes for the whole video, approximate if that's all YouTube reports, `0` if unknown) and `note`.

A clip request names either a `preset` or a `format`:

| Field | Value | Result |
|---|---|---|
| `preset` | `video` | Best H.264/AAC up to 1080p, merged to MP4 |
| `preset` | `audio` | MP3 at ~192 kbit/s |
| `format` | `137` or `137+140` | That yt-dlp format; a video-only format gets the best audio merged in |

```bash
curl -X POST localhost:8080/api/v1/clip   -H 'Content-Type: application/json'   -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","from":"00:00:10","to":"00:00:20","preset":"video"}'
# -> 3f1c...  (job id)
curl localhost:8080/api/v1/jobs/3f1c...
# -> {"id":"3f1c...","status":"completed","error":"","fileName":"Video title [0-00-10 - 0-00-20].mp4","downloadUrl":"/api/v1/clip?jobId=3f1c..."}
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
