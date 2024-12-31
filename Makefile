.PHONY: clean build download-static

download-static:
	go run build/download_static.go

build: download-static
	go build -o ./ytclipper main.go