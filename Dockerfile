# --- Build -------------------------------------------------------------------
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ytclipper-go .

# --- Runtime -----------------------------------------------------------------
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl unzip ffmpeg python3 python3-pip && \
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

COPY --from=build /out/ytclipper-go ./ytclipper-go
COPY templates ./templates
COPY static ./static
RUN mkdir -p /app/videos && chown ytclipper:ytclipper /app/videos

USER ytclipper
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD curl -fsS http://localhost:${YTCLIPPER_PORT:-8080}/health || exit 1

CMD ["./ytclipper-go"]
