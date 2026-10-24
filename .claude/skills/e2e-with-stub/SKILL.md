---
name: e2e-with-stub
description: Run ytclipper's browser end-to-end suite locally against the deterministic yt-dlp stub, the same way CI does. Use after any change to the page, the API the page calls, or how the app invokes yt-dlp, and when CI's e2e job fails and needs reproducing.
---

# Run the e2e suite against the yt-dlp stub

The e2e suite (`test/e2e.go`, chromedp) drives a real Chrome against a running server. CI replaces yt-dlp with `test/ci/yt-dlp` so the whole flow (page → API → job → download) runs offline and deterministically. Reproduce that locally:

1. **Prerequisites:** Go, ffmpeg (only the startup check needs it), Chrome or Chromium, bash.
2. **Put the stub first on PATH** for the server process:
   ```bash
   chmod +x test/ci/yt-dlp
   export PATH="$PWD/test/ci:$PATH"
   yt-dlp --version   # → ytclipper-ci-stub
   ```
3. **Start the server** with cleanup off, in the background, and wait for health:
   ```bash
   YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED=false go run . > server.log 2>&1 &
   until curl -sf localhost:8080/health; do sleep 1; done
   ```
4. **Run the suite:**
   ```bash
   E2E_DOWNLOAD_TIMEOUT=20 E2E_EXPECT_FAILURE=false go run test/e2e.go
   ```
5. **Stop the server** (`kill %1`) and read `server.log` if anything failed.

## When it fails
- A selector timeout usually means the page changed: update the selector constants at the top of `test/e2e.go` in the same PR.
- "format not found" or a download that never completes usually means the app now calls yt-dlp differently than the stub expects: update `test/ci/yt-dlp`.
- On Windows the stub needs Git Bash on the PATH the server sees; CI runs on Linux, so trust CI when in doubt.
