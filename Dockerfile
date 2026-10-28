# --- Build -------------------------------------------------------------------
FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# The release tag (v0.0.1) or dev-<sha>, set by the publish workflow.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/MorrisMorrison/ytclipper-go/internal/version.Version=${VERSION}" \
    -o /out/ytclipper-go ./cmd/ytclipper

# --- Runtime -----------------------------------------------------------------
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl unzip ffmpeg python3 python3-pip tini && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# yt-dlp from PyPI: Debian's package lags behind YouTube changes. The latest
# release is installed at build time, so rebuilding the image picks up fixes.
RUN python3 -m pip install --no-cache-dir --break-system-packages --upgrade "yt-dlp[default]"

# deno is the JavaScript runtime yt-dlp uses to solve YouTube's JS challenges;
# without it some formats go missing. Pinned for reproducible builds.
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh -s v2.8.3 && \
    deno --version

RUN useradd --system --uid 10001 --home-dir /app ytclipper
WORKDIR /app

# The page and static files are embedded in the binary.
COPY --from=build /out/ytclipper-go ./ytclipper-go
RUN mkdir -p /app/videos /app/.cache && chown ytclipper:ytclipper /app/videos /app/.cache

# /app itself is root-owned; yt-dlp and deno keep their caches (YouTube player
# JS, challenge solver) in a directory the non-root user can write.
ENV XDG_CACHE_HOME=/app/.cache \
    DENO_DIR=/app/.cache/deno

USER ytclipper
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD curl -fsS http://localhost:${YTCLIPPER_PORT:-8080}/health || exit 1

# tini as PID 1 forwards signals and reaps orphaned processes (an ffmpeg left
# behind by a killed yt-dlp).
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["./ytclipper-go"]
