# CLAUDE.md

Guidance for Claude Code (and anyone else) working in this repository. Keep it short and correct: when behaviour, commands or layout change, update this file and the README in the same PR.

## What this is
ytclipper-go is a small self-hostable web app that cuts a clip out of a YouTube video and returns it as a file. A Go (Echo) server renders one page, exposes a small JSON API, and runs clip jobs in the background by shelling out to **yt-dlp** (which uses **ffmpeg** for cutting and merging). There is no database: jobs live in memory and clips on disk, both cleaned up by a scheduler.

## Layout
| Path | Responsibility |
|---|---|
| `main.go` | Dependency check, Echo setup, middleware (basic auth, logger, rate limiter), start |
| `routes/` | Route table |
| `api/` | HTTP handlers and input validation |
| `videoprocessing/` | Everything that talks to yt-dlp: argument building (presets), execution with timeout, `-J` parsing and the per-video info cache, download file names |
| `jobs/` | In-memory job store; `GetJobById` returns copies, mutate only through its functions |
| `scheduler/` | Periodic cleanup of old clips and jobs |
| `config/` | `YTCLIPPER_*` environment variables → `config.CONFIG` |
| `middleware/` | Optional basic auth |
| `templates/`, `static/` | The page: `index.html`, CSS, vanilla ES modules (no build step) |
| `test/e2e.go` | Browser end-to-end suite (chromedp) against a running server |
| `test/ci/yt-dlp` | Deterministic offline yt-dlp stub used by CI and the e2e suite |
| `http/` | `.http` request collections for manual API testing |

## Commands
```bash
make check               # gofmt check + go vet + unit tests + build (what CI runs first)
go test -race ./...      # CI also runs the race detector
make e2e                 # e2e suite; needs a running server + Chrome (see the e2e-with-stub skill)
```
Running the server locally needs `yt-dlp`, `ffmpeg` and (for current YouTube) `deno` on `PATH`; the app exits at startup if yt-dlp or ffmpeg is missing. The Docker image bundles all three.

## How tests isolate yt-dlp
- **Unit tests** never run yt-dlp: `videoprocessing` executes commands through the package variable `execContext`, which tests replace with a fake (see `videoprocessing_test.go`). Follow that pattern for any new command.
- **CI and e2e** put `test/ci/yt-dlp` first on `PATH`. It answers the handful of yt-dlp invocations the app makes with fixed output and writes a dummy file for downloads. **If you change how the app calls yt-dlp, update the stub in the same PR**, or CI breaks.
- Nothing in CI talks to real YouTube (datacenter IPs get bot-blocked, so it would be flaky and meaningless). Say so in PRs: "verified against the stub, not real YouTube".

## Conventions
- **Go:** standard library style, `gofmt`. Handlers validate input and return JSON errors as `{"error": "..."}`. Log through `glogger.Log`. Wrap errors with `%w`. No new global state beyond the existing `config.CONFIG` and job store.
- **Config:** every setting is a `YTCLIPPER_*` env var with a sensible default in `config/config.go`, listed in `.env.example` and the README table.
- **yt-dlp:** build arguments in `videoprocessing`, always through `execute` (timeout + common args such as the optional proxy). Never interpolate user input into a shell; pass args as a slice. `-J` uses the command timeout, clip downloads the (longer) download timeout. On Unix each command runs in its own process group and a timeout kills the whole group (yt-dlp starts ffmpeg for `--download-sections`); see `proc_unix.go`. stdout and stderr are kept apart: parse stdout, summarize errors from both.
- **Size and length limits:** yt-dlp always downloads sections with ffmpeg, which ignores `--max-filesize`, so the size limit is passed to ffmpeg (`--downloader-args "ffmpeg:-fs N"`) and a clip that reaches it is deleted and the job failed. Clip length is capped in validation (`YTCLIPPER_MAX_CLIP_LENGTH_IN_MINUTES`) and the job checks `to` against the video's duration.
- **Video metadata** comes only from `GetVideoInfo` (`yt-dlp -J`, cached per video ID for 10 min, concurrent callers share one call). Don't add another yt-dlp call for metadata; read it from the info. Every clip job calls it at start (title, duration, format kinds). The old `-F`/`--get-duration` text parsing and the `/video/formats` and `/video/duration` endpoints are gone.
- **Jobs:** `POST /api/v1/clip` creates jobs through `jobs.TryNewJob`, which enforces `YTCLIPPER_MAX_QUEUED_JOBS` (429 when full). A worker only processes a job if `jobs.StartJob` returns true; cleanup fails jobs that stayed queued for 30 minutes.
- **Middleware order:** logger, recover, rate limiter, basic auth (so failed logins are rate limited). The client IP is the connection address unless `YTCLIPPER_TRUSTED_PROXIES` is set; never trust `X-Forwarded-For` otherwise.
- **Clip format:** a request carries either a preset (`video`, `audio`) or a numeric format (`137`, `137+140`). Never pass user-supplied format selectors through to yt-dlp beyond that pattern.
- **File names** shown to users go through `SanitizeFileName`; the clip on disk is always named by job ID. The download sets `Content-Disposition` with an ASCII `filename` fallback plus the real name in `filename*` (RFC 5987); don't use `c.Attachment`, which only writes the raw name.
- **Front-end:** vanilla ES modules, no framework, no bundler, no vendored libraries (`static/scripts/`: `main.js` controller, `player.js` YouTube IFrame API wrapper, `trimmer.js` range slider, `formats.js` advanced lists, `api.js`, `time.js`, `toast.js`, `theme.js`). The main flow must never block on a slow yt-dlp call: simple mode needs no server call before clipping, and advanced mode fetches `/video/info` only when opened. The YouTube player is optional; the page must keep working (typed times) when it doesn't load, which is the case in CI (the e2e blocks YouTube hosts). Errors come as `{"error"}` (Echo's rate limiter: `{"message"}`); job polling retries 429/5xx with backoff. Styles use the semantic tokens in `static/css/styles.css` (see `docs/STYLEGUIDE.md`).
- **Errors shown to users** come from `summarizeError` (last yt-dlp `ERROR:` line). Never return raw command output: it contains the full command line.
- **Security:** treat the URL and timestamps as untrusted; validate server-side even if the UI validated too.

## Workflow
- Every change starts from a GitHub issue; one branch and one PR per issue, `Closes #N` in the PR body.
- Branch names: `feat/…`, `fix/…`, `chore/…`, `docs/…`.
- PR description: what changed, why, and what was verified (and what wasn't).
- CI must be green before merge.
- `main` publishes the Docker image to GHCR on every push.
