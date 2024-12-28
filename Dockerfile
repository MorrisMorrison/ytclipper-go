FROM golang:1.20

WORKDIR /app

RUN apt-get update && \
    apt-get install -y python3 python3-pip ffmpeg && \
    pip3 install --no-cache-dir yt-dlp

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o ytclipper-go .

EXPOSE 8080

CMD ["./ytclipper-go"]