# mcp-server-mock-go

用于联调的 Mock MCP Server（Go 版），使用 `net/http` 实现，提供标准 JSON-RPC 2.0 风格 `POST /mcp` 接口，支持：

- `initialize`
- `tools/list`
- `tools/call`
- 可选 SSE 返回（`Accept: text/event-stream`）

默认端口：`8080`

## 快速启动

### 环境要求

- Go `1.26+`
- Docker + Docker Compose（用于容器部署）

### 本地启动

```bash
go run ./cmd/mcp-server
```

### Docker Compose 启动

```bash
docker compose up --build
```

### 健康验证（initialize）

```bash
curl -sS -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "health-1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06",
      "capabilities": {},
      "clientInfo": {"name": "curl", "version": "0.0.1"}
    }
  }'
```

## curl 联调全流程

```bash
BASE_URL="http://localhost:8080/mcp"
```

### 1) initialize

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

### 2) tools/list

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

### 3) tools/call - 天气工具

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "3",
    "method": "tools/call",
    "params": {
      "name": "mock.weather.query",
      "arguments": {
        "city": "shanghai",
        "date": "2026-02-14"
      }
    }
  }'
```

### 4) tools/call - 物流工具

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "4",
    "method": "tools/call",
    "params": {
      "name": "mock.logistics.status",
      "arguments": {
        "trackingNo": "SF1234567890",
        "carrier": "SF Express"
      }
    }
  }'
```

### 5) tools/call - 敏感信息检测工具

```bash
curl -sS -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "5",
    "method": "tools/call",
    "params": {
      "name": "mock.sensitive-data.detect",
      "arguments": {
        "text": "我的邮箱是 user@example.com，请联系我。"
      }
    }
  }'
```

### 6) SSE 调用

```bash
curl -sS -N -X POST "$BASE_URL" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "7",
    "method": "tools/list",
    "params": {}
  }'
```

## 响应字段说明

典型成功结构：

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

如 method 不存在，则返回：

- `error.code = -32601`
- `error.message = method not found: <method>`

## 内置工具清单

- `mock.weather.query`
- `mock.logistics.status`
- `mock.ops.runbook.generate`
- `mock.sensitive-data.detect`
- `mock.todo.tasks.list`
- `mock.transport.schedule.query`

工具定义来自 `tools/*.yml`，`tools/list` 按 YAML 原样字段输出（包含可选 `afterCallHint`）。

## 配置项（环境变量）

- `SERVER_PORT`（默认 `8080`，`docker-compose` 下仍为 `8080`，仅对外映射端口为 `11969`）
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`（默认 `./tools/*.yml`）
- `MCP_OBSERVABILITY_LOG_ENABLED`（默认 `true`）
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`（默认 `2000`）
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`（默认 `false`）

## 测试

```bash
go test ./...
```

当前测试覆盖：

- initialize 成功响应
- tools/list 返回 6 个 canonical 工具
- tools/call 返回结构化内容
- legacy 工具名返回 unknown tool
- SSE 返回格式
- observability 日志、日志关闭、脱敏、截断
- 工具 YAML 加载异常时回退空列表
