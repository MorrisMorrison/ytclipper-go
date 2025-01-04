FROM golang:1.23

WORKDIR /app

# Update and install system dependencies
RUN apt-get update && \
    apt-get install -y make python3 python3-pip python3-requests python3-urllib3 curl ffmpeg && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install yt-dlp separately, as it may not have a Debian package
RUN python3 -m pip install --no-warn-script-location yt-dlp --break-system-packages

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy 

# Copy the application source code
COPY . .

# Build the Go application
RUN go build -o ytclipper-go .

# Expose the application's port
EXPOSE 8080

# Command to run the application
CMD ["./ytclipper-go"]

