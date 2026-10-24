# ytclipper-go

[![CI](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MorrisMorrison/ytclipper-go/actions/workflows/ci.yml)

Cut a clip out of any YouTube video and download it. Paste a link, drag the range, get a file.

ytclipper is a small self-hostable web app: a Go server with a plain HTML/JS page in front of [yt-dlp](https://github.com/yt-dlp/yt-dlp) and [FFmpeg](https://ffmpeg.org/). Only the selected section is downloaded, not the whole video. No database, no accounts; clips are deleted automatically after a while.

<img width="2513" height="1197" alt="ytclipper screenshot" src="https://github.com/user-attachments/assets/9480ccbb-a7ec-4407-9428-317c7d3702bb" />

## Features
- The YouTube player loads as soon as you paste a link; nothing waits on the server.
- Pick the range with a two-handle slider under the player (drag, or arrow keys), by typing times (`34`, `1:28` or `1:09:24`), or with "Now" buttons that take the player's current time. **Preview selection** plays just that range.
- One click for a video (H.264/AAC MP4) or audio (MP3) clip. **Advanced** lists every format YouTube offers (resolution, fps, codec, estimated clip size) so you can combine a video and an audio stream yourself.
- Downloads get a readable name: `<title> [from - to].<ext>`.
- Clips are processed as background jobs; the page shows their progress and starts the download when the job is done.
- Works on phones; light and dark theme (follows the system until you pick one).
- The page is plain HTML, CSS and ES modules: no framework, no build step.
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
| `YTCLIPPER_RATE_LIMITER_RATE` | `5` | Requests per second per IP (static files and `/health` are not limited); applies before basic auth |
| `YTCLIPPER_RATE_LIMITER_BURST` | `20` | Burst size |
| `YTCLIPPER_RATE_LIMITER_EXPIRES_IN_MINUTES` | `1` | How long an idle IP's limiter is kept |
| `YTCLIPPER_TRUSTED_PROXIES` | _(empty)_ | Comma-separated IPs or CIDRs of your reverse proxies, e.g. `10.0.0.0/8,192.0.2.10`. Only then is `X-Forwarded-For` used for the client IP; see below |
| `YTCLIPPER_YT_DLP_CLIP_SIZE_LIMIT_IN_MB` | `300` | Maximum clip size (enforced by ffmpeg; a clip that reaches it fails) |
| `YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS` | `60` | Timeout for fetching video info (`yt-dlp -J`) |
| `YTCLIPPER_YT_DLP_DOWNLOAD_TIMEOUT_IN_SECONDS` | `600` | Timeout for downloading and cutting one clip |
| `YTCLIPPER_YT_DLP_EXTRACTOR_RETRIES` | `3` | yt-dlp `--extractor-retries` |
| `YTCLIPPER_YT_DLP_PROXY` | _(empty)_ | HTTP proxy for yt-dlp and ffmpeg, e.g. `http://host:8118` (not SOCKS, see below) |
| `YTCLIPPER_MAX_CLIP_LENGTH_IN_MINUTES` | `30` | Longest clip range a request may ask for; `0` disables the check |
| `YTCLIPPER_MAX_CONCURRENT_JOBS` | `3` | Clips processed at the same time; more wait in the queue |
| `YTCLIPPER_MAX_QUEUED_JOBS` | 4 × `MAX_CONCURRENT_JOBS` | Queued plus processing jobs; further clip requests get `429`. Jobs still queued after 30 minutes are failed by the cleanup |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED` | `true` | Delete old clips and jobs periodically |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_INTERVAL_IN_MINUTES` | `5` | How often cleanup runs |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_RETENTION_IN_MINUTES` | `30` | How long clips and finished jobs are kept |
| `YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH` | `./videos/` | Where clips are stored |

### YouTube blocking datacenter IPs
YouTube often blocks downloads from cloud servers. If clips fail on a VPS but work at home, route yt-dlp through a proxy that exits from a residential network:

```bash
YTCLIPPER_YT_DLP_PROXY=http://proxy-host:8118
```

Use an **HTTP proxy** that supports `CONNECT` (for example tinyproxy, privoxy or gost), not SOCKS. Clips are cut with `--download-sections`, so ffmpeg downloads the video, and ffmpeg only supports HTTP proxies. With a SOCKS proxy, yt-dlp still finds the video through it, but ffmpeg then connects directly, and YouTube refuses with `403 Forbidden` because its media URLs are bound to the IP that looked them up. The app logs a warning at startup when the proxy is SOCKS.

With a residential exit, no cookies or other workarounds are needed.

### Behind a reverse proxy
By default the client IP (for rate limiting and logs) is the address of the TCP connection, and `X-Forwarded-For` is ignored, because any client can send that header. Behind a reverse proxy (Caddy, nginx, Traefik, a Docker network), every request then comes from the proxy and all users share one rate limit. Set `YTCLIPPER_TRUSTED_PROXIES` to the proxy's address or network:

```bash
YTCLIPPER_TRUSTED_PROXIES=172.16.0.0/12   # e.g. the Docker network the proxy is on
```

`X-Forwarded-For` is then followed back through the listed proxies, and only those, to the first address that is not one of them. Make sure the outermost proxy overwrites (not appends to) a client-supplied `X-Forwarded-For`, or at least that only your proxies can reach the app.

## API
The page uses a small JSON API, which you can also call directly. The `http/` folder has ready-made requests for REST clients. Errors are JSON: `{"error": "..."}`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/video/info?youtubeUrl=…` | Title, uploader, duration (s), thumbnail and the format list |
| `POST` | `/api/v1/clip` | Start a clip job; returns the job id as plain text (201) |
| `GET` | `/api/v1/jobs/{id}` | Job state as JSON; `fileName` and `downloadUrl` once completed |
| `GET` | `/api/v1/clip?jobId=…` | Download the finished clip as `<title> [h-mm-ss - h-mm-ss].<ext>` |
| `GET` | `/api/v1/jobs/status?jobId=…` | Legacy: job state as status code (201 running, 200 done, 500 failed) |
| `GET` | `/health` | Health check (never behind auth) |

Video info comes from one `yt-dlp -J` call and is cached in memory per video for 10 minutes, so the formats, duration and clip endpoints don't ask YouTube again. Each format has `id`, `ext`, `kind` (`video` = video only, `audio` = audio only, `both`), `width`, `height`, `fps`, `vcodec`, `acodec`, `tbr`, `abr` (kbit/s), `filesize` (bytes for the whole video, approximate if that's all YouTube reports, `0` if unknown) and `note`.

A clip request names either a `preset` or a `format`:

| Field | Value | Result |
|---|---|---|
| `preset` | `video` | Best H.264/AAC up to 1080p, merged to MP4 |
| `preset` | `audio` | MP3 at ~192 kbit/s |
| `format` | `137` or `137+140` | That yt-dlp format; a video-only format gets the best audio merged in (M4A for MP4 video) |

`from` and `to` are `H:MM:SS` (1 to 3 hour digits, e.g. `0:01:30` or `00:01:30`).

| Response | When |
|---|---|
| `201` + job id | Job queued |
| `400` `{"error": "..."}` | Invalid input, e.g. `Invalid time format. Use HH:MM:SS.`, `End time must be after start time.`, `Clip is too long (max 30 minutes).` |
| `429` `{"error": "The server is busy, try again in a minute."}` | The job queue is full (`YTCLIPPER_MAX_QUEUED_JOBS`) |
| `429` `{"message": "rate limit exceeded"}` | The per-IP rate limit was hit (any endpoint) |

A job can fail later (`status: "error"`) with, among yt-dlp's own errors: `The end time is after the end of the video (3:32).`, `The clip is larger than the 300 MB limit. Choose a shorter range or a smaller format.`, `Timed out while downloading the clip` or `The server was too busy to start the clip. Try again later.`

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
