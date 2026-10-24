.PHONY: run build test fmt fmt-check lint check e2e e2e-real docker-build docker-run docker-stop compose-up compose-down compose-logs

run:
	go run .

build:
	go build -o ./ytclipper .

test:
	go test ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt':"; gofmt -l .; exit 1)

lint:
	go vet ./...

# Everything CI checks before the e2e suite.
check: fmt-check lint test build

# E2E against a server that is already running (CI and the e2e-with-stub skill
# put the yt-dlp stub on PATH first).
e2e:
	E2E_DOWNLOAD_TIMEOUT=20 E2E_EXPECT_FAILURE=false go run test/e2e.go

# E2E against real YouTube, where downloads may legitimately fail.
e2e-real:
	E2E_DOWNLOAD_TIMEOUT=60 E2E_EXPECT_FAILURE=true go run test/e2e.go

docker-build:
	docker build -t ytclipper:latest .

docker-run: docker-build
	docker run -d --name ytclipper -p 8080:8080 ytclipper:latest

docker-stop:
	docker rm -f ytclipper 2>/dev/null || true

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f
