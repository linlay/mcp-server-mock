# CLAUDE.md

## 1. 项目定位

`mcp-server-mock` 是一个用于联调的 Mock MCP Server，基于 Go `net/http` 实现单一入口 `POST /mcp`，对外提供 JSON-RPC 2.0 风格接口，并支持可选 SSE 响应。

核心目标：

- 提供稳定、可预测的工具调用返回结构，方便前后端/Agent 联调。
- 通过工具 YAML 声明暴露工具元数据（`tools/list`），通过 Go 逻辑返回伪造业务数据（`tools/call`）。
- 提供可控的 observability 日志（可开关、脱敏、截断）。

## 2. 架构说明

### 2.1 模块分层

- `cmd/mcp-server/main.go`
  - 应用入口，组装依赖并启动 HTTP 服务。
- `internal/config`
  - 读取环境变量并提供运行配置。
- `internal/mcp/controller.go`
  - HTTP 层：解析请求、路由 method、封装 RPC 响应、SSE 输出。
- `internal/mcp/tool_spec_repository.go`
  - 从 `tools/*.yml` 加载工具定义并缓存。
- `internal/mcp/tool_service.go`
  - 工具业务逻辑与返回结构组装。
- `internal/observability`
  - 结构化日志与脱敏摘要。
- `tools/*.yml`
  - 工具协议定义（名称、描述、入参 schema、可选 `afterCallHint`）。

### 2.2 启动与请求链路

1. `config.Load()` 读取环境变量。
2. 初始化 `ToolSpecRepository`（启动时加载 YAML）。
3. 初始化 `LogSanitizer` + `Logger`。
4. 初始化 `ToolService` + `Controller`。
5. 注册路由 `mux.Handle("/mcp", controller)`，启动 HTTP server。
6. 每次请求由 `Controller.ServeHTTP` 处理：
   - 限制仅 `POST`。
   - 读取并解析 JSON body。
   - 记录请求日志。
   - 按 `method` 分发到 `initialize` / `tools/list` / `tools/call`。
   - 根据 `Accept: text/event-stream` 决定 JSON 或 SSE 输出。

## 3. API 定义

### 3.1 Endpoint

- URL: `/mcp`
- Method: `POST`
- Content-Type: `application/json`
- 可选 SSE：`Accept: text/event-stream`

非 `POST` 返回 `405`。
JSON 非法返回 `400 invalid json`。

### 3.2 通用请求格式（JSON-RPC 风格）

```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "method": "tools/call",
  "params": {}
}
```

说明：

- 当前实现不会强制校验 `jsonrpc` 字段值。
- `id` 原样回传；缺失时回包 `id` 为 `null`。

### 3.3 支持的方法

1. `initialize`
- 返回协议版本、服务信息、能力声明。

2. `tools/list`
- 返回 YAML 加载到的工具列表（字段基本按 YAML 原样输出）。

3. `tools/call`
- 入参：
  - `params.name`：工具名（严格 canonical 名称）
  - `params.arguments`：对象参数
- 成功返回：
  - `result.structuredContent`：结构化对象
  - `result.content`：文本数组（首项为 `structuredContent` 的 JSON 字符串）
  - `result.isError=false`
- 失败返回（未知工具等）：
  - `result.isError=true`
  - `result.error` + `result.content`

4. 未知方法
- 返回 JSON-RPC 错误：
  - `error.code = -32601`
  - `error.message = method not found: <method>`

### 3.4 响应格式

成功：

```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "result": {}
}
```

失败：

```json
{
  "jsonrpc": "2.0",
  "id": "req-1",
  "error": {
    "code": -32601,
    "message": "method not found: xxx"
  }
}
```

SSE 模式下，响应体为单条事件：

```text
data: <json>

```

## 4. 内置工具与数据结构

### 4.1 工具清单（canonical）

- `mock.weather.query`
- `mock.logistics.status`
- `mock.ops.runbook.generate`
- `mock.sensitive-data.detect`
- `mock.todo.tasks.list`
- `mock.transport.schedule.query`

`ToolService` 对工具名做严格校验，不支持 legacy alias（例如 `mock_city_weather` 会报 unknown tool）。

### 4.2 工具定义 YAML 结构

每个 `tools/*.yml` 至少包含：

- `type`（非空）
- `name`（非空）
- `description`（非空）
- `inputSchema`（对象）

可选字段：

- `afterCallHint`

仓库加载策略：

- 只要出现任意 YAML 解析/校验错误或重复 `name`，`tools/list` 将回退为空列表（fail-closed）。

### 4.3 `tools/call` 结果通用结构

```json
{
  "structuredContent": {},
  "content": [{"type": "text", "text": "..."}],
  "isError": false
}
```

### 4.4 各工具 `structuredContent` 字段

1. `mock.weather.query`
- 入参：`city`, `date`
- 出参：`city`, `date`, `temperatureC`, `humidity`, `windLevel`, `condition`, `mockTag`

2. `mock.logistics.status`
- 入参：`trackingNo`（必填）, `carrier`
- 出参：`trackingNo`, `carrier`, `status`, `currentNode`, `etaDate`, `updatedAt`, `mockTag`

3. `mock.ops.runbook.generate`
- 入参：`message` / `query`, `city`
- 出参：`message`, `city`, `riskLevel`, `recommendedCommand`, `steps`, `mockTag`

4. `mock.sensitive-data.detect`
- 入参：`text|content|message|query|document|input`（按顺序取第一个非空）
- 出参：`hasSensitiveData`, `result`, `description`

5. `mock.todo.tasks.list`
- 入参：`owner`
- 出参：`owner`, `total`, `tasks[]`, `mockTag`
- `tasks[i]`：`id`, `title`, `priority`, `status`, `dueDate`

6. `mock.transport.schedule.query`
- 入参：`type`, `fromCity`, `toCity`, `date`
- 出参：`travelType`, `number`, `fromCity`, `toCity`, `date`, `departureTime`, `arrivalTime`, `status`, `gateOrPlatform`, `mockTag`

### 4.5 幂等随机机制

- 所有 mock 数据基于 `arguments` 计算稳定种子（按 key 排序后序列化），再通过 Java 风格 PRNG 生成。
- 同一组参数重复调用，输出稳定；参数变化，输出随之变化。

## 5. 配置项

环境变量：

- `HOST_PORT`（仅 `docker-compose` 使用，默认 `11969`）
- `SERVER_PORT`（默认 `8080`）
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`（默认 `./tools/*.yml`；应用通用变量）
- `MCP_OBSERVABILITY_LOG_ENABLED`（默认 `true`）
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`（默认 `2000`）
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`（默认 `false`）

容器注意：

- `Dockerfile` 默认 `SERVER_PORT=19080` 且 `EXPOSE 19080`。
- `docker-compose.yml` 通过 `env_file: .env` 读取运行参数，并使用 `${HOST_PORT}:${SERVER_PORT}` 做端口映射（含默认值 `11969:8080`）。
- `docker-compose.yml` 中 `MCP_TOOLS_SPEC_LOCATION_PATTERN` 固定为 `./tools/*.yml`，不从 `.env` 读取。

## 6. Observability 设计

日志事件（关键）：

- `event=mcp.request`
- `event=mcp.response`
- `event=mcp.error`
- `event=tool.call.request`
- `event=tool.call.response`
- `event=tool.call.error`

脱敏与截断：

- 按关键字掩码（如 `password/token/apikey/authorization/secret` 等）为 `***`。
- 对超长对象/数组仅保留预览摘要。
- 最终日志字符串按 `LogMaxBodyLength` 截断（最小 80 字符）。

## 7. 开发要点（必须遵守）

1. 新增工具时必须同步 3 处
- 新增 `tools/<name>.yml`
- 在 `tool_service.go` 增加 canonical 常量、名称白名单、switch 实现
- 补齐测试（至少 `tools/list` + `tools/call`）

2. 保持协议兼容
- `tools/call` 返回必须保持 `structuredContent + content + isError` 三段结构。
- 未知工具建议保持 `isError=true` 的 result，而不是 HTTP 非 200。

3. 谨慎修改 YAML 加载策略
- 当前是 fail-closed：任意错误导致空工具列表。若改为部分成功，需要同步测试和文档。

4. 保持 mock 数据“稳定可复现”
- 依赖参数种子而非当前时间，避免联调结果漂移。

5. 变更日志与配置时同步更新文档
- 至少更新 `README.md` 与本文件，确保 API/字段说明一致。

## 8. 本地开发与验证

运行：

```bash
go run ./cmd/mcp-server
```

测试：

```bash
go test ./...
```

建议最小回归用例：

- `initialize` 正常回包
- `tools/list` 返回 6 个 canonical 工具
- `tools/call` 成功结构
- unknown tool 返回 `isError=true`
- SSE 输出格式
- observability 脱敏与截断
