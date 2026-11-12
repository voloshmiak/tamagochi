.PHONY: run run-engine lint build web up down

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

up:
	docker compose up --build

down:
	docker compose down