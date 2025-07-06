.PHONY: clean build download-static test e2e build-prod fmt lint docker-build docker-run docker-stop compose-up compose-down

run:
	go run main.go

download-static:
	go run build/download_static.go

test:
	go test ./... -v

e2e:
	go run test/e2e.go

build:
	go build -o ./ytclipper main.go

fmt:
	go fmt ./...

lint:
	go vet ./...

build-prod: fmt lint test download-static build

# Docker commands
docker-build:
	docker build -t ytclipper:latest .

docker-run: docker-build
	docker run -d --name ytclipper -p 8080:8080 ytclipper:latest

docker-stop:
	docker stop ytclipper 2>/dev/null || true
	docker rm ytclipper 2>/dev/null || true

docker-restart: docker-stop docker-run

# Docker Compose commands
compose-up:
	docker-compose up -d --build

compose-down:
	docker-compose down

compose-restart: compose-down compose-up

compose-logs:
	docker-compose logs -f
