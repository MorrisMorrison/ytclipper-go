.PHONY: run build test fmt fmt-check lint check e2e coverage coverage-unit coverage-e2e screenshots docker-build docker-run docker-stop compose-up compose-down compose-logs

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
	go vet -tags e2e ./test/e2e/

# Everything CI checks before the e2e suite.
check: fmt-check lint test build

# Browser e2e suite (needs Chrome). Builds the app and the fake yt-dlp and
# ffmpeg (test/fakes), starts its own servers and stops them. To test a
# server you started yourself, set E2E_BASE_URL.
e2e:
	go test -tags e2e -count=1 ./test/e2e/

# Coverage profiles for Codecov: coverage-unit.out (go test) and
# coverage-e2e.out (the e2e suite against a -cover build of the app; unix only,
# because the server must exit gracefully to write its counters).
coverage: coverage-unit coverage-e2e

coverage-unit:
	go test -covermode=atomic -coverprofile=coverage-unit.out ./...

coverage-e2e:
	rm -rf .cover/e2e && mkdir -p .cover/e2e
	E2E_COVERDIR="$(CURDIR)/.cover/e2e" go test -tags e2e -count=1 ./test/e2e/
	go tool covdata textfmt -i=.cover/e2e -o coverage-e2e.out

# Renders the README images into $(SCREENSHOTS_DIR)/docs/screenshot-{light,dark}.png.
# Unlike the e2e suite, the browser may reach YouTube, so the real player shows.
SCREENSHOTS_DIR ?= $(CURDIR)
screenshots:
	E2E_SCREENSHOTS="$(SCREENSHOTS_DIR)" go test -tags e2e -count=1 -v -run TestScreenshots ./test/e2e/

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
