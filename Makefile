.PHONY: run run-engine ping vapid migrate lint build web generate generate-api generate-sqlc deps-update up down

run:
	go run ./cmd/api

run-engine:
	go run ./cmd/pet-engine

ping:
	go run ./cmd/pet-engine ping

vapid:
	go run ./cmd/pet-engine vapid

migrate:
	go run ./cmd/migrate

lint:
	gofmt -l .
	go vet ./...

build:
	go build -o bin/ ./cmd/...

web:
	cd web && npm run dev

generate: generate-api generate-sqlc

generate-api:
	go generate ./internal/api/oapi
	cd web && npm run generate:api

generate-sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:1.31.1 generate

deps-update:
	go get -u ./...
	go mod tidy
	cd web && npm update

up:
	docker compose up --build

down:
	docker compose down
