# Configuration
All settings are environment variables; [`.env.example`](../.env.example) lists them with their defaults.

| Variable | Default | Description |
|---|---|---|
| `YTCLIPPER_PORT` | `8080` | HTTP port |
| `YTCLIPPER_DEBUG` | `false` | Echo debug mode and debug-level logs (incl. yt-dlp output) |
| `YTCLIPPER_LOG_FORMAT` | `text` | Log format: `text` or `json` (structured, for log collectors) |
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
| `YTCLIPPER_DEV_ASSETS_DIR` | _(empty)_ | Development only: serve the page and static files from this directory (the repo's `web/`) instead of the embedded copy, so edits show up on reload |

## YouTube blocking datacenter IPs
YouTube often blocks downloads from cloud servers. If clips fail on a VPS but work at home, route yt-dlp through a proxy that exits from a residential network:

```bash
YTCLIPPER_YT_DLP_PROXY=http://proxy-host:8118
```

Use an **HTTP proxy** that supports `CONNECT` (for example tinyproxy, privoxy or gost), not SOCKS. Clips are cut with `--download-sections`, so ffmpeg downloads the video, and ffmpeg only supports HTTP proxies. With a SOCKS proxy, yt-dlp still finds the video through it, but ffmpeg then connects directly, and YouTube refuses with `403 Forbidden` because its media URLs are bound to the IP that looked them up. The app logs a warning at startup when the proxy is SOCKS.

With a residential exit, no cookies or other workarounds are needed.

## Behind a reverse proxy
By default the client IP (for rate limiting and logs) is the address of the TCP connection, and `X-Forwarded-For` is ignored, because any client can send that header. Behind a reverse proxy (Caddy, nginx, Traefik, a Docker network), every request then comes from the proxy and all users share one rate limit. Set `YTCLIPPER_TRUSTED_PROXIES` to the proxy's address or network:

```bash
YTCLIPPER_TRUSTED_PROXIES=172.16.0.0/12   # e.g. the Docker network the proxy is on
```

`X-Forwarded-For` is then followed back through the listed proxies, and only those, to the first address that is not one of them. Make sure the outermost proxy overwrites (not appends to) a client-supplied `X-Forwarded-For`, or at least that only your proxies can reach the app.

---
[Back to the README](../README.md)
