.PHONY: run run-engine lint build web

run:
	go run ./cmd/api

run-engine:
	go run ./cmd/pet-engine

lint:
	gofmt -l .
	go vet ./...

build:
	go build -o bin/ ./cmd/...

web:
	cd web && npm run dev
