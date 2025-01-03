.PHONY: clean build download-static test e2e build-prod

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

build-prod: test download-static build
