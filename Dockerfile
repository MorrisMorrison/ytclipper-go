FROM golang:1.23

WORKDIR /app

# Update and install dependencies (Python3, pip, and FFmpeg)
RUN sudo apt-get update \
sudo apt-get install -y make python3 python3-pip ffmpeg google-chrome-stable \
python3 -m pip install --upgrade pip yt-dlp requests curl_cffi urllib3

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
