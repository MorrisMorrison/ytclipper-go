---
name: e2e
description: Run, debug or extend ytclipper's browser end-to-end suite (go test -tags e2e ./test/e2e/, chromedp) with the fake yt-dlp and ffmpeg, the same way CI does. Use after any change to the page, the API the page calls, or how the app invokes yt-dlp; when CI's e2e job fails; when adding an e2e test; or to re-render the README screenshots.
---

# The e2e suite

`test/e2e` is a normal `go test` package behind the `e2e` build tag. It drives a headless Chrome against the real app. Nothing talks to real YouTube: the app runs with the fakes in `test/fakes/` first on `PATH`, and the browser can't resolve YouTube hosts.

## Run it
```bash
make e2e                                                                  # = go test -tags e2e -count=1 ./test/e2e/
go test -tags e2e -count=1 -run 'TestTrimmer|TestJobError' ./test/e2e/    # a subset
```
Needs Go and Chrome (or Chromium/Edge: `E2E_CHROME=<path>` if chromedp doesn't find it). No ffmpeg, yt-dlp or bash needed; it works on Windows too. A full run takes about 20 s.

What `TestMain` does: builds the app (`./cmd/ytclipper`) and `test/fakes/yt-dlp` + `test/fakes/ffmpeg` into a temp dir (`.exe` on Windows). Servers start on first use, in the test directory (the page is embedded, so the working directory does not matter), on a free port, with that dir first on `PATH`, cleanup off, short timeouts and a high rate limit; the harness waits for `/health` and kills them at the end (process group on Unix). Tests that need a different configuration start their own server with `ownServer` (e.g. `TestQueueFull` with a one-job queue).

`E2E_BASE_URL=http://host:port` targets a server you started yourself instead. It must have the fake yt-dlp first on `PATH` (build it with `go build -o <dir>/yt-dlp ./test/fakes/yt-dlp`); tests that need their own server are skipped. The assertions expect the fake's answers, so this is not a way to test real YouTube.

## The fake yt-dlp (`test/fakes/yt-dlp`)
- `--version`; `-J` returns fixed JSON ("Stub Video", 30 s, formats 137/136 video-only, 18 combined, 140/251 audio); downloads write 64 KiB to the `-o` template (`%(ext)s` → mp3 with `-x`, else `--merge-output-format`, else mp4); non-YouTube URLs fail with `Unsupported URL`.
- Magic video IDs (one shared server can test all of them): `failfailfai` (download fails with `Video unavailable`), `slowslowslo` (slow download), `slowinfoslo` (slow `-J`), `infofailinf` (`-J` fails), `notitlenoti` (empty title).
- Env per server: `FAKE_YTDLP_TITLE`, `FAKE_YTDLP_UPLOADER`, `FAKE_YTDLP_DURATION`, `FAKE_YTDLP_SLOW_SECONDS` (default 4).
- **If the app changes how it calls yt-dlp, update the fake in the same PR.** It ignores unknown flags but needs to know which ones take a value (`flagsWithValue`).

## When it fails
- Artifacts land in `test/e2e/artifacts/` (or `E2E_ARTIFACTS`): `<Test>.png` (full page), `<Test>.console.txt` (console plus every fetch the page made) and `server-<name>.log`. CI uploads them as `e2e-artifacts` on failure.
- A selector timeout usually means the page changed: update the selectors at the top of `e2e_test.go`.
- A job that never completes, or "format not found", usually means the app calls yt-dlp differently than the fake expects.
- Flaky only in CI? Tests run in parallel, each in its own browser context; don't share state between tests, and poll (`poll[T]`) instead of sleeping.

## Writing a test
`p := newPage(t, ...options)` opens a fresh tab (own storage, downloads denied, fetch log in `window.__fetches`). Helpers: `p.openVideo(url)`, `p.setRange(from, to)`, `p.submit()` / `p.clip()`, `p.clipPosts()`, `p.fetches(substr)`, `eval[T]`, `poll[T]`. Call `t.Parallel()` unless the test needs exclusive state. Prefer a magic ID in the fake over a new server.

## README screenshots
`make screenshots` (or `E2E_SCREENSHOTS=<dir> go test -tags e2e -run TestScreenshots ./test/e2e/`) writes `<dir>/docs/screenshot-light.png` and `-dark.png` at 1280 px. This mode lets the browser reach YouTube, so the real player shows Big Buck Bunny (`aqz-KE-bpKQ`, CC BY 3.0, Blender Foundation); the fake gives the server its title and length. Check the images before committing them.
