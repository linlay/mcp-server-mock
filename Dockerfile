FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mcp-server ./cmd/mcp-server

FROM alpine:3.21

WORKDIR /app
RUN adduser -D -u 10001 appuser

COPY --from=builder /out/mcp-server /app/mcp-server
COPY tools /app/tools

ENV SERVER_PORT=19080 \
    MCP_TOOLS_SPEC_LOCATION_PATTERN=./tools/*.yml \
    MCP_OBSERVABILITY_LOG_ENABLED=true \
    MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH=2000 \
    MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS=false

EXPOSE 19080

USER appuser
ENTRYPOINT ["/app/mcp-server"]
