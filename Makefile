APP_NAME := mcp-server-mock

.PHONY: build run test docker-build docker-up docker-down clean

build:
	go build ./cmd/mcp-server

run:
	set -a; [ ! -f .env ] || . ./.env; set +a; SERVER_PORT="$${HOST_PORT:-$${SERVER_PORT:-8080}}" go run ./cmd/mcp-server

test:
	go test ./...

docker-build:
	docker compose build

docker-up:
	docker compose up --build

docker-down:
	docker compose down

clean:
	rm -f mcp-server
