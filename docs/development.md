# Development
The server's entry point is `cmd/ytclipper`; the packages live in `internal/` and the page (`web/templates/`, `web/static/`) is embedded into the binary with `go:embed`. See [`CLAUDE.md`](../CLAUDE.md) for the full layout and conventions.

```bash
make run          # go run ./cmd/ytclipper
make run-dev      # same, but serves web/ from disk (YTCLIPPER_DEV_ASSETS_DIR=web): edit HTML, CSS or JS and reload
make check        # gofmt check, go vet, unit tests (yt-dlp is mocked), build
make e2e          # browser e2e suite (needs Chrome; starts its own servers)
make coverage     # unit and e2e coverage profiles (e2e part needs Linux/macOS)
make screenshots  # renders the README screenshots into docs/
```

The e2e suite (`go test -tags e2e ./test/e2e/`) builds the app plus fake `yt-dlp` and `ffmpeg` binaries (`test/fakes/`), starts the server with them first on the `PATH`, and drives a headless Chrome through the page: presets, advanced formats, the trimmer, job errors, a full queue, the download itself and the phone layout. Failure screenshots and logs go to `test/e2e/artifacts/`. CI never calls real YouTube. Every push to `main` publishes the Docker image to GHCR.

Working with Claude Code? See [`CLAUDE.md`](../CLAUDE.md) and the project skills in [`.claude/skills/`](../.claude/skills/). The page's design tokens are described in [`STYLEGUIDE.md`](STYLEGUIDE.md).

## Contributing
Issues and pull requests are welcome. Please open an issue first for larger changes, keep PRs focused, and make sure `go vet`, `go test` and CI pass.

---
[Back to the README](../README.md)
