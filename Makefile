.PHONY: clean build download-static test e2e build-prod

# Download static files
download-static:
	go run build/download_static.go

# Run unit tests
test:
	go test ./... -v

# Run end-to-end tests
e2e:
	go run test/e2e.go

# Build the application
build:
	go build -o ./ytclipper main.go

# Production build (download static, run tests, and build binary)
build-prod: test download-static build
