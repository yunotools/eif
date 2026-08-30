.PHONY: fmt test build run check

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

build:
	go build -o bin/eif ./cmd/eif

run:
	go run ./cmd/eif

check: fmt test build
