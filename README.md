# mcp-server-mock

## 项目简介

`mcp-server-mock` 是一个用于联调和回归测试的 Mock MCP Server，使用 Go `net/http` 提供单入口 `POST /mcp`，支持 `initialize`、`tools/list`、`tools/call` 和可选 SSE 输出。

项目目标：

- 提供稳定、可重复的 mock 工具响应
- 在启动阶段校验 `tools/*.yml` 与 Go handler 一致性
- 在调用阶段按 tool schema 校验参数

详细协议契约见 `docs/mcp-protocol-definition.md`。

## 快速开始

环境要求：

- Go `1.26+`
- Docker 与 Docker Compose（仅在容器方式运行时需要）

首次使用：

```bash
cp .env.example .env
```

如在中国大陆或阿里云网络环境构建，默认建议保留 `.env` 中的 Go 模块代理配置：

```bash
make docker-up-cn
```

公共默认构建保持 Go 官方源：

```bash
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
```

本地运行：

```bash
make run
```

运行测试：

```bash
make test
```

Docker Compose 启动：

```bash
make docker-up
```

如在阿里云或中国大陆网络环境构建：

```bash
make docker-up-cn
```

启动后可通过以下地址访问：

```bash
BASE_URL="http://localhost:11969/mcp"
```

`make run` 会读取 `.env`，并优先使用 `HOST_PORT` 作为本地监听端口；如果直接执行 `go run ./cmd/mcp-server`，则仍按 `SERVER_PORT` 或内嵌默认值运行。

## 配置说明

默认使用方式：

- 小配置场景只需要 `.env`
- `.env` 必须本地保存，不提交到仓库
- `.env.example` 仅提供键名、默认值/占位和简述

配置层级：

1. 代码默认值
2. 内嵌默认配置 `internal/config/application.yml`
3. 可选外部 yml（仅当设置 `CONFIG_PATH` 时启用）
4. 环境变量

常用变量：

- `HOST_PORT`：`make run` 默认优先使用的本地监听端口，同时也是 Docker Compose 暴露到宿主机的端口
- `CONFIG_PATH`：可选的外部结构化配置文件路径
- `SERVER_PORT`：服务监听端口
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`：tool spec 文件匹配路径
- `MCP_VIEWPORTS_DIR`：viewport 文件目录
- `MCP_HTTP_MAX_BODY_BYTES`：HTTP 请求体大小上限
- `MCP_OBSERVABILITY_LOG_ENABLED`：是否开启日志
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`：日志中请求体截断长度
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`：是否记录请求头

复杂配置场景：

- 当 `.env` 不足以表达结构化配置时，再新增 `configs/*.yml`
- 在 `.env` 中设置 `CONFIG_PATH=./configs/config.dev.yml`
- 外部 yml 只写需要覆写的字段，不复制全部默认配置
- 生产敏感项通过环境变量或 Secret 注入，并覆盖 yml 中同名配置

## 部署

本项目支持两种常见部署方式。

直接运行二进制：

```bash
make build
./mcp-server
```

Docker 镜像：

```bash
make docker-build
docker run --rm -p 8080:8080 --env-file .env mcp-server-mock
```

部署约束：

- 不将 `.env` 打包进镜像
- 不把真实密钥写入 `Dockerfile`
- `internal/config/application.yml` 是内嵌默认配置，不作为部署后外部可编辑配置文件
- 如需复杂环境差异，可通过 `CONFIG_PATH` 指向容器内挂载的外部 yml，再用环境变量覆盖敏感项

## 运维

健康验证：

- 直接向 `/mcp` 发送 `initialize` 或 `tools/list` 请求，确认返回 `200` 和 JSON-RPC 结果

示例请求：

```bash
curl -sS -X POST "http://localhost:11969/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "tools/list",
    "params": {}
  }'
```

常见排查：

- 启动失败并提示配置错误：检查 `.env` 中数值/布尔格式是否正确，或 `CONFIG_PATH` 指向的 yml 是否存在非法字段
- 启动失败并提示 registry 错误：检查 `tools/*.yml` 与 `BuiltinHandlers()` 是否一一对应
- 启动失败并提示 `read viewports dir`：检查镜像/运行目录中是否包含 `viewports/`，或通过 `MCP_VIEWPORTS_DIR` 指向正确目录
- 请求返回 `-32602`：检查 `tools/call.arguments` 是否满足对应 tool 的 `inputSchema`
- Docker 启动后无法访问：确认 `HOST_PORT` 和 `SERVER_PORT` 映射是否正确
- Docker 构建阶段 `go mod download` 超时：公共环境保持默认官方源；阿里云国内网络建议使用 `make docker-build-cn` 或 `make docker-up-cn`

`tools/list` 扩展字段：

- `label` 表示工具的人类可读名称，适合直接给前端显示中文名
- `toolAction: true` 表示该工具应被消费方视为 action 工具
- `toolType + viewportKey` 成对出现时表示该工具应被消费方视为 frontend 工具
- 未声明上述字段时，消费方可按 backend 工具处理
- MCP 客户端应优先显示 `label`，回退到 `name`
- 发起 `tools/call` 时始终使用 `name`，不要使用 `label`

日志说明：

- 默认开启 observability 日志
- 可通过 `MCP_OBSERVABILITY_LOG_ENABLED`、`MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`、`MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS` 调整日志行为

## 附录：请求示例

`initialize`：

```bash
curl -sS -X POST "http://localhost:11969/mcp" \
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

`tools/call`：

```bash
curl -sS -X POST "http://localhost:11969/mcp" \
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
