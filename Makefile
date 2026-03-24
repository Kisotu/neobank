APP_NAME=api
BIN_DIR=bin

.PHONY: build test migrate-up migrate-down sqlc lint run

build:
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/api

test:
	go test ./...

migrate-up:
	migrate -path db/migrations -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path db/migrations -database "$${DATABASE_URL}" down 1

sqlc:
	sqlc generate

lint:
	golangci-lint run ./...

run:
	go run ./cmd/api