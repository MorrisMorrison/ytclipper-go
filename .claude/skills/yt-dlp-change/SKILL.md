---
name: yt-dlp-change
description: Change how ytclipper calls or parses yt-dlp (new flags, a new format preset, a new metadata field, a different output template) end to end without breaking CI. Use for any edit in internal/videoprocessing/ that touches command arguments or yt-dlp output.
---

# Changing a yt-dlp invocation

yt-dlp is the app's only real dependency on the outside world, and three places must agree on how it's called:

| Place | Role |
|---|---|
| `internal/videoprocessing/` | Builds args, runs them through `execute`, parses output |
| `internal/videoprocessing/*_test.go` | Fakes `execContext` and asserts on args and parsing |
| `test/fakes/yt-dlp` | Offline fake yt-dlp (Go) the e2e suite runs against |

## Steps
1. **Check yt-dlp's docs** for the flag (`yt-dlp --help`, or the README on GitHub). Prefer structured output (`-J`, `--print`) over parsing human-readable tables.
2. **Write the test first** in `videoprocessing`: fake `execContext`, assert the exact argument slice, feed realistic output captured from a real yt-dlp run (trim it, keep it in the test file).
3. **Implement** through `execute(...)` so the timeout and common args (proxy, retries) still apply. Args are a slice; never build a shell string from user input.
4. **Update the fake** `test/fakes/yt-dlp` so it answers the new invocation with matching fixed output (add new value-taking flags to `flagsWithValue`).
5. **Run** `go vet ./... && go test ./...`, then the e2e suite (`make e2e`, see the `e2e` skill) if the page flow is affected.
6. **Document** new config in `internal/config/config.go`, `.env.example` and `docs/configuration.md` (API changes: `docs/api.md`).
7. In the PR, state that it was verified against the fake, and whether anyone tried it against real YouTube.

## Gotchas
- Format availability differs per video and changes over time; never hard-code a format id outside tests and the fake.
- "video only" formats have no audio; merge best audio (`<id>+bestaudio`, or `<id>+bestaudio[ext=m4a]/<id>+bestaudio` for MP4 video so the merge needs no re-encode) unless the user explicitly wants video only.
- Merging or remuxing needs ffmpeg; extracting audio (`-x --audio-format mp3`) re-encodes.
- `--download-sections` downloads go through ffmpeg, which only supports **HTTP** proxies. With a SOCKS proxy yt-dlp extracts through it but ffmpeg connects directly, and YouTube answers 403 (media URLs are bound to the extracting IP). `--no-warnings` hides yt-dlp's warning about it.
