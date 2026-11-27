.PHONY: run run-engine migrate lint build web generate generate-api generate-sqlc deps-update up down

run:
	go run ./cmd/api

run-engine:
	go run ./cmd/pet-engine

# Застосувати міграції до DATABASE_URL (API робить це й сам під час старту).
migrate:
	go run ./cmd/migrate

lint:
	gofmt -l .
	go vet ./...

build:
	go build -o bin/ ./cmd/...

web:
	cd web && npm run dev

# Перегенерувати код з контрактів: OpenAPI -> Go-сервер і TS-клієнт, SQL -> Go (sqlc).
generate: generate-api generate-sqlc

generate-api:
	go generate ./internal/api/oapi
	cd web && npm run generate:api

# sqlc потребує cgo, тому запускається з офіційного Docker-образу.
generate-sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:1.31.1 generate

# Оновити залежності до останніх сумісних версій.
deps-update:
	go get -u ./...
	go mod tidy
	cd web && npm update

up:
	docker compose up --build

down:
	docker compose down
