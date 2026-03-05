# CLAUDE.md

## 1. 项目定位

`mcp-server-mock` 是用于联调的 Mock MCP Server，基于 Go `net/http` 实现 `POST /mcp`，输出 JSON-RPC 2.0 风格响应。

核心原则：

- 工具协议定义以 `tools/*.yml` 为唯一来源
- Go 负责：加载 YAML、编译/执行 JSON Schema 校验、执行工具业务逻辑
- YAML 与 handler 不一致时 fail-fast（启动失败）

## 2. 架构说明

### 2.1 分层

- `cmd/mcp-server/main.go`
  - 装配配置、registry、controller 并启动服务
- `internal/mcp/spec`
  - 读取 YAML，做 tool 顶层结构校验
- `internal/mcp/schema`
  - 编译和执行 `inputSchema` 校验
- `internal/mcp/tools`
  - handler 接口、注册表、各工具业务实现
- `internal/mcp/protocol`
  - RPC 请求/响应类型与标准错误码
- `internal/mcp/transport`
  - HTTP 控制器、method 分发、SSE 输出
- `internal/observability`
  - 日志、脱敏、截断

### 2.2 启动链路

1. 读取配置 `config.Load()`
2. `tools.NewRegistry(...)`
   - 加载 `tools/*.yml`
   - 校验顶层定义
   - 编译每个 `inputSchema`
   - 对齐 handler 集合（双向一致）
3. 创建 `transport.Controller`
4. 注册 `/mcp` 并启动 HTTP Server

## 3. API 约束

### 3.1 Endpoint

- URL: `/mcp`
- Method: `POST`
- Content-Type: `application/json`
- 可选 SSE: `Accept: text/event-stream`

### 3.2 支持方法

- `initialize`
- `tools/list`
- `tools/call`

### 3.3 错误语义

- `-32700` parse error
- `-32600` invalid request
- `-32601` method not found
- `-32602` invalid params（`tools/call` 参数 schema 校验失败）
- `-32603` internal error

兼容性约束：unknown tool 仍返回 `result.isError=true`。

## 4. 工具与数据模型

### 4.1 YAML 结构（必须）

- 必填：`type`、`name`、`description`、`inputSchema`
- 可选：`afterCallHint`
- `inputSchema` 使用 JSON Schema 2020-12 语义子集

### 4.2 工具 handler 接口

```go
type ToolHandler interface {
  Name() string
  Call(ctx context.Context, args map[string]any) (map[string]any, error)
}
```

### 4.3 结果结构（保持兼容）

`tools/call` 成功时返回：

- `structuredContent`
- `content`
- `isError=false`

tool 业务报错时返回：

- `isError=true`
- `error`
- `content`

## 5. 配置项

- `SERVER_PORT`（默认 `8080`）
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`（默认 `./tools/*.yml`）
- `MCP_HTTP_MAX_BODY_BYTES`（默认 `1048576`）
- `MCP_OBSERVABILITY_LOG_ENABLED`（默认 `true`）
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`（默认 `2000`）
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`（默认 `false`）

## 6. 开发约束（必须遵守）

1. 新增工具必须同步：
- 新增 `tools/<tool>.yml`
- 新增 handler 并加入 `BuiltinHandlers()`
- 补充 registry/controller 测试

2. 禁止在 Go 里重复维护参数规则：
- 参数校验以 YAML `inputSchema` 为准
- 不允许通过代码默认值绕开必填约束

3. 统一一致性策略：
- YAML 与 handler 不一致直接启动失败

4. 保持 mock 输出稳定：
- 继续使用参数种子驱动幂等随机

5. 文档同步：
- 协议、错误码、配置改动必须更新 `README.md` 与本文件

## 7. 本地验证

运行：

```bash
go run ./cmd/mcp-server
```

测试：

```bash
go test ./...
```

最小回归：

- `initialize` 正常
- `tools/list` 返回 6 工具
- `tools/call` 成功
- `tools/call` 参数错误返回 `-32602`
- SSE 输出格式正常
- observability 脱敏与截断正常
