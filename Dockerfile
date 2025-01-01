FROM golang:1.23

WORKDIR /app

# Update and install dependencies (Python3, pip, and FFmpeg)
RUN apt-get update && \
    apt-get install -y python3 python3-pip ffmpeg yt-dlp && \
    apt-get clean

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go application
RUN go build -o ytclipper-go .

# Expose the application's port
EXPOSE 8080

# Command to run the application
CMD ["./ytclipper-go"]
