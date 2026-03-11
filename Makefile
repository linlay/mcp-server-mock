APP_NAME := mcp-server-mock
GOPROXY ?= https://proxy.golang.org,direct
GOSUMDB ?= sum.golang.org
GO_ENV := GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB)
CN_GOPROXY ?= https://mirrors.aliyun.com/goproxy/,direct
CN_GOSUMDB ?= off

.PHONY: build run test docker-build docker-up docker-build-cn docker-up-cn docker-down clean

build:
	$(GO_ENV) go build ./cmd/mcp-server

run:
	set -a; [ ! -f .env ] || . ./.env; set +a; SERVER_PORT="$${HOST_PORT:-$${SERVER_PORT:-8080}}" $(GO_ENV) go run ./cmd/mcp-server

test:
	$(GO_ENV) go test ./...

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-build-cn:
	GOPROXY=$(CN_GOPROXY) GOSUMDB=$(CN_GOSUMDB) docker compose build

docker-up-cn:
	GOPROXY=$(CN_GOPROXY) GOSUMDB=$(CN_GOSUMDB) docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -f mcp-server
