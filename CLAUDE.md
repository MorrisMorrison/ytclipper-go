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
| `videoprocessing/` | Everything that talks to yt-dlp: argument building, execution with timeout, output parsing |
| `jobs/` | In-memory job store (map + mutex) |
| `scheduler/` | Periodic cleanup of old clips and jobs |
| `config/` | `YTCLIPPER_*` environment variables → `config.CONFIG` |
| `middleware/` | Optional basic auth |
| `templates/`, `static/` | The page: `index.html`, CSS, vanilla ES modules (no build step) |
| `test/e2e.go` | Browser end-to-end suite (chromedp) against a running server |
| `test/ci/yt-dlp` | Deterministic offline yt-dlp stub used by CI and the e2e suite |
| `http/` | `.http` request collections for manual API testing |

## Commands
```bash
go build ./...           # build
go vet ./...             # lint
gofmt -l .               # must print nothing
go test ./...            # unit tests (yt-dlp is mocked, nothing external needed)
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
- **yt-dlp:** build arguments in `videoprocessing`, always through `execute` (timeout + common args such as the optional proxy). Never interpolate user input into a shell; pass args as a slice.
- **Front-end:** vanilla ES modules, no framework, no bundler. The main flow must never block on a slow yt-dlp call.
- **Security:** treat the URL and timestamps as untrusted; validate server-side even if the UI validated too.

## Workflow
- Every change starts from a GitHub issue; one branch and one PR per issue, `Closes #N` in the PR body.
- Branch names: `feat/…`, `fix/…`, `chore/…`, `docs/…`.
- PR description: what changed, why, and what was verified (and what wasn't).
- CI must be green before merge.
- `main` publishes the Docker image to GHCR on every push.
