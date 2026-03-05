# mcp-server-mock

用于联调的 Mock MCP Server（Go 版），使用 `net/http` 提供单入口 `POST /mcp`。

## 核心特性

- YAML 单一事实来源：`tools/*.yml` 定义工具元数据与 `inputSchema`
- 启动期校验：加载 YAML、校验 tool 定义、编译 JSON Schema（2020-12 语义子集）
- 调用期校验：`tools/call` 先按 `inputSchema` 校验 `arguments`，再执行 Go handler
- 一致性 fail-fast：YAML 与 Go handler 不一致时直接启动失败
- 支持 `initialize` / `tools/list` / `tools/call`
- 支持 SSE（`Accept: text/event-stream`）

## 快速启动

### 环境要求

- Go `1.26+`
- Docker + Docker Compose（可选）

### 本地启动

```bash
SERVER_PORT=11969 go run ./cmd/mcp-server
```

### Docker Compose 启动

```bash
cp .env.example .env
docker compose up --build
```

## 请求示例

```bash
BASE_URL="http://localhost:8080/mcp"
```

### initialize

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06",
      "capabilities": {},
      "clientInfo": {"name": "runner", "version": "0.0.1"}
    }
  }'
```

### tools/list

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "2",
    "method": "tools/list",
    "params": {}
  }'
```

### tools/call（成功）

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "3",
    "method": "tools/call",
    "params": {
      "name": "mock.weather.query",
      "arguments": {"city": "shanghai", "date": "2026-02-14"}
    }
  }'
```

### tools/call（参数校验失败，-32602）

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "4",
    "method": "tools/call",
    "params": {
      "name": "mock.weather.query",
      "arguments": {"city": "shanghai"}
    }
  }'
```

## 响应规则

- method 不存在：`error.code = -32601`
- 请求体非法：`error.code = -32700/-32600`
- `tools/call` 参数不合法：`error.code = -32602`
- tool 名不存在：返回 `result.isError=true`（保持兼容）

成功结构（tools/call）：

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "result": {
    "structuredContent": {},
    "content": [{"type": "text", "text": "..."}],
    "isError": false
  }
}
```

## 目录结构

```text
cmd/mcp-server/main.go
internal/mcp/protocol/
internal/mcp/spec/
internal/mcp/schema/
internal/mcp/tools/
internal/mcp/transport/
internal/observability/
tools/*.yml
```

## 配置项（环境变量）

- `SERVER_PORT`（默认 `8080`）
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`（默认 `./tools/*.yml`）
- `MCP_HTTP_MAX_BODY_BYTES`（默认 `1048576`）
- `MCP_OBSERVABILITY_LOG_ENABLED`（默认 `true`）
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`（默认 `2000`）
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`（默认 `false`）
- `HOST_PORT`（仅 docker-compose 使用，默认 `11969`）

## 测试

```bash
go test ./...
```

覆盖范围：

- registry 启动期一致性（重复 name、schema 非法、handler/spec 漂移）
- controller 请求分发、SSE、unknown tool、schema 校验失败
- observability 日志开关、脱敏、截断、错误日志
